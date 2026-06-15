package main

import (
	"bufio"
	"crypto/tls"
	"errors"
	"fmt"
	"net"
	"strings"
	"time"

	"github.com/go-irc/irc"
)

func main() {
	client := NewIRCClient("127.0.0.1", "Shinobu-Kocho-fan-3", 6668)
	client.Connect()
	go func() {
		for msg := range client.Incoming {
			fmt.Println(msg)
			if strings.Contains(msg, "PING") {
				pong(msg, client.connection)
			}
		}
	}()
	select {}
}

// So what are the fundamentals in creating an IRC Client?

// Nickname, Host, Port, Channels, Messages, establishing a connection

var commandMap map[string]string = map[string]string{
	"JOIN": "Joined channel",
	"PART": "Left channel",
}

type IRCClient struct {
	Host       string
	Nickname   string
	Port       int
	connection net.Conn
	Channels   []string // Keep some memory of every channel we have `joined`
	Incoming   chan StructuredMessage
	TLS        bool
	Quit       chan struct{}
}

type StructuredMessage interface {
	RawToReadable(msg *irc.Message) StructuredMessage
	Formatted() string
}

type ChannelMessage struct {
	Timestamp time.Time
	User      string
	Message   string
	Channel   string
}

type CommandMessage struct {
	Timestamp time.Time
	User      string
	Reason    string
	Channel   string
	Command   string
}

func NewIRCClient(host, nickname string, port int, tls bool) *IRCClient {
	return &IRCClient{
		Host:     host,
		Nickname: nickname,
		Port:     port,
		TLS:      tls,
		Incoming: make(chan StructuredMessage, 32),
		Quit:     make(chan struct{}),
	}
}

func (c *IRCClient) ReadMessages() tea.Cmd {
	return func() tea.Msg {
		msg, ok := <-c.Incoming
		if !ok {
			return errMsg(errors.New("connection closed"))
		}
		return incomingMsg(msg)
	}
}

func (c *IRCClient) Connect() {
	var conn net.Conn
	var err error
	address := fmt.Sprintf("%s:%d", c.Host, c.Port)

	if c.TLS {
		conn, err = tls.Dial("tcp", address, nil)

	} else {
		conn, err = net.Dial("tcp", address)
	}

	if err != nil {
		panic(err)
	}

	c.connection = conn
	// need to send the nick and user commands to the server
	fmt.Fprintf(c.connection, "NICK %s\r\n", c.Nickname)
	fmt.Fprintf(c.connection, "USER %s 0 * :%s\r\n", c.Nickname, c.Nickname)
	go c.readLoop(c.connection)

}

func (c *IRCClient) Disconnect() {
	close(c.Quit)
	if c.connection != nil {
		_ = c.connection.Close()
	}
}

func (c *IRCClient) Send(msg string) {
	//fmt.Fprintf(c.connection, "%s\r\n", msg)
	c.ParseUserInput(msg)
	//fmt.Println("Sent message:", msg)
}

func (c *IRCClient) ParseUserInput(input string) {
	if !strings.HasPrefix(input, "/") {
		currentChannel := c.Channels[len(c.Channels)-1] // Send to the most recently joined channel
		fmt.Fprintf(c.connection, "PRIVMSG %s :%s\r\n", currentChannel, input)
	}

	parts := strings.SplitN(input, " ", 3)
	command := strings.ToUpper(parts[0])

	switch command {
	case "/JOIN":
		if len(parts) > 1 {
			c.Channels = append(c.Channels, parts[1])
			fmt.Fprintf(c.connection, "JOIN %s\r\n", parts[1])
		}
	case "/MSG":
		if len(parts) > 2 {
			fmt.Fprintf(c.connection, "PRIVMSG %s :%s\r\n", parts[1], parts[2])
		}
	case "/CHANNELS":
		chans := fmt.Sprintf("Joined channels: %s", strings.Join(c.Channels, ", "))
		msg := ChannelMessage{Timestamp: time.Now(), User: "client", Message: chans, Channel: "system"}
		c.Incoming <- &msg
	}
}

func (cm *ChannelMessage) RawToReadable(msg *irc.Message) StructuredMessage {

	if len(msg.Params) >= 2 {
		return &ChannelMessage{
			Timestamp: time.Now(),
			User:      msg.Prefix.Name,
			Message:   msg.Params[1],
			Channel:   msg.Params[0],
		}
	}

	return &ChannelMessage{
		Timestamp: time.Now(),
		User:      msg.Prefix.Name,
		Message:   msg.String(),
		Channel:   msg.Params[0],
	}
}

func (cm *ChannelMessage) Formatted() string {
	return fmt.Sprintf("[%s] {%s} <%s> : %s", cm.Timestamp.Format("15:04"), cm.Channel, cm.User, cm.Message)
}

func (cm *CommandMessage) RawToReadable(msg *irc.Message) StructuredMessage {
	return &CommandMessage{
		Timestamp: time.Now(),
		User:      msg.Prefix.Name,
		Command:   msg.Command,
		Channel:   msg.Params[0],
	}
}

func (cm *CommandMessage) Formatted() string {
	if cm.User != "" {
		return fmt.Sprintf("[%s] # <%s> %s", cm.Timestamp.Format("15:04"), cm.User, commandMap[cm.Command])
	}
	return fmt.Sprintf("[%s] : {%s} %s", cm.Timestamp.Format("15:04"), cm.Channel, cm.Command)
}

func handleCommands(conn net.Conn, msg *irc.Message, line string, incoming chan StructuredMessage) bool {

	switch msg.Command {
	case "PING":
		{
			pong(line, conn)
			return true
		}
	case "PRIVMSG":
		{
			if len(msg.Params) > 1 {
				text := msg.Params[1]
				if strings.HasPrefix(text, "\x01VERSION") {
					target := msg.Name
					if target == "" && msg.Prefix != nil {
						target = msg.Prefix.Name
					}

					if target != "" {
						// Respond with a NOTICE wrapped in \x01
						fmt.Fprintf(conn, "NOTICE %s :\x01VERSION CustomGoClient:1.0\x01\r\n", target)
					}
					return true
				}
			}
		}
	case "JOIN", "PART":
		{
			incoming <- (&CommandMessage{}).RawToReadable(msg)
			return true
		}
		//case "QUIT":
		//	{
		//	}
		//	return true
	}

	return false
}

func pong(msg string, conn net.Conn) {
	token := strings.Split(msg, " ")[1]
	fmt.Fprintf(conn, "PONG %s\r\n", token)
	fmt.Printf("PONG sent: %s\r\n", token)
}

func (c *IRCClient) readLoop(conn net.Conn) {
	// From the connection, read line by line and print it out to the user:
	scanner := bufio.NewScanner(conn)

	for scanner.Scan() {
		line := scanner.Text()

		msg, err := irc.ParseMessage(line)

		if err != nil {
			fmt.Println("Error parsing message:", err)
			continue
		}

		if handleCommands(conn, msg, line, c.Incoming) {
			continue
		}

		neatMsg := (&ChannelMessage{}).RawToReadable(msg)
		c.Incoming <- neatMsg
	}

	if err := scanner.Err(); err != nil {
		fmt.Println("Error reading from connection:", err)
	}
	close(c.Incoming)
}

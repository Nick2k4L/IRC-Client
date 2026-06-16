package client

import (
	"bufio"
	"crypto/tls"
	"errors"
	"fmt"
	"net"
	"strings"
	"time"

	"github.com/Nick2k4L/IRC-Client/helpers"
	"github.com/Nick2k4L/IRC-Client/servercmds"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/go-irc/irc"
)

type IncomingMsg helpers.StructuredMessage

type ErrMsg error

type IRCClient struct {
	Host       string
	Nickname   string
	Port       int
	Connection net.Conn
	Channels   []string // Keep some memory of every channel we have `joined`
	Incoming   chan helpers.StructuredMessage
	TLS        bool
	Quit       chan struct{}
}

func NewIRCClient(host, nickname string, port int, tls bool) *IRCClient {
	return &IRCClient{
		Host:     host,
		Nickname: nickname,
		Port:     port,
		TLS:      tls,
		Incoming: make(chan helpers.StructuredMessage, 32),
		Quit:     make(chan struct{}),
	}
}

func (c *IRCClient) ReadMessages() tea.Cmd {
	return func() tea.Msg {
		msg, ok := <-c.Incoming
		if !ok {
			return ErrMsg(errors.New("connection closed"))
		}
		return IncomingMsg(msg)
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

	c.Connection = conn
	// need to send the nick and user commands to the server
	fmt.Fprintf(c.Connection, "NICK %s\r\n", c.Nickname)
	fmt.Fprintf(c.Connection, "USER %s 0 * :%s\r\n", c.Nickname, c.Nickname)
	go c.readLoop(c.Connection)

}

func (c *IRCClient) Disconnect() {
	close(c.Quit)
	if c.Connection != nil {
		_ = c.Connection.Close()
	}
}

func (c *IRCClient) Send(msg string) {
	//fmt.Fprintf(c.connection, "%s\r\n", msg)
	c.ParseUserInput(msg)
	//fmt.Println("Sent message:", msg)
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

		if servercmds.HandleCommands(conn, msg, line, c.Incoming) {
			continue
		}

		neatMsg := (&helpers.ChannelMessage{}).RawToReadable(msg)
		c.Incoming <- neatMsg
	}

	if err := scanner.Err(); err != nil {
		fmt.Println("Error reading from connection:", err)
	}
	close(c.Incoming)
}

func (c *IRCClient) ParseUserInput(input string) {

	// TODO: Change this to an error message command
	if !strings.HasPrefix(input, "/") {
		if len(c.Channels) == 0 {
			msg := helpers.CommandMessage{Timestamp: time.Now(), Channel: "Client", Command: "ERROR: you need to specify at least one channel. Use /join <channel> to join a channel"}
			c.Incoming <- &msg
			return
		}
		currentChannel := c.Channels[len(c.Channels)-1] // Send to the most recently joined channel
		fmt.Fprintf(c.Connection, "PRIVMSG %s :%s\r\n", currentChannel, input)
	}

	parts := strings.SplitN(input, " ", 3)
	command := strings.ToUpper(parts[0])

	switch command {
	case "/JOIN":
		if len(parts) > 1 {
			c.Channels = append(c.Channels, parts[1])
			fmt.Fprintf(c.Connection, "JOIN %s\r\n", strings.TrimSpace(parts[1]))
		}
	case "/MSG":
		if len(parts) > 2 {
			fmt.Fprintf(c.Connection, "PRIVMSG %s :%s\r\n", parts[1], parts[2])

			// TODO: Change to a DM message type
			dmMsg := &helpers.ChannelMessage{
				Timestamp: time.Now(),
				User:      c.Nickname,      // It's from you
				Message:   parts[2],        // The message content
				Channel:   "->" + parts[1], // Visual indicator that this is an outbound DM
			}
			c.Incoming <- dmMsg
		}

	case "/PART":
		// Logic for parting
		if len(parts) > 1 {
			fmt.Fprintf(c.Connection, "PART %s\r\n", strings.TrimSpace(parts[1]))
			c.Channels = c.Channels[:len(c.Channels)-1]
		}
	case "/NICK":
		if len(parts) > 1 {
			fmt.Fprintf(c.Connection, "NICK %s\r\n", strings.TrimSpace(parts[1]))
			c.Nickname = parts[1] // Update the nickname in the client state?
		}

	case "/ME":
		if len(parts) > 1 {
			fmt.Fprintf(c.Connection, "PRIVMSG %s :\x01ACTION %s\x01\r\n", c.Channels[len(c.Channels)-1], strings.TrimSpace(parts[2]))
		}

	case "/CHANNELS":
		chans := fmt.Sprintf("Joined channels: %s", strings.Join(c.Channels, ", "))
		msg := helpers.ChannelMessage{Timestamp: time.Now(), User: "client", Message: chans, Channel: "system"}
		c.Incoming <- &msg
	}
}

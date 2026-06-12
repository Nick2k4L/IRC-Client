package main

import (
	"fmt"
	"net"
	"strings"

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

type IRCClient struct {
	Host       string
	Nickname   string
	Port       int
	connection net.Conn
	Incoming   chan string
	Quit       chan struct{}
}

func NewIRCClient(host, nickname string, port int) *IRCClient {
	return &IRCClient{
		Host:     host,
		Nickname: nickname,
		Port:     port,
		Incoming: make(chan string, 32),
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
	conn, err := net.Dial("tcp", fmt.Sprintf("%s:%d", c.Host, c.Port))
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
	fmt.Fprintf(c.connection, "%s\r\n", msg)
	//fmt.Println("Sent message:", msg)
}

func pong(msg string, conn net.Conn) {
	token := strings.Split(msg, " ")[1]
	fmt.Fprintf(conn, "PONG %s\r\n", token)
	fmt.Printf("PONG sent: %s\r\n", token)
}

func (c *IRCClient) readLoop(conn net.Conn) {
	// From the connection, read line by line and print it out to the user:
	buf := make([]byte, 1024)
	for {
		n, _ := conn.Read(buf)
		if n > 0 {
			msg, _ := irc.ParseMessage(string(buf[:n]))
			if msg.Command == "PING" {
				pong(string(buf[:n]), conn)
			}
			c.Incoming <- msg.String()

		}
	}
}

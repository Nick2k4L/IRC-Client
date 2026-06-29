package client

import (
	"bufio"
	"crypto/tls"
	"errors"
	"fmt"
	"net"

	"github.com/Nick2k4L/IRC-Client/helpers"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/go-irc/irc"
)

type IncomingMsg helpers.StructuredMessage

type ErrMsg error

type IRCClient struct {
	Host            string
	Nickname        string
	LastCommand     string // Keep track of the last command we sent to the server, so we can handle responses to it better
	CurrentChannel  string
	ServerID        string
	Channels        []string // Keep some memory of every channel we have `joined`
	DirectMsgs      []string // Keep some memory of every channel we have `dmed`
	PreJoinChannels []string
	TLS             bool
	IsDev           bool
	Incoming        chan helpers.StructuredMessage
	Quit            chan struct{}
	Port            uint16
	Connection      net.Conn
}

func NewIRCClient(host, nickname string, port uint16, isDev, tls bool) *IRCClient {
	serverID := fmt.Sprintf("%s:%d", host, port)

	return &IRCClient{
		Host:     host,
		Nickname: nickname,
		Port:     port,
		TLS:      tls,
		Incoming: make(chan helpers.StructuredMessage, 512),
		Quit:     make(chan struct{}),
		IsDev:    isDev,
		ServerID: serverID,
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

func (c *IRCClient) ReadStructuredMessage() helpers.StructuredMessage {
	msg, ok := <-c.Incoming
	if !ok {
		return nil
	}
	return msg
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

func (c *IRCClient) Send(target, msg string) {
	c.ParseUserInput(target, msg)
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

		if c.HandleNumeric(msg) {
			continue
		}

		if c.HandleCommands(msg, line) {
			continue
		}

		//c.Incoming <- helpers.ParseRawMessages(msg)
	}

	if err := scanner.Err(); err != nil {
		fmt.Println("Error reading from connection:", err)
	}
	close(c.Incoming)
}

// SetCurrentChannel sets the current channel for the IRC client. Most likely set this via POST
func (c *IRCClient) SetCurrentChannel(channel string) {
	c.CurrentChannel = channel
}

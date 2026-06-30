package client

import (
	"bufio"
	"crypto/tls"
	"errors"
	"fmt"
	"net"
	"sync"

	"github.com/Nick2k4L/IRC-Client/helpers"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/go-irc/irc"
	"github.com/google/uuid"
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
	quitOnce        sync.Once
}

func NewIRCClient(host, nickname string, port uint16, isDev, tls bool) *IRCClient {
	serverID := uuid.New().String()

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

func (c *IRCClient) ReadStructuredMessage() (helpers.StructuredMessage, error) {
	msg, ok := <-c.Incoming
	if !ok {
		c.Disconnect("connection closed")
		return nil, errors.New("connection closed")
	}
	return msg, nil
}

func (c *IRCClient) Connect() error {
	var conn net.Conn
	var err error
	address := fmt.Sprintf("%s:%d", c.Host, c.Port)

	if c.TLS {
		conn, err = tls.Dial("tcp", address, nil)

	} else {
		conn, err = net.Dial("tcp", address)
	}

	if err != nil {
		return err
	}

	// init these variables on a new connection
	c.Connection = conn
	c.Incoming = make(chan helpers.StructuredMessage, 512)
	c.quitOnce = sync.Once{}
	c.Quit = make(chan struct{})

	// need to send the nick and user commands to the server
	fmt.Fprintf(c.Connection, "NICK %s\r\n", c.Nickname)
	fmt.Fprintf(c.Connection, "USER %s 0 * :%s\r\n", c.Nickname, c.Nickname)
	go c.readLoop(c.Connection)
	return nil
}

func (c *IRCClient) Disconnect(reason string) {
	c.quitOnce.Do(func() {
		close(c.Quit)
		if c.Connection != nil {
			fmt.Fprintf(c.Connection, "QUIT :%s\r\n", reason)
			_ = c.Connection.Close()
		}
	})
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

package main

import (
	"fmt"
	"net"
)

func main() {
	fmt.Println("Hello, World!")
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
		Incoming: make(chan string),
		Quit:     make(chan struct{}),
	}
}

func (c *IRCClient) Connect() {
	conn, err := net.Dial("tcp", fmt.Sprintf("%s:%d", c.Host, c.Port))
	if err != nil {
		panic(err)
	}
	c.connection = conn

	defer c.connection.Close()

	// need to send the nick and user commands to the server
	fmt.Fprintf(c.connection, "NICK %s\r\n", c.Nickname)
	fmt.Fprintf(c.connection, "USER %s 0 * :%s\r\n", c.Nickname, c.Nickname)
}

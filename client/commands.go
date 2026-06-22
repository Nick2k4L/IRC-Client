package client

import (
	"fmt"
	"net"
	"strings"
	"time"

	"github.com/Nick2k4L/IRC-Client/helpers"
	"github.com/go-irc/irc"
)

func pong(msg string, conn net.Conn) {
	token := strings.Split(msg, " ")[1]
	fmt.Fprintf(conn, "PONG %s\r\n", token)
}

var UserList = helpers.UserListMessage{
	UserList: make(map[string][]string),
}

// NUMERIC SERVER COMMANDS

func (c *IRCClient) HandleNumeric(msg *irc.Message) bool {

	switch msg.Command {
	case "001", "002", "003", "004", "005":
		{
			c.Incoming <- helpers.ParseServerMessage(msg)
			return true
		}
	case "251", "252", "253", "254", "255", "265", "266", "250", "396":
		{
			// We don't really care about these raw stats for the UI right now.
			// Return true so they don't get passed to the RawMessage fallback.
			return true
		}
	case "311", "312", "317", "319", "344", "671":
		{
			// These are WHOIS responses, we can display them to the user
			// this will return a server message...
			c.Incoming <- helpers.ParseWhoIsMessage(msg)
			return true
		}
	case "331", "332", "333", "TOPIC":
		{
			if msg.Command == "331" {
				// 331 is "No topic is set", handle or skip
				return true
			}
			// This is the topic of a channel, we can display it to the user
			c.Incoming <- helpers.ParseTopicMessage(msg)
			return true
		}
	case "318":
		{
			// end of who is list
			return true
		}
	// displayed per buffer
	case "353":
		{
			helpers.ParseUserListMessage(msg, UserList)
			return true
		}
	// displayed per buffer
	case "366":
		{
			if c.LastCommand == "/NAMES" {
				c.Incoming <- helpers.DeepCopyUserListMessage(UserList) // Send a copy of the user
				return true
			}
			delete(UserList.UserList, helpers.LastChannel) // Clear the user list for the channel after sending it
		}
	case "372", "375":
		{
			// MOTD messages, we can display them to the user
			// TODO: Think about making it a MOTDMessage
			c.Incoming <- helpers.ParseServerMessage(msg)
			return true
		}
	case "376":
		{
			c.Incoming <- &helpers.ServerMessage{Timestamp: time.Now(), Message: "<--- Connection Established --->"}
			// TODO: Can add logic here to connect to multiple channels.
			return true
		}
	case "433":
		{
			// Generate a message to the user saying, use /nick to change nick name
			c.Incoming <- &helpers.ErrorMessage{Timestamp: time.Now(), Message: "Nickname already in use. Use /nick to change nick name."}
			return true
		}

	}

	return false
}

// SERVER COMMANDS

func (c *IRCClient) HandleCommands(msg *irc.Message, line string) bool {
	switch msg.Command {
	case "PING":
		{
			pong(line, c.Connection)
			return true
		}
	// TODO: clean this up!
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
						fmt.Fprintf(c.Connection, "NOTICE %s :\x01VERSION CustomGoClient:1.0\x01\r\n", target)
					}
					return true
				}

				// THINK ABOUT MAKING AN ACTION MESSAGE?
				if strings.HasPrefix(text, "\x01ACTION ") {
					// Strip the \x01ACTION and the trailing \x01
					actionText := strings.TrimPrefix(text, "\x01ACTION ")
					actionText = strings.TrimSuffix(actionText, "\x01")

					c.Incoming <- &helpers.ChannelMessage{Timestamp: time.Now(), User: msg.Prefix.Name, Message: fmt.Sprintf("✧ %s ✧", actionText), Channel: msg.Params[0]}
					return true
				}

				// else, it is most likely a normal message, but we need to handle DMS
				if strings.HasPrefix(msg.Params[0], "#") {
					c.Incoming <- helpers.ParseChannelMessages(msg)
					return true // this is most likely a channel message, let the channel message handler handle this
				}

				if strings.Contains(c.Nickname, msg.Prefix.Name) && strings.Contains(c.Nickname, msg.Params[0]) {
					return true // do nothing here, it is a self DM - we already handle out messages
				}

				// if the message even contains my name within a PRIVMSG it is most likely a DM
				if strings.Contains(c.Nickname, msg.Params[0]) {
					c.Incoming <- helpers.ParseDirectMessage(msg)
					return true
				}

			}
		}

	case "JOIN", "PART":
		{
			c.Incoming <- helpers.ParseCommandMessages(msg)
			return true
		}
	case "NICK":
		{
			if msg.Prefix.Name == c.Nickname {
				c.Nickname = msg.Params[0]
			}
			c.Incoming <- helpers.ParseCommandMessages(msg)
			return true
		}
	case "MODE":
		{
			c.Incoming <- helpers.ParseCommandMessages(msg)
			return true
		}
	case "NOTICE":
		{
			c.Incoming <- helpers.ParseCommandMessages(msg)
			return true
		}
		// channel, user who got kicked, admin who kicked, and a reason
	case "KICK":
		{
			c.Incoming <- helpers.ParseCommandMessages(msg)
			return true
		}
		// need to handle invites from a user...
		// might need to be its own type of message since we do have `extra logic` attached to it.
	case "INVITE":
		{
			c.Incoming <- helpers.ParseCommandMessages(msg)
			return true
		}
	case "ERROR":
		{
			c.Incoming <- &helpers.ErrorMessage{Timestamp: time.Now(), Message: "Connection error. Please check your connection and try again."}
			return true
		}
	case "QUIT":
		{
			// display of quit is not really necessary.
			return true
		}
	}

	return false
}

// USER INPUT / COMMANDS

func (c *IRCClient) ParseUserInput(input string) {

	if !strings.HasPrefix(input, "/") {
		if len(c.Channels) == 0 {
			c.Incoming <- &helpers.ErrorMessage{Timestamp: time.Now(), Message: "You need to join at least one channel. Use /join <channel> to join a channel"}
			return
		}

		// TODO: This will eventually need to be changed to allow the user to specify which channel
		//  they want to send the message to, but for now we will just send it to the most recently joined channel.
		//	 possible solution: add a channel selector to the UI and use index of the selected channel
		//	 to determine which channel to send the message to.

		currentChannel := c.Channels[len(c.Channels)-1] // Send to the most recently joined channel
		fmt.Fprintf(c.Connection, "PRIVMSG %s :%s\r\n", currentChannel, input)
	}

	parts := strings.SplitN(input, " ", 3)
	command := strings.ToUpper(parts[0])
	c.LastCommand = command

	switch command {
	case "/JOIN":
		{
			if len(parts) > 1 {
				c.Channels = append(c.Channels, parts[1])
				fmt.Fprintf(c.Connection, "JOIN %s\r\n", strings.TrimSpace(parts[1]))
			}
		}
	case "/MSG":
		{
			if len(parts) > 2 {
				fmt.Fprintf(c.Connection, "PRIVMSG %s :%s\r\n", parts[1], parts[2])
				c.DirectMsgs = append(c.DirectMsgs, parts[2])

				dmMsg := &helpers.DirectMessage{
					Timestamp: time.Now(),
					Sender:    c.Nickname,
					Receiver:  parts[1],
					Message:   parts[2],
				}
				c.Incoming <- dmMsg
			}
		}

	case "/PART":
		{
			// Logic for parting
			if len(parts) > 1 {
				fmt.Fprintf(c.Connection, "PART %s\r\n", strings.TrimSpace(parts[1]))
				c.Channels = c.Channels[:len(c.Channels)-1]
			}
		}
	case "/NICK":
		{
			if len(parts) > 1 {
				fmt.Fprintf(c.Connection, "NICK %s\r\n", strings.TrimSpace(parts[1]))

				// TODO: Wait until we get a proper ACK from the server
				// c.Nickname = parts[1] // Update the nickname in the client state?
			}
		}

	case "/ME":
		{
			if len(parts) > 1 {
				action := strings.Join(parts[1:], " ")

				fmt.Fprintf(c.Connection, "PRIVMSG %s :\x01ACTION %s\x01\r\n", c.Channels[len(c.Channels)-1], action)
				c.Incoming <- &helpers.ChannelMessage{Timestamp: time.Now(), User: c.Nickname, Message: fmt.Sprintf("✧ %s ✧", action), Channel: c.Channels[len(c.Channels)-1]}
			}
		}

	case "/CHANNELS":
		{
			//chans := fmt.Sprintf("Joined channels: %s", strings.Join(c.Channels, ", "))
			//msg := helpers.ChannelMessage{Timestamp: time.Now(), User: "client", Message: chans, Channel: "system"}
			c.Incoming <- &helpers.ServerMessage{Timestamp: time.Now(), Message: fmt.Sprintf("Joined channels: %s", strings.Join(c.Channels, ", "))}
		}

	case "/NAMES":
		{
			if len(c.Channels) > 0 {
				fmt.Fprintf(c.Connection, "NAMES %s\r\n", c.Channels[len(c.Channels)-1])
			}
		}

	case "/WHOIS":
		{
			if len(parts) > 1 {
				fmt.Fprintf(c.Connection, "WHOIS %s\r\n", strings.TrimSpace(parts[1]))
			}
		}
	case "/QUIT":
		{
			if len(parts) > 1 {
				fmt.Fprintf(c.Connection, "QUIT :%s\r\n", strings.Join(parts[1:], " "))
			} else {
				fmt.Fprintf(c.Connection, "QUIT\r\n")
			}
		}

	case "/AWAY":
		{
			if len(parts) > 1 {
				fmt.Fprintf(c.Connection, "AWAY :%s\r\n", strings.Join(parts[1:], " "))
			}
		}
	case "/TOPIC":
		{
			if len(parts) > 2 {
				fmt.Fprintf(c.Connection, "TOPIC %s :%s\r\n", c.Channels[len(c.Channels)-1], parts[2])
			} else {
				// TOPIC OF CURRENT CHANNEL
				fmt.Fprintf(c.Connection, "TOPIC %s\r\n", c.Channels[len(c.Channels)-1])
			}
		}

	case "/INVITE":
		{
			if len(parts) > 2 {
				fmt.Fprintf(c.Connection, "INVITE %s %s\r\n", parts[1], parts[2])
			}
		}
	case "/KICK":
		{
			if len(parts) > 2 {
				// specified channel, user to kick, reason
				fmt.Fprintf(c.Connection, "KICK %s %s :%s\r\n", c.Channels[len(c.Channels)-1], parts[1], parts[2])
			}
		}

	case "/MODE":
		{
			if len(parts) > 2 {
				// specified channel, mode, and parameters
				fmt.Fprintf(c.Connection, "MODE %s %s :%s\r\n", c.Channels[len(c.Channels)-1], parts[1], parts[2])
			}
		}
		// if we dont support a command, let the user send over a RAW command to do a specified action....
	case "/RAW", "/QUOTE":
		{
			rawCommand := strings.Join(parts[1:], " ")
			fmt.Fprintf(c.Connection, "%s\r\n", rawCommand)

			c.Incoming <- &helpers.ServerMessage{
				Timestamp: time.Now(),
				Message:   fmt.Sprintf("[RAW CMD] -> %s", rawCommand),
			}
		}

	}

}

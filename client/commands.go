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
	// NOTE: For right now we are just going to handle the numeric commands that we care about, and ignore the rest.
	// Also, we are separating the numeric commands, even if they do the same thing, just incase we want to make custom messages
	// for each one in the future.
	switch msg.Command {
	case "001", "002", "003", "004", "005":
		{
			c.Incoming <- helpers.ParseServerMessage(msg, c.ServerID)
			return true
		}
	case "251", "252", "253", "254", "255", "265", "266", "250", "396":
		{
			// We don't really care about these raw stats for the UI right now.
			// Return true so they don't get passed to the RawMessage fallback.
			return true
		}
	case "301":
		{
			// can handle something in the future....
			c.Incoming <- helpers.ParseServerMessage(msg, c.ServerID)
			return true
		}
	case "324":
		{
			c.Incoming <- helpers.ParseServerMessage(msg, c.ServerID)
			return true
		}
	case "311", "312", "317", "319", "344", "671":
		{
			// These are WHOIS responses, we can display them to the user
			// this will return a server message...
			c.Incoming <- helpers.ParseWhoIsMessage(msg, c.ServerID)
			return true
		}
	case "331", "332", "333", "TOPIC":
		{
			if msg.Command == "331" {
				// 331 is "No topic is set", handle or skip
				return true
			}
			// This is the topic of a channel, we can display it to the user
			c.Incoming <- helpers.ParseTopicMessage(msg, c.ServerID)
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
				c.Incoming <- helpers.DeepCopyUserListMessage(UserList, c.ServerID) // Send a copy of the user
				return true
			}
			delete(UserList.UserList, helpers.LastChannel) // Clear the user list for the channel after sending it
		}
	case "372", "375":
		{
			// MOTD messages, we can display them to the user
			// TODO: Think about making it a MOTDMessage
			c.Incoming <- helpers.ParseServerMessage(msg, c.ServerID)
			return true
		}
	case "376":
		{
			c.Incoming <- &helpers.ServerMessage{Type: "SERVER", Timestamp: time.Now(),
				Message: "<--- Connection Established --->", Server: c.ServerID}

			// join some pre-marked channels
			for _, channel := range c.PreJoinChannels {
				fmt.Fprintf(c.Connection, "JOIN %s\r\n", channel)
			}

			return true
		}
	case "401", "403", "404", "433", "473", "474", "482": // we will make these the error cases....
		{
			// Generate a message to the user saying, use /nick to change nick name
			if len(msg.Params) > 1 {
				c.Incoming <- &helpers.ErrorMessage{Type: "ERROR", Timestamp: time.Now(),
					Message: strings.Join(msg.Params[1:], " "), Server: c.ServerID}
			} else {
				c.Incoming <- &helpers.ErrorMessage{Type: "ERROR", Timestamp: time.Now(), Message: "Unknown", Server: c.ServerID}
			}

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
	case "PRIVMSG":
		{
			return c.handlePrivMsg(msg)
		}

	case "JOIN", "PART":
		{
			c.Incoming <- helpers.ParseCommandMessages(msg, c.Host)
			return true
		}
	case "NICK":
		{
			if msg.Prefix.Name == c.Nickname {
				c.Nickname = msg.Params[0]
			}
			c.Incoming <- helpers.ParseCommandMessages(msg, c.ServerID)
			return true
		}
	case "MODE":
		{
			c.Incoming <- helpers.ParseCommandMessages(msg, c.ServerID)
			return true
		}
	case "NOTICE":
		{
			c.Incoming <- helpers.ParseCommandMessages(msg, c.ServerID)
			return true
		}
		// channel, user who got kicked, admin who kicked, and a reason
	case "KICK":
		{
			c.Incoming <- helpers.ParseCommandMessages(msg, c.ServerID)
			return true
		}
		// need to handle invites from a user...
		// might need to be its own type of message since we do have `extra logic` attached to it.
	case "INVITE":
		{
			c.Incoming <- helpers.ParseCommandMessages(msg, c.ServerID)
			return true
		}
	case "ERROR":
		{
			c.Incoming <- &helpers.ErrorMessage{Type: "ERROR", Timestamp: time.Now(), Message: msg.Params[0], Server: c.ServerID}
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

func (c *IRCClient) ParseUserInput(target, input string) {

	if len(c.Channels) != 0 && c.IsDev {
		c.SetCurrentChannel(c.Channels[len(c.Channels)-1])

	}

	if !strings.HasPrefix(input, "/") && c.IsDev {
		if len(c.Channels) == 0 {
			c.Incoming <- &helpers.ErrorMessage{Type: "ERROR", Timestamp: time.Now(),
				Message: "You need to join at least one channel. Use /join <channel> to join a channel",
				Server:  c.ServerID}
			return
		}

		fmt.Fprintf(c.Connection, "PRIVMSG %s :%s\r\n", c.CurrentChannel, input)
	}

	// send a message to the target channel
	if !strings.HasPrefix(input, "/") && !c.IsDev {
		fmt.Fprintf(c.Connection, "PRIVMSG %s :%s\r\n", target, input)

		c.sendMessage(target, input) // just for display purposes

		return
	}

	parts := strings.SplitN(input, " ", 3)
	command := strings.ToUpper(parts[0])
	c.LastCommand = command

	switch command {
	case "/JOIN":
		{
			if len(parts) > 1 {
				c.Channels = append(c.Channels, parts[1]) // TODO: Remove this
				fmt.Fprintf(c.Connection, "JOIN %s\r\n", strings.TrimSpace(parts[1]))
			}
		}
	case "/MSG":
		{
			if len(parts) > 2 {
				fmt.Fprintf(c.Connection, "PRIVMSG %s :%s\r\n", parts[1], parts[2])
				c.DirectMsgs = append(c.DirectMsgs, parts[2]) // TODO: Remove this
				c.sendMessage(parts[1], parts[2])
			}
		}

	case "/PART":
		{
			// Logic for parting
			if len(parts) > 1 {
				fmt.Fprintf(c.Connection, "PART %s\r\n", strings.TrimSpace(parts[1]))

				// TODO: Frontend will need to update the channel list
				// This would no user definition, just channel
			} else {
				fmt.Fprintf(c.Connection, "PART %s\r\n", strings.TrimSpace(target))
			}
		}
	case "/NICK":
		{
			if len(parts) > 1 {
				fmt.Fprintf(c.Connection, "NICK %s\r\n", strings.TrimSpace(parts[1]))
			}
		}

	case "/ME":
		{
			if len(parts) > 1 {
				action := strings.Join(parts[1:], " ")

				fmt.Fprintf(c.Connection, "PRIVMSG %s :\x01ACTION %s\x01\r\n", target, action)
				c.Incoming <- &helpers.ChannelMessage{Type: "Channel", Timestamp: time.Now(), User: c.Nickname,
					Message: fmt.Sprintf("✧ %s ✧", action), Channel: target, Server: c.ServerID}
			}
		}
		// frontend will display the channels that the user is in
	case "/CHANNELS":
		{
			//chans := fmt.Sprintf("Joined channels: %s", strings.Join(c.Channels, ", "))
			//msg := helpers.ChannelMessage{Timestamp: time.Now(), User: "client", Message: chans, Channel: "system"}
			c.Incoming <- &helpers.ServerMessage{Type: "Server", Timestamp: time.Now(),
				Message: fmt.Sprintf("Joined channels: %s", strings.Join(c.Channels, ", ")), Server: c.ServerID}
		}

	case "/NAMES":
		{
			// names in current channel
			if len(c.Channels) > 0 {
				fmt.Fprintf(c.Connection, "NAMES %s\r\n", target)
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
				// setting a topic for a channel
				fmt.Fprintf(c.Connection, "TOPIC %s :%s\r\n", parts[1], parts[2])
			} else {
				// TOPIC OF CURRENT CHANNEL
				fmt.Fprintf(c.Connection, "TOPIC %s\r\n", target)
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
				// current channel, user to kick, reason
				fmt.Fprintf(c.Connection, "KICK %s %s :%s\r\n", target, parts[1], parts[2])
			}
		}

	case "/MODE":
		{
			if len(parts) > 2 {
				// current channel, mode, and parameters
				fmt.Fprintf(c.Connection, "MODE %s %s :%s\r\n", target, parts[1], parts[2])
			}
		}
		// if we dont support a command, let the user send over a RAW command to do a specified action....
	case "/RAW", "/QUOTE":
		{
			rawCommand := strings.Join(parts[1:], " ")
			fmt.Fprintf(c.Connection, "%s\r\n", rawCommand)

			c.Incoming <- &helpers.ServerMessage{
				Type:      "Server",
				Timestamp: time.Now(),
				Message:   fmt.Sprintf("[RAW CMD] -> %s", rawCommand),
				Server:    c.ServerID,
			}
		}

	}

}

func (c *IRCClient) handlePrivMsg(msg *irc.Message) bool {
	if len(msg.Params) < 2 || msg.Prefix == nil {
		return false
	}

	target := msg.Params[0]
	text := msg.Params[1]
	sender := msg.Prefix.Name

	if strings.HasPrefix(text, "\x01") {
		return c.handleCTCP(sender, target, text)
	}

	// Standard IRC channels start with #, but some networks use & for local channels
	if strings.HasPrefix(target, "#") || strings.HasPrefix(target, "&") {
		c.Incoming <- helpers.ParseChannelMessages(msg, c.ServerID)
		return true
	}

	if target == c.Nickname {
		// Ignore self-DMs (we already rendered our outbound message in the UI)
		if sender == c.Nickname {
			return true
		}

		c.DirectMsgs = append(c.DirectMsgs, sender) // append the sender to the list of direct messages

		c.Incoming <- helpers.ParseDirectMessage(msg, c.ServerID)
		return true
	}

	return false
}

func (c *IRCClient) handleCTCP(sender, target, text string) bool {
	if strings.HasPrefix(text, "\x01VERSION") {
		// Respond with a NOTICE wrapped in \x01
		fmt.Fprintf(c.Connection, "NOTICE %s :\x01VERSION CustomGoClient:1.0\x01\r\n", sender)
		return true
	}

	// action messages start with \x01ACTION
	if strings.HasPrefix(text, "\x01ACTION ") {
		actionText := strings.TrimPrefix(text, "\x01ACTION ")
		actionText = strings.TrimSuffix(actionText, "\x01")

		c.Incoming <- &helpers.ChannelMessage{
			Type:      "Channel",
			Timestamp: time.Now(),
			User:      sender,
			Message:   fmt.Sprintf("✧ %s ✧", actionText),
			Channel:   target,
			Server:    c.ServerID,
		}
		return true
	}

	return false
}

// sendDMMessage is a helper function to send a direct message to a user.
// It checks if the target is not a channel and then sends the message to the Incoming channel for display.
func (c *IRCClient) sendMessage(target, message string) {
	if !strings.HasPrefix(target, "#") && !strings.HasPrefix(target, "&") {
		dmMsg := &helpers.DirectMessage{
			Type:      "DirectMessage",
			Timestamp: time.Now(),
			Sender:    c.Nickname,
			Receiver:  target,
			Message:   message,
			Server:    c.ServerID,
		}
		c.Incoming <- dmMsg
	} else {
		userMsg := &helpers.UserMessage{
			Type:      "UserMessage",
			Timestamp: time.Now(),
			User:      c.Nickname,
			Message:   message,
			Target:    target,
			Server:    c.ServerID,
		}
		c.Incoming <- userMsg

	}
}

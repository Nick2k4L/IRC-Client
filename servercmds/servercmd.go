package servercmds

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

func HandleNumeric(conn net.Conn, msg *irc.Message, line string, incoming chan helpers.StructuredMessage) bool {

	switch msg.Command {
	case "001", "002", "003", "004", "005":
		{

		}
	case "331", "332", "333", "TOPIC":
		{
			if msg.Command == "331" {
				// 331 is "No topic is set", handle or skip
				return true
			}
			// This is the topic of a channel, we can display it to the user
			incoming <- helpers.ParseTopicMessage(msg)
			return true
		}
	case "353":
		{
			helpers.ParseUserListMessage(msg, UserList)
			return true
		}
	case "366":
		{
			newMap := make(map[string][]string, len(UserList.UserList))

			for k, v := range UserList.UserList {
				// Allocate new slice with same capacity
				newSlice := make([]string, len(v))
				// Copy elements
				copy(newSlice, v)
				// Assign to new map
				newMap[k] = newSlice
			}

			incoming <- &helpers.UserListMessage{
				UserList: newMap,
				Channel:  helpers.LastChannel,
			}

			delete(UserList.UserList, helpers.LastChannel) // Clear the user list for the channel after sending it
			return true
		}
	case "433":
		{
			// Generate a message to the user saying, use /nick to change nick name
			incoming <- &helpers.ErrorMessage{Timestamp: time.Now(), Message: "Nickname already in use. Use /nick to change nick name."}
			return true
		}
	}

	return false
}

func HandleCommands(conn net.Conn, msg *irc.Message, line string, incoming chan helpers.StructuredMessage) bool {

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
			incoming <- helpers.ParseCommandMessages(msg)
			return true
		}
		// change this most likely
	case "NICK":
		{
			incoming <- helpers.ParseCommandMessages(msg)
			return true
		}
	case "MODE":
		{
			incoming <- helpers.ParseCommandMessages(msg)
			return true
		}
	case "NOTICE":
		{
			incoming <- helpers.ParseCommandMessages(msg)
			return true
		}
	case "QUIT":
		{
			// display of quit is not really necessary.
		}
		return true
	}

	return false
}

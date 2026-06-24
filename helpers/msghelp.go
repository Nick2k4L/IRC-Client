package helpers

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/go-irc/irc"
)

var commandMap = map[string]string{
	"JOIN":   "Joined",
	"PART":   "Left",
	"NICK":   "Changed Nickname to",
	"MODE":   "Sets Mode",
	"NOTICE": "Notice",
	"KICK":   "Kicked",
	"INVITE": "Invited You to",
}

var LastChannel string

type StructuredMessage interface {
	Formatted() string
}

type ChannelMessage struct {
	Timestamp time.Time
	Type      string `json:"type"`
	User      string `json:"user"`
	Message   string `json:"message"`
	Channel   string `json:"channel"`
	Server    string `json:"server"`
}

type CommandMessage struct {
	Timestamp time.Time `json:"timestamp"`
	Type      string    `json:"type"`
	User      string    `json:"user"`
	Reason    string    `json:"reason"`
	Channel   string    `json:"channel"`
	Command   string    `json:"command"`
	Message   string    `json:"message"`
	Server    string    `json:"server"`
}

type ErrorMessage struct {
	Timestamp time.Time `json:"timestamp"`
	Type      string    `json:"type"`
	Message   string    `json:"message"`
	Server    string    `json:"server"`
}

type DirectMessage struct {
	Timestamp time.Time `json:"timestamp"`
	Type      string    `json:"type"`
	Sender    string    `json:"sender"`
	Receiver  string    `json:"receiver"`
	Message   string    `json:"message"`
	Server    string    `json:"server"`
}

type RawMessage struct {
	Timestamp time.Time `json:"timestamp"`
	Message   string    `json:"message"`
}

type TopicMessage struct {
	Timestamp time.Time `json:"timestamp"`
	Type      string    `json:"type"`
	Channel   string    `json:"channel"`
	Topic     string    `json:"topic"`
	Server    string    `json:"server"`
}

type TopicMetadataMessage struct {
	Timestamp time.Time `json:"timestamp"`
	Time      time.Time `json:"time"`
	Type      string    `json:"type"`
	Channel   string    `json:"channel"`
	User      string    `json:"user"`
	Server    string    `json:"server"`
}

type UserListMessage struct {
	// maps to channel and list of users....
	UserList map[string][]string `json:"userlist"`
	Channel  string              `json:"channel"`
	Type     string              `json:"type"`
	Server   string              `json:"server"`
}

type ServerMessage struct {
	Timestamp time.Time `json:"timestamp"`
	Type      string    `json:"type"`
	Message   string    `json:"message"`
	Server    string    `json:"server"`
}

type UserMessage struct {
	Timestamp time.Time `json:"timestamp"`
	Type      string    `json:"type"`
	User      string    `json:"user"`
	Message   string    `json:"message"`
	Target    string    `json:"target"`
	Server    string    `json:"server"`
}

// user message

func (um *UserMessage) Formatted() string {
	// "["+time.Now().Format("15:04")+"] <✧"+m.client.Nickname+"✧> "+text
	return fmt.Sprintf("[%s] <✧%s✧> %s", um.Timestamp.Format("15:04"), um.User, um.Message)
}

func ParseWhoIsMessage(msg *irc.Message, server string) StructuredMessage {
	if len(msg.Params) < 2 {
		return &ServerMessage{
			Timestamp: time.Now(),
			Message:   "No message content",
			Server:    server,
		}
	}

	usersName := msg.Params[1]
	var messageText string

	switch msg.Command {
	case "311":
		messageText = fmt.Sprintf("%s has userhost %s@%s and real name is `%s` ", usersName, usersName, msg.Params[3], msg.Params[5])
	case "312":
		messageText = fmt.Sprintf("%s is connected on %s (%s)", usersName, msg.Params[2], msg.Params[3])
	case "319":
		messageText = fmt.Sprintf("%s is in the following channels: %s", usersName, msg.Params[2:])
	case "344":
		messageText = fmt.Sprintf("%s %s", usersName, msg.Params[3])
	case "317":
		secondsIdle, _ := strconv.ParseInt(msg.Params[2], 10, 64)
		unixTime, _ := strconv.ParseInt(msg.Params[3], 10, 64)
		messageText = fmt.Sprintf("%s has been idle for %s and has been connected since %s",
			usersName, time.Duration(secondsIdle)*time.Second, time.Unix(unixTime, 0).Format(time.RFC850))
	case "671":
		messageText = fmt.Sprintf("%s has SSL enabled", usersName)
	default:
		messageText = "Unhandled WHOIS response"
	}

	return &ServerMessage{
		Timestamp: time.Now(),
		Message:   messageText,
		Server:    server,
	}

}

// Direct Messages
func ParseDirectMessage(msg *irc.Message, server string) StructuredMessage {
	return &DirectMessage{
		Type:      "DirectMessage",
		Timestamp: time.Now(),
		Sender:    msg.Prefix.Name,
		Receiver:  msg.Params[0],
		Message:   msg.Params[1],
		Server:    server,
	}
}

func (dm *DirectMessage) Formatted() string {
	return fmt.Sprintf("[%s] DM from %s to %s: %s", dm.Timestamp.Format("15:04"), dm.Sender, dm.Receiver, dm.Message)
}

// Server Messages

func ParseServerMessage(msg *irc.Message, server string) StructuredMessage {

	if len(msg.Params) < 2 {
		return &ServerMessage{
			Type:      "SERVER",
			Timestamp: time.Now(),
			Message:   "No message content",
			Server:    server,
		}
	}

	fullMessage := strings.Join(msg.Params[1:], " ")
	return &ServerMessage{
		Type:      "SERVER",
		Timestamp: time.Now(),
		Message:   fullMessage,
		Server:    server,
	}

}

// "✧"
func (sm *ServerMessage) Formatted() string {
	return fmt.Sprintf("[%s] SERVER: %s", sm.Timestamp.Format("15:04"), sm.Message)
}

// USER LIST MESSAGES

func ParseUserListMessage(msg *irc.Message, ul UserListMessage) {

	if len(msg.Params) > 3 {
		channel := msg.Params[2]
		rawUsers := msg.Params[3]

		users := strings.Split(rawUsers, " ")
		ul.UserList[channel] = append(ul.UserList[channel], users...)
		LastChannel = channel
	}
}

func DeepCopyUserListMessage(ul UserListMessage, server string) StructuredMessage {
	newMap := make(map[string][]string, len(ul.UserList))

	for k, v := range ul.UserList {
		// Allocate new slice with same capacity
		newSlice := make([]string, len(v))
		// Copy elements
		copy(newSlice, v)
		// Assign to new map
		newMap[k] = newSlice
	}

	return &UserListMessage{
		Type:     "User List",
		UserList: newMap,
		Channel:  LastChannel,
		Server:   server,
	}

}
func (ul *UserListMessage) Formatted() string {
	separated := strings.Join(ul.UserList[LastChannel], "\n*")
	return fmt.Sprintf("USER LIST:\n %v", separated)
}

// TOPIC MESSAGES

func ParseTopicMessage(msg *irc.Message, server string) StructuredMessage {
	now := time.Now()

	if msg.Command == "333" {
		unixTime, _ := strconv.ParseInt(msg.Params[3], 10, 64)

		return &TopicMetadataMessage{
			Type:      "Topic Metadata",
			Timestamp: now,
			Channel:   msg.Params[1],
			User:      msg.Params[2],
			Time:      time.Unix(unixTime, 0),
			Server:    server,
		}
	}

	return &TopicMessage{
		Type:      "Topic",
		Timestamp: now,
		Channel:   msg.Params[1],
		Topic:     msg.Params[2],
		Server:    server,
	}

}

func (tm *TopicMessage) Formatted() string {
	return fmt.Sprintf("[%s] TOPIC: %s", tm.Timestamp.Format("15:04"), tm.Topic)
}

func (tmm *TopicMetadataMessage) Formatted() string {
	return fmt.Sprintf("[%s] {%s} Set by %s on %s", tmm.Timestamp.Format("15:04"), tmm.Channel, tmm.User, tmm.Time.Format(time.RFC850))
}

// RAW MESSAGES -- this is mostly for debugging

func ParseRawMessages(msg *irc.Message) StructuredMessage {
	return &RawMessage{
		Timestamp: time.Now(),
		Message:   msg.String(),
	}

}

func (rm *RawMessage) Formatted() string {
	return fmt.Sprintf("[%s] RAW: %s", rm.Timestamp.Format("15:04"), rm.Message)
}

// ERROR MESSAGES

func (em *ErrorMessage) Formatted() string {
	return fmt.Sprintf("[%s] ERROR: %s", em.Timestamp.Format("15:04"), em.Message)
}

// CHANNEL MESSAGES

func ParseChannelMessages(msg *irc.Message, server string) StructuredMessage {

	if len(msg.Params) >= 2 {
		return &ChannelMessage{
			Type:      "Channel",
			Timestamp: time.Now(),
			User:      msg.Prefix.Name,
			Message:   msg.Params[1],
			Channel:   msg.Params[0],
			Server:    server,
		}
	}

	return &ChannelMessage{
		Type:      "Channel",
		Timestamp: time.Now(),
		User:      msg.Prefix.Name,
		Message:   " <---- Could Not Properly Parse Message ----> ",
		Channel:   msg.Params[0],
		Server:    server,
	}
}

func (cm *ChannelMessage) Formatted() string {
	return fmt.Sprintf("[%s] {%s} <%s> %s", cm.Timestamp.Format("15:04"), cm.Channel, cm.User, cm.Message)
}

// COMMANDS

func ParseCommandMessages(msg *irc.Message, server string) StructuredMessage {

	if len(msg.Params) > 1 {
		return &CommandMessage{
			Timestamp: time.Now(),
			Type:      "Command",
			User:      msg.Prefix.Name,
			Command:   msg.Command,
			Channel:   msg.Params[0],
			Message:   msg.Params[1],
			Server:    server,
		}
	}

	return &CommandMessage{
		Timestamp: time.Now(),
		Type:      "Command",
		User:      msg.Prefix.Name,
		Command:   msg.Command,
		Channel:   msg.Params[0],
		Server:    server,
	}
}

func (cm *CommandMessage) Formatted() string {

	if cm.User != "" && cm.Message != "" {
		return fmt.Sprintf("[%s] ✧ %s %s %s", cm.Timestamp.Format("15:04"), cm.User, commandMap[cm.Command], cm.Message)

	}

	if cm.User != "" {
		return fmt.Sprintf("[%s] ✧ %s %s %s", cm.Timestamp.Format("15:04"), cm.User, commandMap[cm.Command], cm.Channel)
	}

	return fmt.Sprintf("[%s] : {%s} %s", cm.Timestamp.Format("15:04"), cm.Channel, cm.Command)
}

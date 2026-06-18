package helpers

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/go-irc/irc"
)

var commandMap map[string]string = map[string]string{
	"JOIN": "Joined",
	"PART": "Left",
	"NICK": "Changed Nickname to",
}

var LastChannel string

type StructuredMessage interface {
	Formatted() string
}

type ChannelMessage struct {
	Timestamp              time.Time
	User, Message, Channel string
}

type CommandMessage struct {
	Timestamp                      time.Time
	User, Reason, Channel, Command string
}

type ErrorMessage struct {
	Timestamp time.Time
	Message   string
}

type DirectMessage struct {
	Timestamp                 time.Time
	Sender, Receiver, Message string
}

type RawMessage struct {
	Timestamp time.Time
	Message   string
}

type TopicMessage struct {
	Timestamp      time.Time
	Channel, Topic string
}

type TopicMetadataMessage struct {
	Timestamp time.Time
	Channel   string
	User      string
	Time      time.Time
}

type UserListMessage struct {
	// maps to channel and list of users....
	UserList map[string][]string
	Channel  string
}

type ServerMessage struct {
	Timestamp time.Time
	Message   string
}

// Direct Messages

func ParseDirectMessage(msg *irc.Message) StructuredMessage {
	return &DirectMessage{
		Timestamp: time.Now(),
		Sender:    msg.Prefix.Name,
		Receiver:  msg.Params[0],
		Message:   msg.Params[1],
	}
}

func (dm *DirectMessage) Formatted() string {
	return fmt.Sprintf("[%s] DM from %s to %s: %s", dm.Timestamp.Format("15:04"), dm.Sender, dm.Receiver, dm.Message)
}

// Server Messages

func ParseServerMessage(msg *irc.Message) StructuredMessage {
	if len(msg.Params) == 0 {
		return &ServerMessage{
			Timestamp: time.Now(),
			Message:   "No message content",
		}
	}

	fullMessage := strings.Join(msg.Params[1:], " ")
	return &ServerMessage{
		Timestamp: time.Now(),
		Message:   fullMessage,
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

func DeepCopyUserListMessage(ul UserListMessage) StructuredMessage {
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
		UserList: newMap,
		Channel:  LastChannel,
	}

}
func (ul *UserListMessage) Formatted() string {
	separated := strings.Join(ul.UserList[LastChannel], "\n*")
	return fmt.Sprintf("USER LIST:\n %v", separated)
}

// TOPIC MESSAGES

func ParseTopicMessage(msg *irc.Message) StructuredMessage {
	now := time.Now()

	if msg.Command == "333" {
		unixTime, _ := strconv.ParseInt(msg.Params[3], 10, 64)

		return &TopicMetadataMessage{
			Timestamp: now,
			Channel:   msg.Params[1],
			User:      msg.Params[2],
			Time:      time.Unix(unixTime, 0),
		}
	}

	return &TopicMessage{
		Timestamp: now,
		Channel:   msg.Params[1],
		Topic:     msg.Params[2],
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

func ParseChannelMessages(msg *irc.Message) StructuredMessage {

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
	return fmt.Sprintf("[%s] {%s} <%s> %s", cm.Timestamp.Format("15:04"), cm.Channel, cm.User, cm.Message)
}

// COMMANDS

func ParseCommandMessages(msg *irc.Message) StructuredMessage {
	return &CommandMessage{
		Timestamp: time.Now(),
		User:      msg.Prefix.Name,
		Command:   msg.Command,
		Channel:   msg.Params[0],
	}
}

func (cm *CommandMessage) Formatted() string {
	if cm.User != "" {
		return fmt.Sprintf("[%s] ✧ %s %s %s", cm.Timestamp.Format("15:04"), cm.User, commandMap[cm.Command], cm.Channel)
	}

	return fmt.Sprintf("[%s] : {%s} %s", cm.Timestamp.Format("15:04"), cm.Channel, cm.Command)
}

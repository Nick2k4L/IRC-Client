package helpers

import (
	"fmt"
	"time"

	"github.com/go-irc/irc"
)

var commandMap map[string]string = map[string]string{
	"JOIN": "Joined",
	"PART": "Left",
	"NICK": "Changed Nickname",
}

type StructuredMessage interface {
	RawToReadable(msg *irc.Message) StructuredMessage
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

func (cm *ChannelMessage) RawToReadable(msg *irc.Message) StructuredMessage {

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
	return fmt.Sprintf("[%s] {%s} <%s> : %s", cm.Timestamp.Format("15:04"), cm.Channel, cm.User, cm.Message)
}

// COMMANDS

func (cm *CommandMessage) RawToReadable(msg *irc.Message) StructuredMessage {
	return &CommandMessage{
		Timestamp: time.Now(),
		User:      msg.Prefix.Name,
		Command:   msg.Command,
		Channel:   msg.Params[0],
	}
}

func (cm *CommandMessage) Formatted() string {
	if cm.User != "" {
		return fmt.Sprintf("[%s] # %s %s %s", cm.Timestamp.Format("15:04"), cm.User, commandMap[cm.Command], cm.Channel)
	}

	return fmt.Sprintf("[%s] : {%s} %s", cm.Timestamp.Format("15:04"), cm.Channel, cm.Command)
}

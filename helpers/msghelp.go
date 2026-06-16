package helpers

import (
	"fmt"
	"strconv"
	"time"

	"github.com/go-irc/irc"
)

var commandMap map[string]string = map[string]string{
	"JOIN": "Joined",
	"PART": "Left",
	"NICK": "Changed Nickname",
}

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
	Timestamp     time.Time
	User, Message string
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

// RAW MESSAGES

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
	return fmt.Sprintf("[%s] {%s} <%s> : %s", cm.Timestamp.Format("15:04"), cm.Channel, cm.User, cm.Message)
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
		return fmt.Sprintf("[%s] # %s %s %s", cm.Timestamp.Format("15:04"), cm.User, commandMap[cm.Command], cm.Channel)
	}

	return fmt.Sprintf("[%s] : {%s} %s", cm.Timestamp.Format("15:04"), cm.Channel, cm.Command)
}

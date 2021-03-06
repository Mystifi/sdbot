package sdbot

import (
	"regexp"
	"strconv"
	"strings"
	"time"
)

// Message represents a message sent by a user to either a room the bot is
// currently in, or to the bot via private messages. A message also defines
// behaviour in its methods to reply to these messages.
type Message struct {
	Bot       *Bot
	Time      time.Time
	Command   string
	Params    []string
	Timestamp int
	Room      *Room
	User      *User
	Auth      string
	Target    Target
	Message   string
	Matches   map[string]map[*regexp.Regexp][]string
}

// NewMessage creates a new message and parses the message.
func NewMessage(s string, bot *Bot) *Message {
	m := &Message{
		Bot:     bot,
		Time:    time.Now(),
		Matches: make(map[string]map[*regexp.Regexp][]string),
	}
	m.Command, m.Params, m.Timestamp, m.Room, m.User, m.Auth, m.Target, m.Message = parseMessage(s, bot)
	return m
}

// Parse a raw message and return data in the following order:
// Command, Params, Timestamp, Room, User, Auth, Target, Message
// TODO Reduce cyclic complexity? Lots of ifs ands and switches.
func parseMessage(s string, b *Bot) (string, []string, int, *Room, *User, string, Target, string) {
	newlineDelimited := strings.Split(s, "\n")
	vertbarDelimited := strings.Split(s, "|")

	var command string
	var params []string
	var timestamp int
	var room *Room
	var user *User
	var auth string
	var message string

	// The command is always after the first vertical bar.
	if len(vertbarDelimited) < 2 {
		command = "none"
	} else {
		command = string(vertbarDelimited[1])
	}

	// Parse the parameters following a command.
	if command == "" {
		params = []string{}
	} else {
		if len(vertbarDelimited) > 2 {
			params = vertbarDelimited[2:]
		} else {
			params = []string{}
		}
	}

	// Parse the timestamp of a chat event.
	var err error
	if strings.Contains(command, ":") {
		timestamp, err = strconv.Atoi(params[0])
		CheckErr(err)
	} else {
		timestamp = 0
	}

	// If the message starts with a ">" then it comes from a room.
	if newlineDelimited[0] == "" {
		room = &Room{}
	} else {
		if string(newlineDelimited[0][0]) == ">" {
			room = FindRoomEnsured(string(newlineDelimited[0][1:]), b)
		} else {
			room = &Room{}
		}
	}

	// Parse the user sending a command, and their auth level.
	switch strings.ToLower(command) {
	case "c:":
		auth = string(vertbarDelimited[3][0])
		user = FindUserEnsured(string(vertbarDelimited[3][1:]), b)
	case "c":
		fallthrough
	case "j":
		fallthrough
	case "l":
		fallthrough
	case "n":
		fallthrough
	case "pm":
		auth = string(vertbarDelimited[2][0])
		user = FindUserEnsured(string(vertbarDelimited[2][1:]), b)
	}

	// Parse the message
	if command == "" {
		message = ""
	} else {
		switch strings.ToLower(command) {
		case "c:", "pm":
			message = strings.Join(vertbarDelimited[4:], "|")
		case "none":
			message = newlineDelimited[len(newlineDelimited)-1]
		}
	}

	// Decide the target
	if strings.ToLower(command) == "pm" {
		return command, params, timestamp, room, user, auth, user, message
	}
	return command, params, timestamp, room, user, auth, room, message
}

// Reply responds to a message and prepends the username of the user the bot
// is responding to.
func (m *Message) Reply(res string) {
	m.Target.Reply(m, res)
}

// RawReply responds to a message without prepending anything to the message.
// Note that you should take care to not allow users to influence a raw
// reply message to do a client command. For this reason, prefer to use
// Reply unless you are responding with a static message. You may want to
// event freeze the string.
func (m *Message) RawReply(res string) {
	m.Target.RawReply(m, res)
}

// Match adds matches to the message and return true if there was no previous
// match and if there was indeed a match.
func (m *Message) Match(r *regexp.Regexp, event string) bool {
	if m.Matches[event][r] == nil {
		matches := r.FindStringSubmatch(m.Message)
		if matches != nil {
			m.Matches[event][r] = matches
			return true
		}
	}
	return false
}

// Private returns true if the message was sent in a private message.
func (m *Message) Private() bool {
	return m.User == m.Target
}

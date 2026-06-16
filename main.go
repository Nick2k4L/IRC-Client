package main

import (
	"strings"

	"github.com/Nick2k4L/IRC-Client/client"
	tea "github.com/charmbracelet/bubbletea"
)

// Model holds the program's state
type model struct {
	client   *client.IRCClient
	input    []rune
	cursor   int
	messages []string
	err      error
}

// Init is called once at the start of the program
func (m model) Init() tea.Cmd {
	return m.client.ReadMessages()
}

// Update handles user input
func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case client.IncomingMsg:

		m.messages = append(m.messages, msg.Formatted())
		return m, m.client.ReadMessages()
	case client.ErrMsg:
		m.err = msg
		return m, tea.Quit
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			m.client.Disconnect()
			return m, tea.Quit
		case "enter":
			text := strings.TrimSpace(string(m.input))
			if text != "" {
				m.client.Send(text)
				m.messages = append(m.messages, "[YOU] : "+text)
			}
			m.input = nil
			m.cursor = 0
		case "backspace":
			if m.cursor > 0 {
				m.input = append(m.input[:m.cursor-1], m.input[m.cursor:]...)
				m.cursor--
			}
		case "left":
			if m.cursor > 0 {
				m.cursor--
			}
		case "right":
			if m.cursor < len(m.input) {
				m.cursor++
			}
		default:
			if len(msg.Runes) > 0 {
				input := make([]rune, 0, len(m.input)+len(msg.Runes))
				input = append(input, m.input[:m.cursor]...)
				input = append(input, msg.Runes...)
				input = append(input, m.input[m.cursor:]...)
				m.input = input
				m.cursor += len(msg.Runes)
			}
		}
	}
	return m, nil
}

// View renders the UI
func (m model) View() string {
	var b strings.Builder

	b.WriteString("IRC Client (Enter to send, Ctrl+C/q to quit)\n\n")
	if len(m.messages) == 0 {
		b.WriteString("No messages yet.\n")
	} else {
		b.WriteString(strings.Join(m.messages, "\n"))
		b.WriteRune('\n')
	}

	if m.err != nil {
		b.WriteString("\nError: ")
		b.WriteString(m.err.Error())
		b.WriteRune('\n')
	}

	b.WriteString("\n> ")
	b.WriteString(string(m.input[:m.cursor]))
	b.WriteRune('|')
	b.WriteString(string(m.input[m.cursor:]))

	return b.String()
}

func main() {
	client := client.NewIRCClient("irc.hybridirc.com", "Shinobu123498", 6697, true)
	client.Connect()

	p := tea.NewProgram(model{client: client})
	if _, err := p.Run(); err != nil {
		panic(err)
	}

}

// So what are the fundamentals in creating an IRC Client?

// Nickname, Host, Port, Channels, Messages, establishing a connection

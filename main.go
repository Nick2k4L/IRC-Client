package main

import (
	"strings"

	"github.com/Nick2k4L/IRC-Client/client"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
)

// Model holds the program's state
type model struct {
	client   *client.IRCClient
	input    []rune
	cursor   int
	messages []string
	err      error
	viewport viewport.Model // Handles the scrollable window
	ready    bool           // Tracks if terminal size is initialized
}

// Init is called once at the start of the program
func (m model) Init() tea.Cmd {
	return m.client.ReadMessages()
}

// Update handles user input and system events
func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var (
		cmd  tea.Cmd
		cmds []tea.Cmd
	)

	switch msg := msg.(type) {

	// 1. Handle Terminal Resizing
	case tea.WindowSizeMsg:
		// We leave space at the top for the header, and bottom for input/errors
		headerHeight := 2
		footerHeight := 4
		verticalMarginHeight := headerHeight + footerHeight

		if !m.ready {
			m.viewport = viewport.New(msg.Width, msg.Height-verticalMarginHeight)
			m.viewport.SetContent("No messages yet.")
			m.ready = true
		} else {
			m.viewport.Width = msg.Width
			m.viewport.Height = msg.Height - verticalMarginHeight
		}

	// 2. Handle Incoming IRC Messages
	case client.IncomingMsg:
		m.messages = append(m.messages, msg.Formatted())
		m.viewport.SetContent(strings.Join(m.messages, "\n"))
		m.viewport.GotoBottom() // Auto-scroll on new message
		return m, m.client.ReadMessages()

	// 3. Handle Errors
	case client.ErrMsg:
		m.err = msg
		return m, tea.Quit

	// 4. Handle Key Presses
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q": // Note: 'q' here means typing 'q' alone will quit. Consider changing this if users need to type 'q' in chat!
			m.client.Disconnect()
			return m, tea.Quit

		case "enter":
			text := strings.TrimSpace(string(m.input))
			if text != "" {
				m.client.Send(text)
				m.messages = append(m.messages, "[YOU] : "+text)

				// Update viewport immediately when you send a message
				m.viewport.SetContent(strings.Join(m.messages, "\n"))
				m.viewport.GotoBottom()
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

		// Route scrolling keys directly to the viewport
		case "up", "down", "pgup", "pgdown":
			m.viewport, cmd = m.viewport.Update(msg)
			cmds = append(cmds, cmd)
			return m, tea.Batch(cmds...)

		default:
			// Capture standard typing
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

	// Ensure the viewport processes any internal events it needs
	m.viewport, cmd = m.viewport.Update(msg)
	cmds = append(cmds, cmd)

	return m, tea.Batch(cmds...)
}

// View renders the UI
func (m model) View() string {
	if !m.ready {
		return "Initializing...\n"
	}

	var b strings.Builder

	// Header
	b.WriteString("IRC Client (Enter to send, Arrows to scroll, Ctrl+C to quit)\n\n")

	// Viewport (The scrollable chat history)
	b.WriteString(m.viewport.View())
	b.WriteRune('\n')

	// Error Display (if any)
	if m.err != nil {
		b.WriteString("\nError: ")
		b.WriteString(m.err.Error())
		b.WriteRune('\n')
	}

	// Input Prompt
	b.WriteString("\n> ")
	b.WriteString(string(m.input[:m.cursor]))
	b.WriteRune('|')
	b.WriteString(string(m.input[m.cursor:]))

	return b.String()
}

func main() {
	// Initialize your client using your custom package
	ircClient := client.NewIRCClient("irc.hybridirc.com", "ShinbouFave23", 6697, true)
	ircClient.Connect()

	// Pass the client into the Bubble Tea model
	p := tea.NewProgram(model{client: ircClient}, tea.WithAltScreen()) // AltScreen creates a full-terminal window

	if _, err := p.Run(); err != nil {
		panic(err)
	}
}

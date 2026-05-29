package tui

import (
	tea "github.com/charmbracelet/bubbletea"

	"github.com/bluefunda/abaper/tui/slash"
)

// Model is the root Bubble Tea model.
type Model struct {
	chat   *chatModel
	width  int
	height int
}

// New creates the root TUI model.
func New(version string) *Model {
	return &Model{
		chat: newChatModel(version),
	}
}

func (m *Model) Init() tea.Cmd {
	return m.chat.Init()
}

func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.chat.SetSize(msg.Width, msg.Height)
		return m, nil

	case tea.KeyMsg:
		// Global quit (Ctrl+C / Ctrl+D) when not streaming
		if !m.chat.streaming {
			switch msg.Type {
			case tea.KeyCtrlC, tea.KeyCtrlD:
				return m, tea.Quit
			}
		}

	// Slash command outcomes that live at app level
	case slash.QuitMsg:
		return m, tea.Quit

	case slash.ClearMsg:
		m.chat.messages = nil
		m.chat.rebuildViewport()
		return m, nil

	case slash.HelpMsg:
		m.chat.messages = append(m.chat.messages, chatMessage{
			kind:    kindSystem,
			content: helpText,
		})
		m.chat.rebuildViewport()
		return m, nil

	case slash.UnknownCmdMsg:
		// stub commands: silently ignore for now
		return m, nil
	}

	// Delegate everything else to the chat model
	newChat, cmd := m.chat.Update(msg)
	m.chat = newChat
	return m, cmd
}

func (m *Model) View() string {
	if m.width == 0 {
		return ""
	}
	return m.chat.View()
}

const helpText = `Commands: /help /clear /compact /object /activate /transport /diff /review /test /profile /model /cost /settings /logout /quit
Keys: Enter submit · Shift+Enter newline · Ctrl+C/Esc cancel stream · Ctrl+C quit · ↑↓ scroll · / commands`

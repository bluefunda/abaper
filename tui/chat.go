package tui

import (
	"context"
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/glamour"
	"github.com/charmbracelet/lipgloss"
	"github.com/google/uuid"

	"github.com/bluefunda/abaper/internal/client"
	"github.com/bluefunda/abaper/internal/config"
	"github.com/bluefunda/abaper/styles"
	"github.com/bluefunda/abaper/tui/slash"
)

// ── message types ──────────────────────────────────────────────────────────

type msgKind int

const (
	kindLogo      msgKind = iota // startup logo block
	kindUser                     // human turn
	kindAssistant                // content built incrementally during streaming
	kindTool                     // tool execution status line
	kindSystem                   // dim informational line
	kindError
)

type chatMessage struct {
	kind       msgKind
	content    string
	toolName   string // for kindTool
	toolStatus string // "running" | "completed" | "failed"
	durationMs int
}

// ── tea messages ───────────────────────────────────────────────────────────

type streamChunkMsg struct{ content string }
type streamToolMsg  struct {
	name, status string
	durationMs   int
}
type streamDoneMsg struct{ err error }

// ── layout constants ───────────────────────────────────────────────────────

const (
	statusBarHeight = 1
	sepHeight       = 1
	inputHeight     = 3
	normalOverhead  = statusBarHeight + sepHeight + inputHeight
)

// ── chat model ─────────────────────────────────────────────────────────────

type chatModel struct {
	version  string
	messages []chatMessage

	viewport viewport.Model
	textarea textarea.Model
	spinner  spinner.Model
	renderer *glamour.TermRenderer

	streaming    bool
	cancelStream context.CancelFunc
	streamCh     chan client.ChatEvent

	chatID    string
	model     string
	isNewChat bool

	slashOpen bool
	slashMenu *slash.MenuModel

	width  int
	height int
}

func newChatModel(version string) *chatModel {
	ta := textarea.New()
	ta.Placeholder = "Type your ABAP request or / for commands"
	ta.ShowLineNumbers = false
	ta.SetHeight(inputHeight)
	ta.CharLimit = 0
	// Shift+Enter for newline; plain Enter is intercepted for submit
	ta.KeyMap.InsertNewline.SetKeys("shift+enter", "ctrl+j")
	ta.Focus()

	sp := spinner.New()
	sp.Spinner = spinner.Dot
	sp.Style = lipgloss.NewStyle().Foreground(styles.ColorAccent)

	loggedIn := false
	if _, err := config.LoadTokens(); err == nil {
		loggedIn = true
	}

	m := &chatModel{
		version:   version,
		spinner:   sp,
		textarea:  ta,
		chatID:    uuid.New().String(),
		model:     "groq:openai/gpt-oss-120b",
		isNewChat: true,
	}

	m.messages = []chatMessage{{kind: kindLogo}}
	if !loggedIn {
		m.messages = append(m.messages, chatMessage{
			kind:    kindSystem,
			content: "Not logged in — run `abaper login` to authenticate.",
		})
	}

	return m
}

func (m *chatModel) Init() tea.Cmd {
	return textarea.Blink
}

// SetSize recalculates component sizes when the terminal is resized.
func (m *chatModel) SetSize(w, h int) {
	m.width = w
	m.height = h
	m.textarea.SetWidth(w - 2)
	m.rebuildViewport()

	var err error
	m.renderer, err = glamour.NewTermRenderer(
		glamour.WithStandardStyle("dark"),
		glamour.WithWordWrap(w-6),
	)
	if err != nil {
		m.renderer = nil
	}
}

func (m *chatModel) viewportHeight() int {
	extra := 0
	if m.slashOpen && m.slashMenu != nil {
		extra = m.slashMenu.Height()
	}
	h := m.height - normalOverhead - extra
	if h < 3 {
		h = 3
	}
	return h
}

func (m *chatModel) rebuildViewport() {
	atBottom := m.viewport.AtBottom()
	m.viewport.Width = m.width
	m.viewport.Height = m.viewportHeight()
	m.viewport.SetContent(m.renderMessages())
	if atBottom {
		m.viewport.GotoBottom()
	}
}

// ── Update ─────────────────────────────────────────────────────────────────

func (m *chatModel) Update(msg tea.Msg) (*chatModel, tea.Cmd) {
	var cmds []tea.Cmd

	// ── slash menu is open: route events into it ──────────────────────────
	if m.slashOpen {
		switch ev := msg.(type) {
		case slash.SelectedMsg:
			m.slashOpen = false
			m.rebuildViewport()
			m.textarea.Focus()
			return m, tea.Batch(textarea.Blink, m.runSlashCmd(ev.Cmd, ev.Args))

		case slash.CancelMsg:
			m.slashOpen = false
			m.rebuildViewport()
			m.textarea.Focus()
			return m, textarea.Blink

		default:
			newMenu, cmd := m.slashMenu.Update(msg)
			m.slashMenu = newMenu
			cmds = append(cmds, cmd)
			// Rebuild in case filter changed command list height
			m.rebuildViewport()
			return m, tea.Batch(cmds...)
		}
	}

	// ── key events ────────────────────────────────────────────────────────
	if keyMsg, ok := msg.(tea.KeyMsg); ok {
		switch {
		// Cancel active stream
		case (keyMsg.Type == tea.KeyCtrlC || keyMsg.Type == tea.KeyEsc) && m.streaming:
			if m.cancelStream != nil {
				m.cancelStream()
			}
			return m, nil

		// Show help on `?`
		case keyMsg.Type == tea.KeyRunes && len(keyMsg.Runes) == 1 &&
			keyMsg.Runes[0] == '?' && !m.streaming && !m.slashOpen:
			return m, func() tea.Msg { return slash.HelpMsg{} }

		// Open slash menu when `/` is typed into an empty input
		case keyMsg.Type == tea.KeyRunes &&
			len(keyMsg.Runes) == 1 && keyMsg.Runes[0] == '/' &&
			strings.TrimSpace(m.textarea.Value()) == "" &&
			!m.streaming:
			m.slashOpen = true
			menuW := m.width - 4
			if menuW < 44 {
				menuW = 44
			}
			m.slashMenu = slash.NewMenu(menuW)
			m.rebuildViewport()
			return m, nil // do NOT pass '/' to textarea

		// Submit on Enter (shift+enter handled by textarea for newlines)
		case keyMsg.Type == tea.KeyEnter && !m.streaming:
			input := strings.TrimSpace(m.textarea.Value())
			if input == "" {
				return m, nil
			}
			// Typed slash command (e.g. `/quit`)
			if strings.HasPrefix(input, "/") {
				parts := strings.Fields(strings.TrimPrefix(input, "/"))
				if len(parts) > 0 {
					if cmd := slash.Lookup(parts[0]); cmd != nil {
						m.textarea.Reset()
						return m, m.runSlashCmd(*cmd, parts[1:])
					}
				}
			}
			m.textarea.Reset()
			return m, m.sendMessage(input)
		}
	}

	// ── streaming events ──────────────────────────────────────────────────
	switch ev := msg.(type) {
	case streamChunkMsg:
		m.appendToLastAssistant(ev.content)
		m.rebuildViewport()
		return m, m.waitForStream()

	case streamToolMsg:
		m.upsertTool(ev.name, ev.status, ev.durationMs)
		m.rebuildViewport()
		return m, m.waitForStream()

	case streamDoneMsg:
		m.streaming = false
		m.cancelStream = nil
		if ev.err != nil && ev.err != context.Canceled {
			m.messages = append(m.messages, chatMessage{kind: kindError, content: ev.err.Error()})
		}
		m.rebuildViewport()
		return m, textarea.Blink

	case spinner.TickMsg:
		if m.streaming {
			var cmd tea.Cmd
			m.spinner, cmd = m.spinner.Update(msg)
			cmds = append(cmds, cmd)
			// Re-render viewport so spinner frame updates in place
			m.viewport.SetContent(m.renderMessages())
		}
	}

	// ── pass through to textarea and viewport ─────────────────────────────
	var taCmd, vpCmd tea.Cmd
	m.textarea, taCmd = m.textarea.Update(msg)
	m.viewport, vpCmd = m.viewport.Update(msg)
	cmds = append(cmds, taCmd, vpCmd)

	return m, tea.Batch(cmds...)
}

// ── slash command execution ────────────────────────────────────────────────

func (m *chatModel) runSlashCmd(cmd slash.Command, args []string) tea.Cmd {
	if cmd.Handler == nil {
		m.messages = append(m.messages, chatMessage{
			kind:    kindSystem,
			content: fmt.Sprintf("/%s — not yet implemented", cmd.Name),
		})
		m.rebuildViewport()
		return nil
	}
	return func() tea.Msg {
		innerCmd := cmd.Handler(args)
		if innerCmd == nil {
			return slash.UnknownCmdMsg{Name: cmd.Name}
		}
		return innerCmd()
	}
}

// ── streaming ─────────────────────────────────────────────────────────────

func (m *chatModel) sendMessage(input string) tea.Cmd {
	m.messages = append(m.messages, chatMessage{kind: kindUser, content: input})
	m.messages = append(m.messages, chatMessage{kind: kindAssistant, content: ""})
	m.streaming = true
	m.rebuildViewport()

	ch := make(chan client.ChatEvent, 64)
	m.streamCh = ch

	ctx, cancel := context.WithCancel(context.Background())
	m.cancelStream = cancel

	isNew := m.isNewChat
	m.isNewChat = false

	chatID := m.chatID
	model := m.model

	go func() {
		defer close(ch)

		c, err := client.NewClient()
		if err != nil {
			ch <- client.ChatEvent{Type: "error", Error: err.Error()}
			return
		}

		req := client.ChatRequest{
			Prompt:    input,
			Model:     model,
			AgentName: "abaper",
			IsNewChat: isNew,
		}

		err = c.StreamChat(ctx, chatID, req, func(ev client.ChatEvent) {
			select {
			case ch <- ev:
			case <-ctx.Done():
			}
		})

		if err != nil && ctx.Err() == nil {
			ch <- client.ChatEvent{Type: "error", Error: err.Error()}
		}
	}()

	return tea.Batch(m.spinner.Tick, m.waitForStream())
}

// waitForStream is a recursive Cmd that drains one meaningful event from streamCh.
func (m *chatModel) waitForStream() tea.Cmd {
	return func() tea.Msg {
		for {
			ev, ok := <-m.streamCh
			if !ok {
				return streamDoneMsg{}
			}
			switch ev.Type {
			case "stream_chunk":
				if ev.Content != "" {
					return streamChunkMsg{content: ev.Content}
				}
			case "stream_tool_execution":
				return streamToolMsg{name: ev.ToolName, status: ev.Status, durationMs: ev.DurationMs}
			case "stream_end":
				return streamDoneMsg{}
			case "error", "stream_error":
				msg := ev.Error
				if msg == "" {
					msg = ev.Message
				}
				return streamDoneMsg{err: fmt.Errorf("%s", msg)}
			}
			// unknown event types: keep looping
		}
	}
}

// ── message helpers ────────────────────────────────────────────────────────

func (m *chatModel) appendToLastAssistant(content string) {
	for i := len(m.messages) - 1; i >= 0; i-- {
		if m.messages[i].kind == kindAssistant {
			m.messages[i].content += content
			return
		}
	}
}

func (m *chatModel) upsertTool(name, status string, durationMs int) {
	// Update existing "running" entry for this tool
	for i := len(m.messages) - 1; i >= 0; i-- {
		if m.messages[i].kind == kindTool && m.messages[i].toolName == name && m.messages[i].toolStatus == "running" {
			m.messages[i].toolStatus = status
			m.messages[i].durationMs = durationMs
			return
		}
	}
	m.messages = append(m.messages, chatMessage{
		kind:       kindTool,
		toolName:   name,
		toolStatus: "running",
	})
}

// ── rendering ─────────────────────────────────────────────────────────────

func (m *chatModel) renderMessages() string {
	var parts []string
	for _, msg := range m.messages {
		parts = append(parts, m.renderMessage(msg))
	}
	if m.streaming {
		parts = append(parts, m.spinner.View()+" "+
			lipgloss.NewStyle().Foreground(styles.ColorMuted).Render("generating..."))
	}
	return strings.Join(parts, "\n\n")
}

func (m *chatModel) renderMessage(msg chatMessage) string {
	switch msg.kind {
	case kindLogo:
		return renderLogo(m.version, m.model)

	case kindUser:
		label := styles.StyleUserMsg.
			Background(styles.ColorBorder).
			Padding(0, 1).
			Render("You")
		return label + "  " + msg.content

	case kindAssistant:
		label := styles.StyleAssistantLabel.Render("◆ ABAPer")
		if msg.content == "" {
			return label
		}
		rendered := msg.content
		if m.renderer != nil {
			if r, err := m.renderer.Render(msg.content); err == nil {
				rendered = strings.TrimRight(r, "\n")
			}
		}
		return label + "\n" + rendered

	case kindTool:
		icon := lipgloss.NewStyle().Foreground(styles.ColorWarning).Render("⚙")
		statusStr := lipgloss.NewStyle().Foreground(styles.ColorMuted).Render("● running")
		switch msg.toolStatus {
		case "completed":
			icon = lipgloss.NewStyle().Foreground(styles.ColorSuccess).Render("✓")
			statusStr = lipgloss.NewStyle().Foreground(styles.ColorSuccess).
				Render(fmt.Sprintf("✓ %.1fs", float64(msg.durationMs)/1000))
		case "failed":
			icon = lipgloss.NewStyle().Foreground(styles.ColorError).Render("✗")
			statusStr = lipgloss.NewStyle().Foreground(styles.ColorError).Render("✗ failed")
		}
		name := lipgloss.NewStyle().Foreground(styles.ColorMuted).Render(msg.toolName)
		return "  " + icon + "  " + name + "  " + statusStr

	case kindSystem:
		return styles.StyleMuted.Render("  " + msg.content)

	case kindError:
		return styles.StyleError.Render("  ✗ " + msg.content)
	}
	return ""
}


func (m *chatModel) renderStatusBar() string {
	left := lipgloss.NewStyle().Foreground(styles.ColorAccent).Render("◉")
	left += styles.StyleMuted.Render(" connected")

	if m.streaming {
		right := styles.StyleWarning.Render("streaming  ·  Ctrl+C to cancel")
		gap := m.width - lipgloss.Width(left) - lipgloss.Width(right) - 2
		if gap < 1 {
			gap = 1
		}
		return left + strings.Repeat(" ", gap) + right
	}

	hint := styles.StyleMuted.Render("[↑] history  [/] commands  [?] help")
	gap := m.width - lipgloss.Width(left) - lipgloss.Width(hint) - 2
	if gap < 1 {
		gap = 1
	}
	return left + strings.Repeat(" ", gap) + hint
}

func (m *chatModel) renderInput() string {
	if m.streaming {
		return styles.StyleMuted.Render("  " + m.spinner.View() + " thinking...  (Ctrl+C to cancel)")
	}
	prompt := lipgloss.NewStyle().Foreground(styles.ColorAccent).Render("❯ ")
	return prompt + m.textarea.View()
}

// ── View ───────────────────────────────────────────────────────────────────

func (m *chatModel) View() string {
	sep := lipgloss.NewStyle().Foreground(styles.ColorBorder).
		Render(strings.Repeat("─", m.width))

	if m.slashOpen && m.slashMenu != nil {
		return lipgloss.JoinVertical(lipgloss.Left,
			m.viewport.View(),
			m.renderStatusBar(),
			m.slashMenu.View(),
			sep,
			m.renderInput(),
		)
	}

	return lipgloss.JoinVertical(lipgloss.Left,
		m.viewport.View(),
		m.renderStatusBar(),
		sep,
		m.renderInput(),
	)
}

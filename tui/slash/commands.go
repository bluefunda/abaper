package slash

import tea "github.com/charmbracelet/bubbletea"

// ClearMsg tells the chat model to clear conversation history.
type ClearMsg struct{}

// QuitMsg tells the app to exit.
type QuitMsg struct{}

// HelpMsg tells the chat model to show help text.
type HelpMsg struct{}

// UnknownCmdMsg is returned when a slash command is not found.
type UnknownCmdMsg struct{ Name string }

// Command is a slash command entry.
type Command struct {
	Name        string
	Description string
	Aliases     []string
	Handler     func(args []string) tea.Cmd
}

// Registry is the list of all registered slash commands.
var Registry = []Command{
	{
		Name:        "help",
		Description: "Show all commands",
		Handler:     func(_ []string) tea.Cmd { return func() tea.Msg { return HelpMsg{} } },
	},
	{
		Name:        "clear",
		Description: "Clear conversation history",
		Handler:     func(_ []string) tea.Cmd { return func() tea.Msg { return ClearMsg{} } },
	},
	{
		Name:        "compact",
		Description: "Summarize & compress context",
		Handler:     func(_ []string) tea.Cmd { return nil },
	},
	{
		Name:        "object",
		Description: "Load ABAP object from SAP",
		Handler:     func(_ []string) tea.Cmd { return nil },
	},
	{
		Name:        "activate",
		Description: "Activate current object",
		Handler:     func(_ []string) tea.Cmd { return nil },
	},
	{
		Name:        "transport",
		Description: "Add to transport request",
		Handler:     func(_ []string) tea.Cmd { return nil },
	},
	{
		Name:        "diff",
		Description: "Show full diff view",
		Handler:     func(_ []string) tea.Cmd { return nil },
	},
	{
		Name:        "review",
		Description: "Code review mode",
		Handler:     func(_ []string) tea.Cmd { return nil },
	},
	{
		Name:        "test",
		Description: "Generate ABAP unit tests",
		Handler:     func(_ []string) tea.Cmd { return nil },
	},
	{
		Name:        "profile",
		Description: "Switch SAP system profile",
		Handler:     func(_ []string) tea.Cmd { return nil },
	},
	{
		Name:        "model",
		Description: "Switch LLM model",
		Handler:     func(_ []string) tea.Cmd { return nil },
	},
	{
		Name:        "cost",
		Description: "Show token usage + cost",
		Handler:     func(_ []string) tea.Cmd { return nil },
	},
	{
		Name:        "settings",
		Description: "Open settings",
		Handler:     func(_ []string) tea.Cmd { return nil },
	},
	{
		Name:        "logout",
		Description: "Log out",
		Handler:     func(_ []string) tea.Cmd { return nil },
	},
	{
		Name:        "quit",
		Description: "Exit ABAPer",
		Aliases:     []string{"q", "exit"},
		Handler:     func(_ []string) tea.Cmd { return func() tea.Msg { return QuitMsg{} } },
	},
}

// Lookup finds a command by name or alias. Returns nil if not found.
func Lookup(name string) *Command {
	for i := range Registry {
		c := &Registry[i]
		if c.Name == name {
			return c
		}
		for _, alias := range c.Aliases {
			if alias == name {
				return c
			}
		}
	}
	return nil
}

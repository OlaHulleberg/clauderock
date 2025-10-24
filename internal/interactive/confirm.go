package interactive

import (
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

var (
	warningStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("9"))
	promptStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("11"))
	mutedStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("8"))
)

// confirmModel is the Bubbletea model for confirmation dialogs
type confirmModel struct {
	title       string
	message     string
	details     []string
	textInput   textinput.Model
	confirmed   bool
	cancelled   bool
	requireYes  bool
}

// Confirm shows a confirmation dialog that requires typing "yes" to confirm
func Confirm(title string, message string, details []string) (bool, error) {
	ti := textinput.New()
	ti.Placeholder = "yes/no"
	ti.Focus()
	ti.CharLimit = 10
	ti.Width = 20

	m := confirmModel{
		title:      title,
		message:    message,
		details:    details,
		textInput:  ti,
		requireYes: true,
	}

	p := tea.NewProgram(m)
	finalModel, err := p.Run()
	if err != nil {
		return false, err
	}

	result := finalModel.(confirmModel)
	if result.cancelled {
		return false, nil
	}

	return result.confirmed, nil
}

func (m confirmModel) Init() tea.Cmd {
	return textinput.Blink
}

func (m confirmModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyEsc, tea.KeyCtrlC:
			m.cancelled = true
			return m, tea.Quit

		case tea.KeyEnter:
			input := strings.ToLower(strings.TrimSpace(m.textInput.Value()))
			if input == "yes" {
				m.confirmed = true
				return m, tea.Quit
			} else if input == "no" || input == "n" {
				m.cancelled = true
				return m, tea.Quit
			}
			// Invalid input, continue
		}
	}

	m.textInput, cmd = m.textInput.Update(msg)
	return m, cmd
}

func (m confirmModel) View() string {
	if m.cancelled || m.confirmed {
		return ""
	}

	var b strings.Builder

	// Title with warning symbol
	b.WriteString(warningStyle.Render("⚠️  " + m.title))
	b.WriteString("\n\n")

	// Main message
	b.WriteString(m.message)
	b.WriteString("\n\n")

	// Details
	if len(m.details) > 0 {
		for _, detail := range m.details {
			b.WriteString(mutedStyle.Render("  " + detail))
			b.WriteString("\n")
		}
		b.WriteString("\n")
	}

	// Prompt
	b.WriteString(promptStyle.Render("Are you sure you want to continue?"))
	b.WriteString(" ")
	b.WriteString(mutedStyle.Render("(yes/no)"))
	b.WriteString("\n")
	b.WriteString(m.textInput.View())
	b.WriteString("\n\n")

	// Help
	b.WriteString(mutedStyle.Render("Type 'yes' to confirm or 'no' to cancel • Esc to cancel"))

	return b.String()
}

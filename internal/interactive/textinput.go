package interactive

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
)

// textInputModel is the Bubbletea model for text input
type textInputModel struct {
	title      string
	example    string
	textInput  textinput.Model
	value      string
	quitting   bool
	cancelled  bool
}

// PromptTextInput provides a reusable interactive text input with example text
func PromptTextInput(title, placeholder, example string) (string, error) {
	// Initialize text input
	ti := textinput.New()
	ti.Placeholder = placeholder
	ti.Focus()
	ti.CharLimit = 200
	ti.Width = 60

	m := textInputModel{
		title:     title,
		example:   example,
		textInput: ti,
	}

	p := tea.NewProgram(m)
	finalModel, err := p.Run()
	if err != nil {
		return "", err
	}

	result := finalModel.(textInputModel)
	if result.cancelled {
		return "", fmt.Errorf("input cancelled")
	}

	return result.value, nil
}

// Init initializes the model
func (m textInputModel) Init() tea.Cmd {
	return textinput.Blink
}

// Update handles key presses and updates the model
func (m textInputModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyEsc, tea.KeyCtrlC:
			m.quitting = true
			m.cancelled = true
			return m, tea.Quit

		case tea.KeyEnter:
			m.value = strings.TrimSpace(m.textInput.Value())
			m.quitting = true
			return m, tea.Quit
		}
	}

	m.textInput, cmd = m.textInput.Update(msg)
	return m, cmd
}

// View renders the UI
func (m textInputModel) View() string {
	if m.quitting {
		return ""
	}

	var b strings.Builder

	// Title
	b.WriteString(titleStyle.Render(m.title))
	b.WriteString("\n")

	// Example text if provided
	if m.example != "" {
		b.WriteString(helpStyle.Render(fmt.Sprintf("(e.g., %s)", m.example)))
		b.WriteString("\n")
	}

	// Input
	b.WriteString(m.textInput.View())
	b.WriteString("\n\n")

	// Help text
	b.WriteString(helpStyle.Render("Enter: confirm • Esc: cancel"))

	return b.String()
}

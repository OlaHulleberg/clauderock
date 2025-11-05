package interactive

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

const (
	defaultInputCharLimit = 100
	defaultInputWidth     = 60
	defaultSelectorWidth  = 80
	defaultSelectorHeight = 20
	maxVisibleOptions     = 10
)

var (
	titleStyle    = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("12"))
	selectedStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("10"))
	normalStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("7"))
	headerStyle   = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("14")).Underline(true)
	helpStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("8"))
	countStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("8")).Italic(true)
)

// SelectOption represents an option in the selector
type SelectOption struct {
	ID       string // The value to return when selected
	Display  string // The text to display
	IsHeader bool   // If true, this is a non-selectable header
}

// selectorModel is the Bubbletea model for real-time selection
type selectorModel struct {
	title       string
	placeholder string
	textInput   textinput.Model
	options     []SelectOption
	filtered    []SelectOption
	cursor      int
	selected    string
	width       int
	height      int
	quitting    bool
	cancelled   bool
}

// InteractiveSelect provides a reusable interactive selector with real-time filtering
func InteractiveSelect(title, placeholder string, options []SelectOption, currentValue string) (string, error) {
	// Initialize text input
	ti := textinput.New()
	ti.Placeholder = placeholder
	ti.Focus()
	ti.CharLimit = defaultInputCharLimit
	ti.Width = defaultInputWidth

	// Find initial cursor position (skip headers)
	cursor := 0
	for i, opt := range options {
		if !opt.IsHeader && opt.ID == currentValue {
			cursor = i
			break
		}
	}

	m := selectorModel{
		title:       title,
		placeholder: placeholder,
		textInput:   ti,
		options:     options,
		filtered:    options,
		cursor:      cursor,
		width:       defaultSelectorWidth,
		height:      defaultSelectorHeight,
	}

	// Ensure cursor starts on a non-header item
	m.moveCursorToNearestSelectableOption()

	p := tea.NewProgram(m)
	finalModel, err := p.Run()
	if err != nil {
		return "", err
	}

	result := finalModel.(selectorModel)
	if result.cancelled {
		return "", fmt.Errorf("selection cancelled")
	}

	return result.selected, nil
}

// moveCursorToNearestSelectableOption moves cursor to nearest non-header item
func (m *selectorModel) moveCursorToNearestSelectableOption() {
	if m.cursor >= len(m.filtered) {
		m.cursor = len(m.filtered) - 1
	}
	if m.cursor < 0 {
		m.cursor = 0
	}

	// Try moving forward to find non-header
	for m.cursor < len(m.filtered) && m.filtered[m.cursor].IsHeader {
		m.cursor++
	}

	// If all forward items are headers, try backwards
	if m.cursor >= len(m.filtered) {
		m.cursor = len(m.filtered) - 1
		for m.cursor > 0 && m.filtered[m.cursor].IsHeader {
			m.cursor--
		}
	}
}

// Init initializes the model
func (m selectorModel) Init() tea.Cmd {
	return textinput.Blink
}

// Update handles key presses and updates the model
func (m selectorModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyEsc, tea.KeyCtrlC:
			m.quitting = true
			m.cancelled = true
			return m, tea.Quit

		case tea.KeyEnter:
			if len(m.filtered) > 0 && !m.filtered[m.cursor].IsHeader {
				m.selected = m.filtered[m.cursor].ID
				m.quitting = true
				return m, tea.Quit
			}

		case tea.KeyUp:
			if m.cursor > 0 {
				m.cursor--
				// Skip headers when moving up
				for m.cursor > 0 && m.filtered[m.cursor].IsHeader {
					m.cursor--
				}
			}

		case tea.KeyDown:
			if m.cursor < len(m.filtered)-1 {
				m.cursor++
				// Skip headers when moving down
				for m.cursor < len(m.filtered)-1 && m.filtered[m.cursor].IsHeader {
					m.cursor++
				}
			}

		default:
			// Update text input
			m.textInput, cmd = m.textInput.Update(msg)

			// Filter options in real-time
			m.filtered = filterOptions(m.options, m.textInput.Value())

			// Reset cursor if out of bounds and ensure it's on a selectable item
			m.moveCursorToNearestSelectableOption()

			return m, cmd
		}
	}

	m.textInput, cmd = m.textInput.Update(msg)
	return m, cmd
}

// View renders the UI
func (m selectorModel) View() string {
	if m.quitting {
		return ""
	}

	var b strings.Builder

	// Title and input
	b.WriteString(titleStyle.Render(m.title))
	b.WriteString("\n")
	b.WriteString(m.textInput.View())
	b.WriteString("\n\n")

	// Show filtered results count
	b.WriteString(countStyle.Render(fmt.Sprintf("Showing %d of %d options", len(m.filtered), len(m.options))))
	b.WriteString("\n\n")

	// Render filtered list
	start := m.cursor - maxVisibleOptions/2
	if start < 0 {
		start = 0
	}
	end := start + maxVisibleOptions
	if end > len(m.filtered) {
		end = len(m.filtered)
		start = end - maxVisibleOptions
		if start < 0 {
			start = 0
		}
	}

	for i := start; i < end; i++ {
		option := m.filtered[i]

		if option.IsHeader {
			// Render headers with special style
			b.WriteString(headerStyle.Render(option.Display))
		} else if i == m.cursor {
			b.WriteString(selectedStyle.Render("> " + option.Display))
		} else {
			b.WriteString(normalStyle.Render("  " + option.Display))
		}
		b.WriteString("\n")
	}

	// Help text
	b.WriteString("\n")
	b.WriteString(helpStyle.Render("↑/↓: navigate • Enter: select • Esc: cancel"))

	return b.String()
}

// filterOptions filters options based on search term
func filterOptions(options []SelectOption, searchTerm string) []SelectOption {
	if searchTerm == "" {
		return options
	}

	searchLower := strings.ToLower(searchTerm)
	var filtered []SelectOption
	var currentHeader *SelectOption
	inRecommendedSection := false

	for _, option := range options {
		if option.IsHeader {
			// Track if we're entering the RECOMMENDED section
			if option.Display == recommendedSectionHeader {
				inRecommendedSection = true
				currentHeader = nil
				continue
			}

			// Exit RECOMMENDED section when we hit another header
			if inRecommendedSection {
				inRecommendedSection = false
			}

			// Keep track of current header
			currentHeader = &option
			continue
		}

		// Skip items in RECOMMENDED section during search
		if inRecommendedSection {
			continue
		}

		// Match against ID or Display (case-insensitive)
		if strings.Contains(strings.ToLower(option.ID), searchLower) ||
			strings.Contains(strings.ToLower(option.Display), searchLower) {
			// Add the header before the first match in this section
			if currentHeader != nil {
				filtered = append(filtered, *currentHeader)
				currentHeader = nil // Only add header once per section
			}
			filtered = append(filtered, option)
		}
	}

	return filtered
}

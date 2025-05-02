package main

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/atotto/clipboard"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// tickMsg signals cursor blink toggles.
type tickMsg struct{}

// thinkMsg drives AI-thinking animation.
type thinkMsg struct{}

// tab represents a single chat tab.
type tab struct {
	title    string
	messages []string
	input    string
	thinking bool
	dots     int
}

// model holds UI state, including tabs, sidebar visibility, and terminal size.
type model struct {
	tabs          []tab
	currentTab    int
	blink         bool
	lastKey       string
	showSidebar   bool
	width, height int
}

// initialModel sets up a single default tab with sidebar visible.
func initialModel() model {
	t1 := tab{title: "Tab 1", messages: []string{"Welcome to Retro Chat Terminal!"}}
	return model{tabs: []tab{t1}, currentTab: 0, blink: true, showSidebar: true}
}

// Init enters alternate screen, enables mouse, and starts cursor blinking.
func (m model) Init() tea.Cmd {
	return tea.Batch(enterAltScreen, tea.EnableMouseAllMotion, blinkCmd())
}

// blinkCmd toggles the cursor every 500ms.
func blinkCmd() tea.Cmd {
	return tea.Tick(500*time.Millisecond, func(t time.Time) tea.Msg { return tickMsg{} })
}

// thinkCmd animates ellipsis then issues thinkMsg.
func thinkCmd() tea.Cmd {
	return tea.Tick(300*time.Millisecond, func(t time.Time) tea.Msg { return thinkMsg{} })
}

// enterAltScreen switches to alternate screen buffer.
var enterAltScreen = tea.EnterAltScreen

// Update handles events: key presses, mouse clicks, ticks, resize.
func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	current := &m.tabs[m.currentTab]
	switch msg := msg.(type) {
	case tea.KeyMsg:
		// Toggle sidebar: Ctrl+Shift+T
		if msg.Type == tea.KeyRunes && msg.String() == "ctrl+shift+t" {
			m.showSidebar = !m.showSidebar
			return m, nil
		}
		// Navigate tabs with arrows when sidebar visible
		if m.showSidebar {
			switch msg.String() {
			case "up":
				m.currentTab = (m.currentTab - 1 + len(m.tabs)) % len(m.tabs)
				return m, nil
			case "down":
				m.currentTab = (m.currentTab + 1) % len(m.tabs)
				return m, nil
			}
		}
		// Vim-style double 'd' to close a tab
		switch msg.String() {
		case "d":
			if m.lastKey == "d" && len(m.tabs) > 1 {
				idx := m.currentTab
				m.tabs = append(m.tabs[:idx], m.tabs[idx+1:]...)
				if idx >= len(m.tabs) {
					m.currentTab = len(m.tabs) - 1
				}
			}
			m.lastKey = "d"
			return m, nil
		case "ctrl+t": // new tab
			n := len(m.tabs) + 1
			m.tabs = append(m.tabs, tab{title: fmt.Sprintf("Tab %d", n), messages: []string{"New tab created."}})
			m.currentTab = len(m.tabs) - 1
			m.lastKey = ""
			return m, nil
		case "ctrl+c", "ctrl+q": // quit
			return m, tea.Quit
		case "ctrl+l": // clear
			current.messages = nil
			current.input = ""
			m.lastKey = ""
			return m, nil
		case "ctrl+v": // paste
			if clip, err := clipboard.ReadAll(); err == nil {
				current.input += clip
			}
			m.lastKey = ""
			return m, nil
		case "enter": // send
			current.messages = append(current.messages, fmt.Sprintf("You: %s", current.input))
			current.input = ""
			current.thinking = true
			current.dots = 0
			m.lastKey = ""
			return m, thinkCmd()
		case "backspace":
			if len(current.input) > 0 {
				current.input = current.input[:len(current.input)-1]
			}
			m.lastKey = ""
			return m, nil
		}
		m.lastKey = ""
		return m, nil

	case tea.MouseMsg:
		if current.thinking {
			return m, nil
		}
		// Sidebar click area when visible
		if m.showSidebar && msg.Type == tea.MouseLeft {
			sidebarW := 16
			row := msg.Y - 1 // account for padding
			if msg.X < sidebarW && row >= 0 && row < len(m.tabs) {
				// detect [x] region
				titleLen := len(m.tabs[row].title)
				xStart := 2 + titleLen
				if msg.X >= xStart && msg.X < xStart+3 && len(m.tabs) > 1 {
					// close tab
					m.tabs = append(m.tabs[:row], m.tabs[row+1:]...)
					if m.currentTab >= len(m.tabs) {
						m.currentTab = len(m.tabs) - 1
					}
				} else {
					// switch tab
					m.currentTab = row
				}
			}
			return m, nil
		}
		return m, nil

	case tickMsg:
		m.blink = !m.blink
		return m, blinkCmd()

	case thinkMsg:
		if current.thinking {
			if current.dots < 3 {
				current.dots++
				return m, thinkCmd()
			}
			resp := generateAIResponse(current.messages[len(current.messages)-1])
			current.messages = append(current.messages, fmt.Sprintf("AI: %s", resp))
			current.thinking = false
			current.dots = 0
		}
		return m, nil

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil
	}
	return m, nil
}

// View renders sidebar, chat panel, and nav bar horizontally.
func (m model) View() string {
	// Styles
	sidebarStyle := lipgloss.NewStyle().
		Width(16).
		Border(lipgloss.HiddenBorder()).
		Padding(1).
		Background(lipgloss.Color("#000000")).
		Foreground(lipgloss.Color("#00FF00"))

	text := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#00FF00")).
		Background(lipgloss.Color("#000000"))

	// Build sidebar if shown
	sidebar := ""
	if m.showSidebar {
		sb := []string{}
		for i, t := range m.tabs {
			marker := " "
			if i == m.currentTab {
				marker = ">"
			}
			line := fmt.Sprintf("%s %s [x]", marker, t.title)
			// add bottom border
			line = lipgloss.NewStyle().BorderBottom(true).BorderForeground(lipgloss.Color("#00FF00")).Render(line)
			sb = append(sb, line)
		}
		sidebar = sidebarStyle.Render(strings.Join(sb, "\n"))
	}

	// Chat panel style
	borderStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("#00FF00")).
		Padding(1, 2).
		Background(lipgloss.Color("#000000"))

	// Build chat content
	current := m.tabs[m.currentTab]
	content := ""
	for _, line := range current.messages {
		content += text.Render(line) + "\n"
	}
	if current.thinking {
		dots := strings.Repeat(".", current.dots)
		content += text.Render("AI is thinking"+dots) + "\n"
	}
	cursor := ""
	if m.blink && !current.thinking {
		cursor = "_"
	}
	content += text.Render("> " + current.input + cursor)

	// Chat dimensions
	head := 2
	height := m.height - head - 2
	width := m.width - func() int {
		if m.showSidebar {
			return 16 + 4
		}
		return 4
	}()

	chat := borderStyle.Width(width).Height(height).Render(content)

	// Combine
	panel := chat
	if m.showSidebar {
		panel = lipgloss.JoinHorizontal(lipgloss.Top, sidebar, chat)
	}

	// Nav hints
	nav := text.Faint(true).Align(lipgloss.Center).Width(m.width).
		Render("Tab: Ctrl+T | Toggle Sidebar: Ctrl+Shift+T | Next: Ctrl+I/Down | Prev: Ctrl+J/Up | Close: dd or Click [x] | Enter: Send | Ctrl+L: Clear | Ctrl+V/Paste | Ctrl+Q/C: Quit")

	return panel + "\n" + nav
}

// generateAIResponse returns a stub reply.
func generateAIResponse(_ string) string {
	return "wow that's crazy haha"
}

func main() {
	p := tea.NewProgram(initialModel())
	if err := p.Start(); err != nil {
		fmt.Println("Error:", err)
		os.Exit(1)
	}
}

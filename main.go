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

// Messages for internal events
type tickMsg struct{}
type thinkMsg struct{}
type pasteTickMsg struct{}

// tab represents a single chat tab.
type tab struct {
	title    string
	messages []string
	input    string
	thinking bool
	dots     int
}

// model holds UI state, including paste queue and sidebar.
type model struct {
	tabs          []tab
	currentTab    int
	blink         bool
	lastKey       string
	showSidebar   bool
	width, height int
	// paste animation
	pasteQueue   []string
	pasteRunning bool
}

// initialModel sets up a default tab with sidebar shown.
func initialModel() model {
	t1 := tab{title: "Tab 1", messages: []string{"Welcome to Retro Chat Terminal!"}}
	return model{tabs: []tab{t1}, currentTab: 0, blink: true, showSidebar: true}
}

// Init enters alt screen, enables mouse, starts blink.
func (m model) Init() tea.Cmd {
	return tea.Batch(enterAltScreen, tea.EnableMouseAllMotion, blinkCmd())
}

// blinkCmd toggles cursor every 500ms.
func blinkCmd() tea.Cmd {
	return tea.Tick(500*time.Millisecond, func(t time.Time) tea.Msg { return tickMsg{} })
}

// thinkCmd animates ellipsis then issues thinkMsg.
func thinkCmd() tea.Cmd {
	return tea.Tick(300*time.Millisecond, func(t time.Time) tea.Msg { return thinkMsg{} })
}

// pasteTick schedules next pasteTickMsg when queue non-empty.
func pasteTick() tea.Cmd {
	return tea.Tick(200*time.Millisecond, func(t time.Time) tea.Msg { return pasteTickMsg{} })
}

// enterAltScreen switches to alternate buffer.
var enterAltScreen = tea.EnterAltScreen

// Update handles events and animations.
func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	current := &m.tabs[m.currentTab]

	switch msg := msg.(type) {
	case tea.KeyMsg:
		s := msg.String()
		// Vim quit
		if s == "q" {
			return m, tea.Quit
		}
		// Toggle sidebar
		if s == "z" {
			m.showSidebar = !m.showSidebar
			return m, nil
		}
		// navigate tabs
		if m.showSidebar {
			if s == "j" {
				m.currentTab = (m.currentTab + 1) % len(m.tabs)
				return m, nil
			}
			if s == "k" {
				m.currentTab = (m.currentTab - 1 + len(m.tabs)) % len(m.tabs)
				return m, nil
			}
		}
		// multi-key
		switch s {
		case "g":
			m.lastKey = "g"
			return m, nil
		case "t":
			if m.lastKey == "g" {
				m.currentTab = (m.currentTab + 1) % len(m.tabs)
			}
			m.lastKey = ""
			return m, nil
		case "T": // new tab
			n := len(m.tabs) + 1
			m.tabs = append(m.tabs, tab{title: fmt.Sprintf("Tab %d", n), messages: []string{"New tab created."}})
			m.currentTab = len(m.tabs) - 1
			m.lastKey = ""
			return m, nil
		case "d": // close tab
			if m.lastKey == "d" && len(m.tabs) > 1 {
				idx := m.currentTab
				m.tabs = append(m.tabs[:idx], m.tabs[idx+1:]...)
				if idx >= len(m.tabs) {
					m.currentTab = len(m.tabs) - 1
				}
			}
			m.lastKey = "d"
			return m, nil
		case "p": // paste with animation
			if clip, err := clipboard.ReadAll(); err == nil {
				if strings.Contains(clip, "\n") {
					m.pasteQueue = strings.Split(clip, "\n")
					m.pasteRunning = true
					return m, pasteTick()
				} else {
					current.input += clip
				}
			}
			return m, nil
		case "u": // clear
			current.messages = nil
			current.input = ""
			return m, nil
		case "enter": // send
			current.messages = append(current.messages, fmt.Sprintf("You: %s", current.input))
			current.input = ""
			current.thinking = true
			current.dots = 0
			return m, thinkCmd()
		case "backspace":
			if len(current.input) > 0 {
				current.input = current.input[:len(current.input)-1]
			}
			return m, nil
		}
		// default input
		if r := msg.Runes; len(r) > 0 {
			current.input += string(r)
		}
		m.lastKey = ""
		return m, nil

	case pasteTickMsg:
		// animate pasteQueue
		if len(m.pasteQueue) > 0 {
			// append next line as if typed + send
			line := m.pasteQueue[0]
			m.pasteQueue = m.pasteQueue[1:]
			current.messages = append(current.messages, fmt.Sprintf("You: %s", line))
			if m.pasteRunning {
				return m, pasteTick()
			}
		}
		// done
		m.pasteRunning = false
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

// View constructs the UI.
func (m model) View() string {
	// sidebar
	sb := ""
	if m.showSidebar {
		lines := []string{}
		for i, t := range m.tabs {
			marker := " "
			if i == m.currentTab {
				marker = ">"
			}
			line := fmt.Sprintf("%s %s [x]", marker, t.title)
			lines = append(lines, lipgloss.NewStyle().BorderBottom(true).BorderForeground(lipgloss.Color("#00FF00")).Render(line))
		}
		sb = lipgloss.NewStyle().Width(16).Padding(1).Background(lipgloss.Color("#000")).Foreground(lipgloss.Color("#0f0")).Render(strings.Join(lines, "\n"))
	}
	// chat
	border := lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(lipgloss.Color("#0f0")).Padding(1, 2).Background(lipgloss.Color("#000"))
	text := lipgloss.NewStyle().Foreground(lipgloss.Color("#0f0")).Background(lipgloss.Color("#000"))
	curtab := m.tabs[m.currentTab]

	// Compute dimensions
	head := 2
	height := m.height - head - 2
	width := m.width - func() int {
		if m.showSidebar {
			return 16 + 2
		}
		return 2
	}()

	// Prepare content style to wrap text within padding
	// border.Padding left+right = 4 (2 each), so subtract from width
	innerWidth := width - 4
	contentStyle := text.Copy().Width(innerWidth).Align(lipgloss.Left)

	// Build chat lines
	chatLines := []string{}
	for _, line := range curtab.messages {
		wrapped := contentStyle.Render(line)
		chatLines = append(chatLines, wrapped)
	}
	// Thinking animation
	if curtab.thinking {
		dots := strings.Repeat(".", curtab.dots)
		wrapped := contentStyle.Render("AI is thinking" + dots)
		chatLines = append(chatLines, wrapped)
	}
	// Prompt
	cursor := ""
	if m.blink && !curtab.thinking {
		cursor = "_"
	}
	prompt := contentStyle.Render("> " + curtab.input + cursor)
	chatLines = append(chatLines, prompt)

	chatContent := strings.Join(chatLines, "\n")

	chat := border.Width(width).Height(height).Render(chatContent)
	panel := chat
	if m.showSidebar {
		panel = lipgloss.JoinHorizontal(lipgloss.Top, sb, chat)
	}
	nav := text.Faint(true).Align(lipgloss.Center).Width(m.width).Render("q: Quit | T: New tab | gt: Next | gT:Prev | dd:Close | z:Sidebar | j/k:Nav | enter:Send | p:Paste | u:Clear")
	return panel + "\n" + nav
}

func generateAIResponse(_ string) string { return "wow that's crazy haha" }

func main() {
	p := tea.NewProgram(initialModel())
	if err := p.Start(); err != nil {
		fmt.Println("Error:", err)
		os.Exit(1)
	}
}

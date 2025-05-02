package main

import (
	"fmt"
	"os"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/atotto/clipboard"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// highlight style for pasted/typed code
var codeStyle = lipgloss.NewStyle().
	Background(lipgloss.Color("#002b36")). // dark slate
	Foreground(lipgloss.Color("#93a1a1")). // light cyan
	Padding(0, 1)                          // horizontal padding

// internal tick messages
type tickMsg struct{}
type thinkMsg struct{}
type pasteTickMsg struct{}

// a chat tab
type tab struct {
	title    string
	messages []string
	input    string
	thinking bool
	dots     int
}

// model holds state (including paste buffer, insert mode, etc.)
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

	// vim-style mode
	insertMode bool
}

func initialModel() model {
	return model{
		tabs:        []tab{{title: "Tab 1", messages: []string{"Welcome!"}}},
		currentTab:  0,
		blink:       true,
		showSidebar: true,
		// seed size so we see something before WindowSizeMsg arrives
		width:      80,
		height:     24,
		insertMode: false,
	}
}

func (m model) Init() tea.Cmd {
	return tea.Batch(
		tea.EnterAltScreen,
		tea.EnableMouseAllMotion,
		blinkCmd(),
	)
}

func blinkCmd() tea.Cmd {
	return tea.Tick(500*time.Millisecond, func(t time.Time) tea.Msg { return tickMsg{} })
}

func thinkCmd() tea.Cmd {
	return tea.Tick(300*time.Millisecond, func(t time.Time) tea.Msg { return thinkMsg{} })
}

func pasteTick() tea.Cmd {
	return tea.Tick(100*time.Millisecond, func(t time.Time) tea.Msg { return pasteTickMsg{} })
}

// chunkByWidth splits s into substrings each at most width runes long.
func chunkByWidth(s string, width int) []string {
	var out []string
	runes := []rune(s)
	for len(runes) > 0 {
		n := width
		if n > len(runes) {
			n = len(runes)
		}
		out = append(out, string(runes[:n]))
		runes = runes[n:]
	}
	return out
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	current := &m.tabs[m.currentTab]

	switch msg := msg.(type) {
	case tea.KeyMsg:
		s := msg.String()

		if !m.insertMode {
			// ── NORMAL MODE ──

			// handle 'g' prefix
			if s == "g" {
				m.lastKey = "g"
				return m, nil
			}
			// 'gt' => next tab
			if s == "t" && m.lastKey == "g" {
				m.currentTab = (m.currentTab + 1) % len(m.tabs)
				m.lastKey = ""
				return m, nil
			}
			// 'gT' => previous tab
			if s == "T" && m.lastKey == "g" {
				m.currentTab = (m.currentTab - 1 + len(m.tabs)) % len(m.tabs)
				m.lastKey = ""
				return m, nil
			}

			// other normal-mode keys
			switch s {
			case "q":
				return m, tea.Quit
			case "i":
				m.insertMode = true
				return m, nil
			case "p", "P", tea.KeyCtrlV.String():
				// paste in normal mode only
				if clip, err := clipboard.ReadAll(); err == nil {
					sidebarW := 0
					if m.showSidebar {
						sidebarW = 18
					}
					innerW := m.width - sidebarW - 4
					if innerW < 1 {
						innerW = 1
					}
					m.pasteQueue = chunkByWidth(clip, innerW)
					m.pasteRunning = true
					return m, pasteTick()
				}
				return m, nil
			case "z":
				m.showSidebar = !m.showSidebar
				return m, nil
			case "j":
				if m.showSidebar {
					m.currentTab = (m.currentTab + 1) % len(m.tabs)
				}
				return m, nil
			case "k":
				if m.showSidebar {
					m.currentTab = (m.currentTab - 1 + len(m.tabs)) % len(m.tabs)
				}
				return m, nil
			case "T":
				// new tab (only when not used as 'gT')
				n := len(m.tabs) + 1
				m.tabs = append(m.tabs, tab{
					title:    fmt.Sprintf("Tab %d", n),
					messages: []string{"New tab"},
				})
				m.currentTab = len(m.tabs) - 1
				return m, nil
			case "d":
				// close tab on 'dd'
				if m.lastKey == "d" && len(m.tabs) > 1 {
					idx := m.currentTab
					m.tabs = append(m.tabs[:idx], m.tabs[idx+1:]...)
					if m.currentTab >= len(m.tabs) {
						m.currentTab = len(m.tabs) - 1
					}
				}
				m.lastKey = "d"
				return m, nil
			}

			// reset any leftover 'g'
			m.lastKey = ""
			return m, nil
		}

		// ── INSERT MODE ──
		if s == "esc" {
			m.insertMode = false
			return m, nil
		}
		switch s {
		case "enter":
			current.messages = append(current.messages, "You: "+current.input)
			current.input = ""
			current.thinking = true
			current.dots = 0
			return m, thinkCmd()
		case "backspace":
			if len(current.input) > 0 {
				_, size := utf8.DecodeLastRuneInString(current.input)
				current.input = current.input[:len(current.input)-size]
			}
			return m, nil
		default:
			// insert any character, including 'p'
			if len(msg.Runes) > 0 {
				current.input += string(msg.Runes)
			}
			m.lastKey = s
			return m, nil
		}

	case pasteTickMsg:
		if len(m.pasteQueue) > 0 {
			line := m.pasteQueue[0]
			m.pasteQueue = m.pasteQueue[1:]
			if len(current.input) > 0 {
				current.input += "\n"
			}
			current.input += line
			return m, pasteTick()
		}
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
			current.messages = append(current.messages,
				"AI: "+generateAIResponse(current.messages[len(current.messages)-1]))
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

func (m model) View() string {
	// sidebar
	sb := ""
	if m.showSidebar {
		var tabs []string
		for i, t := range m.tabs {
			marker := " "
			if i == m.currentTab {
				marker = ">"
			}
			tabs = append(tabs, fmt.Sprintf("%s %s", marker, t.title))
		}
		sb = lipgloss.NewStyle().
			Width(16).Padding(1).
			Background(lipgloss.Color("#000")).
			Foreground(lipgloss.Color("#0f0")).
			Render(strings.Join(tabs, "\n"))
	}

	// compute chat area dimensions
	head := 2
	chatHeight := m.height - head - 2
	chatWidth := m.width
	if m.showSidebar {
		chatWidth -= (16 + 2)
	}
	innerW := chatWidth - 4
	if innerW < 10 {
		innerW = 10
	}

	// build chat lines
	current := m.tabs[m.currentTab]
	var chatLines []string
	style := lipgloss.NewStyle().Width(innerW).Align(lipgloss.Left)
	for _, msg := range current.messages {
		chatLines = append(chatLines, style.Render(msg))
	}
	if current.thinking {
		dots := strings.Repeat(".", current.dots)
		chatLines = append(chatLines, style.Render("AI is thinking"+dots))
	}

	// scroll
	if len(chatLines) > chatHeight {
		chatLines = chatLines[len(chatLines)-chatHeight:]
	}

	// render input
	inputStyle := codeStyle.Width(innerW).Align(lipgloss.Left)
	lines := strings.Split(current.input, "\n")
	for i, l := range lines {
		prefix := "> "
		if i > 0 {
			prefix = "  "
		}
		line := prefix + l
		if i == len(lines)-1 && m.blink {
			line += "_"
		}
		chatLines = append(chatLines, inputStyle.Render(line))
	}

	// assemble panes
	chatPane := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("#0f0")).
		Padding(1, 2).
		Background(lipgloss.Color("#000")).
		Width(chatWidth).
		Height(chatHeight).
		Render(strings.Join(chatLines, "\n"))

	panel := chatPane
	if m.showSidebar {
		panel = lipgloss.JoinHorizontal(lipgloss.Top, sb, chatPane)
	}

	// footer based on mode
	var cmds []string
	if m.insertMode {
		cmds = []string{"-- INSERT --", "Esc:Normal", "enter:Send", "backspace:Del"}
	} else {
		cmds = []string{"-- NORMAL --", "i:Insert", "q:Quit", "p:Paste", "T:New tab", "dd:Close", "gt:Next", "gT:Prev", "z:Sidebar", "j/k:Nav"}
	}
	footer := lipgloss.NewStyle().
		Faint(true).
		Align(lipgloss.Center).
		Width(m.width).
		Render(strings.Join(cmds, " | "))

	return panel + "\n" + footer
}

func generateAIResponse(_ string) string {
	return "wow that's crazy haha"
}

func main() {
	p := tea.NewProgram(
		initialModel(),
		tea.WithAltScreen(),
		tea.WithMouseAllMotion(),
	)
	if _, err := p.Run(); err != nil {
		fmt.Println("Error:", err)
		os.Exit(1)
	}
}

package main

import (
	"fmt"
	"log"
	"os"
	"time"
	"unicode/utf8"

	"github.com/Billy-Davies-2/tui-chat/pkg/clipboard"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// codeStyle highlights pasted/typed code
var codeStyle = lipgloss.NewStyle().
	Background(lipgloss.Color("#002b36")). // dark slate
	Foreground(lipgloss.Color("#93a1a1")). // light cyan
	Padding(0, 1)

// tick messages
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

// model holds TUI state
type model struct {
	tabs          []tab
	currentTab    int
	blink         bool
	lastKey       string
	showSidebar   bool
	width, height int

	pasteQueue   []string
	pasteRunning bool

	insertMode bool
}

func initialModel() model {
	return model{
		tabs:        []tab{{title: "Tab 1", messages: []string{"Welcome!"}}},
		currentTab:  0,
		blink:       true,
		showSidebar: true,
		width:       80,
		height:      24,
		insertMode:  false,
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
		log.Printf("KeyMsg: %q insertMode=%v lastKey=%q", s, m.insertMode, m.lastKey)

		if !m.insertMode {
			// NORMAL MODE

			// yy to copy last message
			if s == "y" {
				if m.lastKey == "y" && len(current.messages) > 0 {
					clipboard.WriteAll(current.messages[len(current.messages)-1])
					log.Printf("Yanked: %q", current.messages[len(current.messages)-1])
				}
				m.lastKey = ""
				return m, nil
			}

			// gt / gT for tab nav
			if s == "g" {
				m.lastKey = "g"
				return m, nil
			}
			if s == "t" && m.lastKey == "g" {
				m.currentTab = (m.currentTab + 1) % len(m.tabs)
				log.Printf("gt → tab %d", m.currentTab)
				m.lastKey = ""
				return m, nil
			}
			if s == "T" && m.lastKey == "g" {
				m.currentTab = (m.currentTab - 1 + len(m.tabs)) % len(m.tabs)
				log.Printf("gT → tab %d", m.currentTab)
				m.lastKey = ""
				return m, nil
			}

			// p / P / Ctrl+V to paste
			if s == "p" || s == "P" || msg.Type == tea.KeyCtrlV {
				log.Print("Normal: Paste requested")
				if clip, err := clipboard.ReadAll(); err == nil {
					log.Printf("Clipboard content: %q", clip)
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
					log.Printf("Pasting %d chunks", len(m.pasteQueue))
					return m, pasteTick()
				}
				return m, nil
			}

			// other normal keys…
			switch s {
			case "q":
				return m, tea.Quit
			case "i":
				m.insertMode = true
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
				n := len(m.tabs) + 1
				m.tabs = append(m.tabs, tab{title: fmt.Sprintf("Tab %d", n), messages: []string{"New tab"}})
				m.currentTab = len(m.tabs) - 1
				return m, nil
			case "d":
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
			m.lastKey = ""
			return m, nil
		}

		// INSERT MODE
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
			resp := "wow that's crazy haha"
			current.messages = append(current.messages, "AI: "+resp)
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
	// … same View() as before …
}

func main() {
	// set up logging to tui.log
	f, err := os.OpenFile("tui.log", os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		fmt.Println("Unable to open log file:", err)
		os.Exit(1)
	}
	log.SetOutput(f)
	log.SetFlags(log.LstdFlags | log.Lmicroseconds)

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

package main

import (
	"fmt"
	"log"
	"os"
	"strings"
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

// wrapText splits text into lines no longer than maxWidth.
// When a line is wrapped, subsequent lines get prefixed with indent.
func wrapText(text string, maxWidth int, indent string) string {
	if maxWidth <= 0 {
		return text
	}
	var result []string
	// first, split on existing newlines
	for _, line := range strings.Split(text, "\n") {
		if len(line) <= maxWidth {
			result = append(result, line)
			continue
		}
		// wrap long lines
		for len(line) > 0 {
			if len(line) <= maxWidth {
				result = append(result, line)
				break
			}
			// find last space within maxWidth
			cut := maxWidth
			for cut > 0 && line[cut-1] != ' ' {
				cut--
			}
			if cut == 0 {
				// no space found, hard cut
				cut = maxWidth
			}
			result = append(result, line[:cut])
			// indent the remainder
			line = indent + strings.TrimLeft(line[cut:], " ")
		}
	}
	return strings.Join(result, "\n")
}

func (m model) View() string {
	// Sidebar
	sb := ""
	if m.showSidebar {
		lines := []string{}
		for i, t := range m.tabs {
			marker := " "
			if i == m.currentTab {
				marker = ">"
			}
			line := fmt.Sprintf("%s %s [x]", marker, t.title)
			lines = append(lines, lipgloss.NewStyle().
				BorderBottom(true).
				BorderForeground(lipgloss.Color("#00FF00")).
				Render(line))
		}
		// horizontal rule under tabs
		horizontalLine := strings.Repeat("─", 12)
		lines = append(lines, lipgloss.NewStyle().
			Foreground(lipgloss.Color("#00FF00")).
			Render(horizontalLine))

		sb = lipgloss.NewStyle().
			Width(16).
			Padding(1).
			Background(lipgloss.Color("#000")).
			Foreground(lipgloss.Color("#0f0")).
			Render(strings.Join(lines, "\n"))
	}

	// Chat panel styling
	border := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("#0f0")).
		Padding(1, 2).
		Background(lipgloss.Color("#000"))

	text := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#0f0")).
		Background(lipgloss.Color("#000"))

	curtab := m.tabs[m.currentTab]

	// Compute dimensions
	head := 2
	height := m.height - head - 2
	width := m.width
	if m.showSidebar {
		width -= 16 + 2
	}

	innerWidth := width - 4
	innerWidth = max(innerWidth, 10)
	contentStyle := text.Width(innerWidth).Align(lipgloss.Left)

	// Build chat lines
	chatLines := []string{}
	for _, msg := range curtab.messages {
		msgWidth := innerWidth - 5
		msgWidth = max(msgWidth, 5)
		wrapped := wrapText(msg, msgWidth, "  ")
		for _, wl := range strings.Split(wrapped, "\n") {
			chatLines = append(chatLines, contentStyle.Render(wl))
		}
	}

	// Thinking animation
	if curtab.thinking {
		dots := strings.Repeat(".", curtab.dots)
		chatLines = append(chatLines, contentStyle.Render("AI is thinking"+dots))
	}

	// Prompt & input
	cursor := ""
	if m.blink && !curtab.thinking {
		cursor = "_"
	}
	displayInput := strings.ReplaceAll(curtab.input, "\n", "⏎\n")
	wrappedInput := wrapText("> "+displayInput, innerWidth, "  ")
	for i, line := range strings.Split(wrappedInput, "\n") {
		if i == len(strings.Split(wrappedInput, "\n"))-1 && m.blink && !curtab.thinking {
			line += cursor
		}
		chatLines = append(chatLines, contentStyle.Render(line))
	}

	chatContent := strings.Join(chatLines, "\n")
	chatPane := border.Width(width).Height(height).Render(chatContent)

	// Combine sidebar + chat
	panel := chatPane
	if m.showSidebar {
		panel = lipgloss.JoinHorizontal(lipgloss.Top, sb, chatPane)
	}

	// Footer nav
	nav := text.Faint(true).Align(lipgloss.Center).Width(m.width).
		Render("q: Quit | T: New tab | gt: Next | gT:Prev | dd:Close | z:Sidebar | j/k:Nav | enter:Send | p:Paste | u:Clear")

	return panel + "\n" + nav
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

	// seed the in-memory clipboard from the OS
	clipboard.Init()

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

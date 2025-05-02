# TUI Chat Terminal

A retro-style terminal chat application written in Go, powered by [Bubble Tea](https://github.com/charmbracelet/bubbletea) and [Lip Gloss](https://github.com/charmbracelet/lipgloss).

Inspired heavily by [cool-retro-term](https://github.com/Swordfish90/cool-retro-term)

## Features

* **Multi‑tab** chat interface with a sidebar for navigation
* **Vim‑style** Normal/Insert modes
* **In‑memory clipboard** with `yy` (yank) and `p` (paste) operations
* **Optional OS‑clipboard seeding** (via `golang.design/x/clipboard`) at startup
* **Retro green‑on‑black** color scheme with lipgloss styling
* **Automatic scrolling** of chat pane when content overflows
* **Blinking cursor** and “AI thinking” animation
* Cross‑platform (macOS, Linux, Windows) with no external binary dependencies

## Requirements

* Go **1.24** or later
* A terminal supporting ANSI escape codes

## Installation

1. **Clone the repo**

   ```bash
   git clone https://github.com/Billy-Davies-2/tui-chat.git
   cd tui-chat
   ```

2. **Fetch dependencies**

   ```bash
   go mod tidy
   ```

3. **Build**

   ```bash
   go build -o tui-chat main.go
   ```

## Usage

Run the built binary:

```bash
./tui-chat
```

## Keybindings

### Normal Mode

| Key       | Action                            |
| --------- | --------------------------------- |
| `i`       | Enter **Insert** mode             |
| `q`       | Quit                              |
| `yy`      | Yank (copy) last chat message     |
| `p` / `P` | Paste yanked text into input      |
| `gt`      | Next tab                          |
| `gT`      | Previous tab                      |
| `T`       | Create new tab                    |
| `dd`      | Close current tab                 |
| `j` / `k` | Navigate tabs (when sidebar open) |
| `z`       | Toggle sidebar                    |

### Insert Mode

| Key         | Action                        |
| ----------- | ----------------------------- |
| `Esc`       | Return to Normal mode         |
| `Enter`     | Send current input as message |
| `Backspace` | Delete last character         |
| Any other   | Insert typed character        |

## Roadmap

- [ ] Flesh out chat client and AI endpoints.
- [x] mouse support
- [ ] mouse support in chat terminal 
- [ ] more complex vim bindings.
- [ ] parallax or some sort of cool backgrounds.


## Contributing

Contributions, issues and feature requests are welcome! Feel free to fork the project and submit a pull request.

## License

This project is licensed under the **MIT License**. See the [LICENSE](LICENSE) file for details.


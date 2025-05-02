// pkg/clipboard/clipboard.go
package clipboard

import (
	"bytes"
	"fmt"
	"os/exec"
	"runtime"
)

// ReadAll returns the current clipboard contents by shelling out
// to the native OS clipboard commands.
func ReadAll() (string, error) {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("pbpaste")
	case "linux":
		// Prefer Wayland’s wl-paste, fall back to xclip
		if _, err := exec.LookPath("wl-paste"); err == nil {
			cmd = exec.Command("wl-paste")
		} else {
			cmd = exec.Command("xclip", "-selection", "clipboard", "-o")
		}
	case "windows":
		cmd = exec.Command("powershell", "-nologo", "-noprofile", "-command", "Get-Clipboard")
	default:
		return "", fmt.Errorf("unsupported platform: %s", runtime.GOOS)
	}
	out, err := cmd.Output()
	return string(out), err
}

// WriteAll sets the clipboard to the given text.
func WriteAll(text string) error {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("pbcopy")
	case "linux":
		// Prefer wl-copy, fall back to xclip
		if _, err := exec.LookPath("wl-copy"); err == nil {
			cmd = exec.Command("wl-copy")
		} else {
			cmd = exec.Command("xclip", "-selection", "clipboard")
		}
	case "windows":
		cmd = exec.Command("powershell", "-nologo", "-noprofile", "-command", "Set-Clipboard")
	default:
		return fmt.Errorf("unsupported platform: %s", runtime.GOOS)
	}
	cmd.Stdin = bytes.NewBufferString(text)
	return cmd.Run()
}

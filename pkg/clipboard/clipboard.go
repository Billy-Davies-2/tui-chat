// pkg/clipboard/clipboard.go
package clipboard

import (
	"log"
	"sync"

	xclip "golang.design/x/clipboard"
)

var (
	mu    sync.RWMutex
	store string
)

// Init pulls in the OS clipboard exactly once.
// Call this at startup to seed your in-memory buffer.
func Init() {
	// initialize the x/clipboard driver
	if err := xclip.Init(); err != nil {
		log.Printf("clipboard: x/clipboard.Init failed: %v", err)
		return
	}
	// read whatever text is on the OS clipboard now
	if data := xclip.Read(xclip.FmtText); data != nil {
		mu.Lock()
		store = string(data)
		mu.Unlock()
		log.Printf("clipboard: seeded buffer from OS clipboard: %q", store)
	}
}

// ReadAll returns the current clipboard buffer (seeded, yanked, whatever).
func ReadAll() (string, error) {
	mu.RLock()
	defer mu.RUnlock()
	return store, nil
}

// WriteAll replaces the clipboard buffer with the given text (e.g. in your 'yy' yank).
func WriteAll(text string) error {
	mu.Lock()
	store = text
	mu.Unlock()
	log.Printf("clipboard: in-mem buffer updated to: %q", text)
	return nil
}

package ui

import (
	"fmt"
	"strings"
	"sync"
	"time"
)

// Header prints the ASCII art header
func Header() {
	fmt.Println("   __             ")
	fmt.Println("  / /  ___ _    __")
	fmt.Println(" / _ \\/ _ \\ |/|/ /")
	fmt.Println("/_//_/\\___/__,__/ ")
	fmt.Println()
	fmt.Println("Ask me how to do anything in your terminal!")
}

// Spinner displays an animated spinner while work is being done
type Spinner struct {
	stop    chan bool
	message string
	wg      sync.WaitGroup
}

// NewSpinner creates a new spinner with the given message
func NewSpinner(message string) *Spinner {
	return &Spinner{
		stop:    make(chan bool),
		message: message,
	}
}

// Start begins the spinner animation
func (s *Spinner) Start() {
	frames := []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}
	s.wg.Add(1)
	go func() {
		defer s.wg.Done()
		i := 0
		for {
			select {
			case <-s.stop:
				// Clear the spinner line
				fmt.Print("\r" + strings.Repeat(" ", len(s.message)+2) + "\r")
				return
			default:
				fmt.Printf("\r%s %s", frames[i%len(frames)], s.message)
				i++
				time.Sleep(100 * time.Millisecond)
			}
		}
	}()
}

// Stop stops the spinner animation
func (s *Spinner) Stop() {
	close(s.stop)
	s.wg.Wait()
}

// TypewriterPrint prints text with a typewriter effect
func TypewriterPrint(text string) {
	for _, char := range text {
		fmt.Print(string(char))
		time.Sleep(10 * time.Millisecond)
	}
	fmt.Println()
}

// CleanResponse removes code block markers from the response
func CleanResponse(text string) string {
	text = strings.TrimSpace(text)

	// Remove triple backticks
	if strings.HasPrefix(text, "```") && strings.HasSuffix(text, "```") {
		lines := strings.SplitN(text, "\n", 2)
		firstLine := lines[0]
		if len(firstLine) > 3 {
			text = text[len(firstLine) : len(text)-3]
		} else {
			text = text[3 : len(text)-3]
		}
		text = strings.TrimSpace(text)
	} else if strings.HasPrefix(text, "`") && strings.HasSuffix(text, "`") {
		// Remove single backticks
		text = text[1 : len(text)-1]
		text = strings.TrimSpace(text)
	}

	return strings.TrimSpace(text)
}

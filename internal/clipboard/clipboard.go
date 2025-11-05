package clipboard

import (
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"

	"golang.design/x/clipboard"
)

// CopyToClipboard attempts to copy text to clipboard using various methods
func CopyToClipboard(text string) error {
	// Try the golang.design/x/clipboard library first
	if err := clipboard.Init(); err == nil {
		clipboard.Write(clipboard.FmtText, []byte(text))
		return nil
	}

	// If that fails, try platform-specific alternatives
	return copyWithFallback(text)
}

// copyWithFallback tries alternative clipboard methods
func copyWithFallback(text string) error {
	switch runtime.GOOS {
	case "linux":
		return copyLinux(text)
	case "darwin":
		return copyMacOS(text)
	case "windows":
		return copyWindows(text)
	default:
		return fmt.Errorf("clipboard not supported on %s", runtime.GOOS)
	}
}

// copyLinux tries various Linux clipboard methods
func copyLinux(text string) error {
	// Try xclip first
	if err := tryCommand("xclip", []string{"-selection", "clipboard"}, text); err == nil {
		return nil
	}

	// Try xsel
	if err := tryCommand("xsel", []string{"--clipboard", "--input"}, text); err == nil {
		return nil
	}

	// Try wl-copy (Wayland)
	if err := tryCommand("wl-copy", []string{}, text); err == nil {
		return nil
	}

	// If we're in SSH, try OSC 52 escape sequence
	if isSSH() {
		return copyWithOSC52(text)
	}

	return fmt.Errorf("no clipboard method available")
}

// copyMacOS uses pbcopy on macOS
func copyMacOS(text string) error {
	return tryCommand("pbcopy", []string{}, text)
}

// copyWindows uses clip.exe on Windows
func copyWindows(text string) error {
	return tryCommand("clip", []string{}, text)
}

// tryCommand attempts to run a command with the given text as input
func tryCommand(command string, args []string, text string) error {
	// Check if command exists
	if _, err := exec.LookPath(command); err != nil {
		return err
	}

	cmd := exec.Command(command, args...)
	cmd.Stdin = strings.NewReader(text)
	return cmd.Run()
}

// isSSH checks if we're running in an SSH session
func isSSH() bool {
	return os.Getenv("SSH_CLIENT") != "" || os.Getenv("SSH_TTY") != "" || os.Getenv("SSH_CONNECTION") != ""
}

// copyWithOSC52 uses OSC 52 escape sequence for SSH clipboard
func copyWithOSC52(text string) error {
	// OSC 52 is supported by many modern terminals including Windows Terminal
	// This allows clipboard access through SSH
	if len(text) > 100000 {
		return fmt.Errorf("text too long for OSC 52 (max ~100KB)")
	}

	// Base64 encode the text
	encoded := base64Encode(text)
	
	// Send OSC 52 escape sequence
	fmt.Printf("\033]52;c;%s\033\\", encoded)
	
	return nil
}

// Simple base64 encoding without importing encoding/base64
func base64Encode(data string) string {
	const chars = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789+/"
	
	input := []byte(data)
	var result strings.Builder
	
	for i := 0; i < len(input); i += 3 {
		var b1, b2, b3 byte
		b1 = input[i]
		if i+1 < len(input) {
			b2 = input[i+1]
		}
		if i+2 < len(input) {
			b3 = input[i+2]
		}
		
		result.WriteByte(chars[b1>>2])
		result.WriteByte(chars[((b1&0x03)<<4)|((b2&0xf0)>>4)])
		
		if i+1 < len(input) {
			result.WriteByte(chars[((b2&0x0f)<<2)|((b3&0xc0)>>6)])
		} else {
			result.WriteByte('=')
		}
		
		if i+2 < len(input) {
			result.WriteByte(chars[b3&0x3f])
		} else {
			result.WriteByte('=')
		}
	}
	
	return result.String()
}

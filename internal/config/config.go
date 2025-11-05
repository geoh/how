package config

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

var (
	configDir   string
	apiKeyFile  string
	historyFile string
)

func init() {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		homeDir = "."
	}
	configDir = filepath.Join(homeDir, ".how-cli")
	apiKeyFile = filepath.Join(configDir, ".google_api_key")
	historyFile = filepath.Join(configDir, "history.log")
}

// GetOrCreateAPIKey retrieves the API key from environment or file, or prompts for it
func GetOrCreateAPIKey(forceReenter bool) (string, error) {
	var apiKey string

	if !forceReenter {
		// Check environment variable first
		apiKey = os.Getenv("GOOGLE_API_KEY")

		// If not in environment, check the file
		if apiKey == "" {
			data, err := os.ReadFile(apiKeyFile)
			if err == nil {
				apiKey = strings.TrimSpace(string(data))
			}
		}
	}

	// If still no API key or force re-enter, prompt the user
	if apiKey == "" || forceReenter {
		// Check if stdin is a terminal
		fileInfo, _ := os.Stdin.Stat()
		if (fileInfo.Mode() & os.ModeCharDevice) == 0 {
			return "", fmt.Errorf("GOOGLE_API_KEY not found in non-interactive session")
		}

		fmt.Println("Paste your Google Gemini API key:")
		fmt.Print("API Key: ")

		reader := bufio.NewReader(os.Stdin)
		input, err := reader.ReadString('\n')
		if err != nil {
			return "", fmt.Errorf("API key input cancelled")
		}

		apiKey = strings.TrimSpace(input)
		if apiKey == "" {
			return "", fmt.Errorf("API key cannot be empty")
		}

		// Save the API key
		if err := SaveAPIKey(apiKey); err != nil {
			// Log warning but continue
			fmt.Fprintf(os.Stderr, "Warning: Could not save API key: %v\n", err)
		}
	}

	return apiKey, nil
}

// SaveAPIKey saves the API key to the config file
func SaveAPIKey(apiKey string) error {
	// Create config directory if it doesn't exist
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return err
	}

	// Write the API key to the file
	if err := os.WriteFile(apiKeyFile, []byte(apiKey), 0600); err != nil {
		return err
	}

	return nil
}

// LogHistory appends a question and commands to the history file
func LogHistory(question string, commands []string) error {
	// Create config directory if it doesn't exist
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return err
	}

	// Open file in append mode
	f, err := os.OpenFile(historyFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer f.Close()

	// Write the log entry
	timestamp := time.Now().Format("2006-01-02 15:04:05")
	if _, err := fmt.Fprintf(f, "[%s] Q: %s\nCommands:\n", timestamp, question); err != nil {
		return err
	}

	for _, cmd := range commands {
		if _, err := fmt.Fprintf(f, "%s\n", cmd); err != nil {
			return err
		}
	}

	if _, err := fmt.Fprintln(f); err != nil {
		return err
	}

	return nil
}

// ShowHistory displays the history file contents
func ShowHistory() error {
	data, err := os.ReadFile(historyFile)
	if err != nil {
		if os.IsNotExist(err) {
			fmt.Println("No history found.")
			return nil
		}
		return fmt.Errorf("error reading history file: %v", err)
	}

	fmt.Print(string(data))
	return nil
}

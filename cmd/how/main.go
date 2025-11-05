package main

import (
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/geoh/how/internal/api"
	"github.com/geoh/how/internal/clipboard"
	"github.com/geoh/how/internal/config"
	"github.com/geoh/how/internal/context"
	"github.com/geoh/how/internal/ui"
)

func main() {
	// Handle interrupts gracefully
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-sigChan
		fmt.Println("\n👋 Interrupted.")
		os.Exit(130)
	}()

	// Check for help flag or no arguments
	if len(os.Args) < 2 || hasFlag("--help") {
		ui.Header()
		printHelp()
		os.Exit(0)
	}

	// Handle --history flag
	if hasFlag("--history") {
		if err := config.ShowHistory(); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
		os.Exit(0)
	}

	// Handle --api-key flag
	if hasFlag("--api-key") {
		idx := findFlagIndex("--api-key")
		if idx != -1 && len(os.Args) > idx+1 && !strings.HasPrefix(os.Args[idx+1], "--") {
			newKey := strings.TrimSpace(os.Args[idx+1])
			if newKey == "" {
				fmt.Println("Error: API key cannot be empty.")
				os.Exit(1)
			}
			if err := config.SaveAPIKey(newKey); err != nil {
				fmt.Fprintf(os.Stderr, "Error saving API key: %v\n", err)
				os.Exit(1)
			}
			fmt.Println("Gemini API key replaced successfully.")
			os.Exit(0)
		}
	}

	// Parse flags
	silent := hasFlag("--silent")
	typeEffect := hasFlag("--type") && !silent

	// Get question from arguments (excluding flags)
	args := filterFlags(os.Args[1:])
	if len(args) == 0 {
		fmt.Println("Error: No question provided.")
		os.Exit(1)
	}
	question := strings.Join(args, " ")

	// Get or create API key
	apiKey, err := config.GetOrCreateAPIKey(false)
	if err != nil {
		fmt.Fprintf(os.Stderr, "❌ Authentication Error: %v\n", err)
		os.Exit(1)
	}

	// Gather system context
	ctx, err := context.Gather()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: Failed to gather system context: %v\n", err)
		// Continue with default values
		ctx = &context.SystemContext{
			OS:             "Unknown",
			Shell:          "Unknown",
			CurrentDir:     "Unknown",
			User:           "Unknown",
			GitRepo:        "No",
			Files:          "Unknown",
			InstalledTools: "Unknown",
		}
	}

	// Build the prompt
	prompt := fmt.Sprintf(`SYSTEM:
You are an expert, concise shell assistant. Your goal is to provide accurate, executable shell commands.

CONTEXT:
-   **OS:** %s
-   **Shell:** %s
-   **CWD:** %s
-   **User:** %s
-   **Git Repo:** %s
-   **Files (top 20):** %s
-   **Available Tools:** %s

RULES:
1.  **Primary Goal:** Generate *only* the exact, executable shell command(s) for the %s environment.
2.  **Context is Key:** Use the CONTEXT (CWD, Files, OS) to write specific, correct commands.
3.  **No Banter:** Do NOT include greetings, sign-offs, or conversational filler (e.g., "Here is the command:").
4.  **Safety:** If a command is complex or destructive (e.g., ` + "`rm -rf`, `find -delete`" + `), add a single-line comment (` + "`# ...`" + `) *after* the command explaining what it does.
5.  **Questions:** If the user asks a question (e.g., "what is ` + "`ls`" + `?"), provide a concise, one-line answer. Do not output a command.
6.  **Ambiguity:** If the request is unclear, ask a single, direct clarifying question. Start the line with ` + "`#`" + `.

REQUEST:
%s

RESPONSE:
`, ctx.OS, ctx.Shell, ctx.CurrentDir, ctx.User, ctx.GitRepo, ctx.Files, ctx.InstalledTools, ctx.Shell, question)

	// Generate response with spinner
	var spinner *ui.Spinner
	if !silent {
		spinner = ui.NewSpinner("Generating")
		spinner.Start()
	}

	text, err := api.GenerateResponse(apiKey, prompt, 3)

	if !silent && spinner != nil {
		spinner.Stop()
	}

	if err != nil {
		switch err.(type) {
		case *api.AuthError:
			fmt.Fprintf(os.Stderr, "\n❌ Error: %v\n", err)
		case *api.ContentError:
			fmt.Fprintf(os.Stderr, "\n❌ Error: %v\n", err)
		case *api.ApiTimeoutError:
			fmt.Fprintf(os.Stderr, "\n❌ Error: %v\n", err)
		case *api.ApiError:
			fmt.Fprintf(os.Stderr, "\n❌ Error: %v\n", err)
		default:
			fmt.Fprintf(os.Stderr, "\n❌ Error: %v\n", err)
		}
		os.Exit(1)
	}

	// Clean the response
	cleanedText := ui.CleanResponse(text)
	commands := strings.Split(cleanedText, "\n")

	// Filter out empty lines
	var filteredCommands []string
	for _, cmd := range commands {
		if trimmed := strings.TrimSpace(cmd); trimmed != "" {
			filteredCommands = append(filteredCommands, trimmed)
		}
	}

	if len(filteredCommands) == 0 {
		fmt.Println("⚠️ No valid commands generated.")
		os.Exit(1)
	}

	fullCommand := strings.Join(filteredCommands, "\n")

	// Print the result
	if typeEffect {
		ui.TypewriterPrint(fullCommand)
	} else {
		fmt.Println(fullCommand)
	}

	// Copy to clipboard
	if err := clipboard.CopyToClipboard(fullCommand); err != nil {
		// Only show clipboard error in verbose mode or if DISPLAY is set
		if os.Getenv("DISPLAY") != "" || os.Getenv("HOW_DEBUG") != "" {
			fmt.Fprintf(os.Stderr, "Warning: Could not copy to clipboard: %v\n", err)
		}
	}

	// Log to history
	if err := config.LogHistory(question, filteredCommands); err != nil {
		// Just log a warning, don't fail
		fmt.Fprintf(os.Stderr, "Warning: Failed to write history: %v\n", err)
	}
}

func printHelp() {
	fmt.Println("Usage: how <question> [--silent] [--history] [--type] [--help] [--api-key]")
	fmt.Println()
	fmt.Println("Options:")
	fmt.Println("  --silent      Suppress spinner and typewriter effect")
	fmt.Println("  --type        Show output with typewriter effect")
	fmt.Println("  --history     Show command/question history")
	fmt.Println("  --help        Show this help message and exit")
	fmt.Println("  --api-key     Set the Gemini API key (usage: --api-key <API_KEY>)")
}

func hasFlag(flag string) bool {
	for _, arg := range os.Args[1:] {
		if arg == flag {
			return true
		}
	}
	return false
}

func findFlagIndex(flag string) int {
	for i, arg := range os.Args {
		if arg == flag {
			return i
		}
	}
	return -1
}

func filterFlags(args []string) []string {
	var result []string
	skipNext := false

	for i, arg := range args {
		if skipNext {
			skipNext = false
			continue
		}

		if arg == "--silent" || arg == "--history" || arg == "--type" {
			continue
		}

		if arg == "--api-key" {
			// Skip this flag and the next argument (the API key value)
			if i+1 < len(args) && !strings.HasPrefix(args[i+1], "--") {
				skipNext = true
			}
			continue
		}

		result = append(result, arg)
	}

	return result
}

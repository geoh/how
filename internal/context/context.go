package context

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

// SystemContext holds information about the current system environment
type SystemContext struct {
	OS            string
	Shell         string
	CurrentDir    string
	User          string
	GitRepo       string
	Files         string
	InstalledTools string
}

// Gather collects system context information
func Gather() (*SystemContext, error) {
	ctx := &SystemContext{}

	// Get OS information
	ctx.OS = fmt.Sprintf("%s %s", runtime.GOOS, getOSVersion())

	// Get shell
	ctx.Shell = getCurrentTerminal()

	// Get current directory
	cwd, err := os.Getwd()
	if err != nil {
		cwd = "Unknown"
	}
	ctx.CurrentDir = cwd

	// Get current user
	if user := os.Getenv("USER"); user != "" {
		ctx.User = user
	} else if user := os.Getenv("USERNAME"); user != "" {
		ctx.User = user
	} else {
		ctx.User = "Unknown"
	}

	// Check if current directory is a git repository
	gitDir := filepath.Join(cwd, ".git")
	if _, err := os.Stat(gitDir); err == nil {
		ctx.GitRepo = "Yes"
	} else {
		ctx.GitRepo = "No"
	}

	// List files in current directory
	ctx.Files = listFiles(cwd)

	// Get installed tools
	ctx.InstalledTools = getInstalledTools()

	return ctx, nil
}

// getOSVersion returns the OS version/release
func getOSVersion() string {
	switch runtime.GOOS {
	case "linux":
		// Try to get kernel version
		out, err := exec.Command("uname", "-r").Output()
		if err == nil {
			return strings.TrimSpace(string(out))
		}
	case "darwin":
		// Try to get macOS version
		out, err := exec.Command("sw_vers", "-productVersion").Output()
		if err == nil {
			return strings.TrimSpace(string(out))
		}
	case "windows":
		// Try to get Windows version
		out, err := exec.Command("ver").Output()
		if err == nil {
			return strings.TrimSpace(string(out))
		}
	}
	return "Unknown"
}

// getCurrentTerminal returns the name of the current terminal/shell
func getCurrentTerminal() string {
	// Check common shell environment variables
	if shell := os.Getenv("SHELL"); shell != "" {
		return filepath.Base(shell)
	}

	// Try to get parent process name (similar to Python psutil approach)
	if runtime.GOOS == "linux" || runtime.GOOS == "darwin" {
		ppid := os.Getppid()
		// Try to read /proc/<ppid>/comm (Linux)
		if runtime.GOOS == "linux" {
			commPath := fmt.Sprintf("/proc/%d/comm", ppid)
			data, err := os.ReadFile(commPath)
			if err == nil {
				return strings.TrimSpace(string(data))
			}
		}
		// Try using ps command
		out, err := exec.Command("ps", "-p", fmt.Sprintf("%d", ppid), "-o", "comm=").Output()
		if err == nil {
			return strings.TrimSpace(string(out))
		}
	}

	return "Unknown"
}

// listFiles returns a comma-separated list of files in the directory
func listFiles(dir string) string {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return "Error listing files"
	}

	var files []string
	maxFiles := 20
	for i, entry := range entries {
		if i >= maxFiles {
			files = append(files, "...")
			break
		}
		files = append(files, entry.Name())
	}

	return strings.Join(files, ", ")
}

// getInstalledTools checks for commonly installed development tools
func getInstalledTools() string {
	tools := []string{"git", "npm", "node", "python", "docker", "pip", "go", "rustc", "cargo", "java", "mvn", "gradle"}
	var installed []string

	for _, tool := range tools {
		if _, err := exec.LookPath(tool); err == nil {
			installed = append(installed, tool)
		}
	}

	return strings.Join(installed, ", ")
}

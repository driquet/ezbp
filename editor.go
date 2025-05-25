package main

import (
	"fmt"
	"os"
	"os/exec"
	"runtime"
)

// getDefaultEditor returns the user's preferred editor.
func getDefaultEditor(editor string) string {
	if editor != "" {
		return editor
	}

	// Check environment variables in order of preference
	if editor := os.Getenv("VISUAL"); editor != "" {
		return editor
	}
	if editor := os.Getenv("EDITOR"); editor != "" {
		return editor
	}

	// Platform-specific defaults
	switch runtime.GOOS {
	case "windows":
		return "notepad"
	case "darwin":
		return "nano" // or "vim", "emacs"
	default:
		return "nano" // Linux and others
	}
}

func editContent(editor, initialContent string) (string, error) {
	// Create temporary file
	filename, err := createTempFile(initialContent)
	if err != nil {
		return "", err
	}
	defer os.Remove(filename)

	// Launch editor.
	if err := openEditor(editor, filename); err != nil {
		return "", err
	}

	// Read temporary file
	content, err := os.ReadFile(filename)
	if err != nil {
		return "", fmt.Errorf("failed to read file: %w", err)
	}
	return string(content), nil
}

// createTempFile creates a temporary file with initial content
func createTempFile(initialContent string) (string, error) {
	f, err := os.CreateTemp("", "boilerplate_*.txt")
	if err != nil {
		return "", fmt.Errorf("failed to create temp file: %w", err)
	}
	defer f.Close()

	// Write initial content if provided
	if initialContent != "" {
		if _, err := f.WriteString(initialContent); err != nil {
			os.Remove(f.Name())
			return "", fmt.Errorf("failed to write initial content: %w", err)
		}
	}

	return f.Name(), nil
}

// openEditor opens the specified file in the user's default editor
func openEditor(editor, filename string) error {
	editor = getDefaultEditor(editor)

	var cmd *exec.Cmd
	if runtime.GOOS == "windows" {
		cmd = exec.Command("cmd", "/c", editor, filename)
	} else {
		cmd = exec.Command(editor, filename)
	}

	// Connect editor to terminal
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	return cmd.Run()
}

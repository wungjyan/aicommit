package prompt

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// Confirm shows the generated commit message and asks for user confirmation.
// Returns: action ("commit", "edit", "regenerate", "quit"), edited message (if action is "edit"), error.
func Confirm(message string) (action string, editedMessage string, err error) {
	fmt.Println()
	fmt.Println("Generated commit message:")
	fmt.Println()
	fmt.Println("  " + message)
	fmt.Println()
	fmt.Println("[Enter] commit  [e] edit  [r] regenerate  [q] quit")

	reader := bufio.NewReader(os.Stdin)
	input, err := reader.ReadString('\n')
	if err != nil {
		return "", "", fmt.Errorf("failed to read input: %w", err)
	}

	input = strings.TrimSpace(strings.ToLower(input))

	switch input {
	case "", "y", "yes":
		return "commit", message, nil
	case "e", "edit":
		edited, editErr := editMessage(message)
		if editErr != nil {
			return "", "", editErr
		}
		return "commit", edited, nil
	case "r", "regenerate":
		return "regenerate", "", nil
	case "q", "quit":
		return "quit", "", nil
	default:
		fmt.Println("Invalid input. Please enter, e, r, or q.")
		return Confirm(message)
	}
}

func editMessage(original string) (string, error) {
	editor := os.Getenv("EDITOR")
	if editor == "" {
		editor = os.Getenv("VISUAL")
	}
	if editor == "" {
		editor = "vim"
	}

	tmpFile, err := os.CreateTemp("", "aicommit-*.txt")
	if err != nil {
		return "", fmt.Errorf("failed to create temp file: %w", err)
	}
	defer os.Remove(tmpFile.Name())

	if _, err := tmpFile.WriteString(original + "\n"); err != nil {
		tmpFile.Close()
		return "", fmt.Errorf("failed to write temp file: %w", err)
	}
	tmpFile.Close()

	cmd := exec.Command(editor, tmpFile.Name())
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("editor failed: %w", err)
	}

	content, err := os.ReadFile(tmpFile.Name())
	if err != nil {
		return "", fmt.Errorf("failed to read edited file: %w", err)
	}

	edited := strings.TrimSpace(string(content))
	if edited == "" {
		return original, nil
	}

	return edited, nil
}

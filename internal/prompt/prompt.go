package prompt

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"os/exec"
	"regexp"
	"strings"
)

var validTypes = map[string]bool{
	"feat": true, "fix": true, "docs": true, "style": true,
	"refactor": true, "perf": true, "test": true, "chore": true,
	"ci": true, "build": true,
}

// Conventional commit header: type[(scope)][!]: description
var commitHeaderRe = regexp.MustCompile(`^(\w+)(?:\(([^)]*)\))?(!)?:\s+(.+)$`)

// ValidateMessage checks whether a commit message conforms to Conventional Commits.
// It returns a non-nil error describing what is wrong, or nil if the message is valid.
func ValidateMessage(message string) error {
	message = strings.TrimSpace(message)
	if message == "" {
		return fmt.Errorf("commit message is empty")
	}

	// The header is the first line (before any blank line).
	lines := strings.SplitN(message, "\n", 2)
	header := strings.TrimSpace(lines[0])

	if len(header) > 72 {
		return fmt.Errorf("header is %d characters (max 72): %q", len(header), header)
	}

	m := commitHeaderRe.FindStringSubmatch(header)
	if m == nil {
		return fmt.Errorf("header does not match Conventional Commits format (<type>[optional scope]): <description>): %q", header)
	}

	msgType := m[1]
	if !validTypes[msgType] {
		return fmt.Errorf("invalid commit type %q; expected one of: feat, fix, docs, style, refactor, perf, test, chore, ci, build", msgType)
	}

	desc := m[4]
	if desc == "" {
		return fmt.Errorf("commit description is empty")
	}

	return nil
}

// Style formats prompt labels and generated messages without choosing an output
// destination.
type Style interface {
	Bold(string) string
	Highlight(string) string
}

// Confirm shows the generated commit message and asks for user confirmation.
// When valid is false (message failed ValidateMessage), the commit option is
// hidden — the user must edit or regenerate first.
// Returns: action ("commit", "edit", "regenerate", "quit"), edited message (if action is "edit"), error.
func Confirm(in io.Reader, out io.Writer, style Style, message string, valid bool) (action string, editedMessage string, err error) {
	reader := bufio.NewReader(in)
	for {
		fmt.Fprintln(out)
		fmt.Fprintln(out, style.Bold("Generated commit message:"))
		fmt.Fprintln(out)
		fmt.Fprintln(out, "  "+style.Highlight(message))
		fmt.Fprintln(out)

		if valid {
			fmt.Fprintf(out, "%s  %s  %s  %s\n",
				style.Bold("[Enter]")+" commit",
				style.Bold("[e]")+" edit",
				style.Bold("[r]")+" regenerate",
				style.Bold("[q]")+" quit",
			)
		} else {
			fmt.Fprintf(out, "%s  %s  %s\n",
				style.Bold("[Enter/e]")+" edit",
				style.Bold("[r]")+" regenerate",
				style.Bold("[q]")+" quit",
			)
		}

		input, err := reader.ReadString('\n')
		if err != nil {
			return "", "", fmt.Errorf("failed to read input: %w", err)
		}
		input = strings.TrimSpace(strings.ToLower(input))

		if valid {
			switch input {
			case "", "y", "yes":
				return "commit", message, nil
			case "e", "edit":
				edited, editErr := EditMessage(in, out, out, message)
				if editErr != nil {
					return "", "", editErr
				}
				return "edit", edited, nil
			case "r", "regenerate":
				return "regenerate", "", nil
			case "q", "quit":
				return "quit", "", nil
			default:
				fmt.Fprintln(out, "Invalid input. Please enter, e, r, or q.")
			}
			continue
		}

		switch input {
		case "", "e", "edit":
			edited, editErr := EditMessage(in, out, out, message)
			if editErr != nil {
				return "", "", editErr
			}
			return "edit", edited, nil
		case "r", "regenerate":
			return "regenerate", "", nil
		case "q", "quit":
			return "quit", "", nil
		default:
			fmt.Fprintln(out, "Invalid input. Please enter e, r, or q.")
		}
	}
}

// EditMessage opens the configured editor using the supplied terminal streams.
func EditMessage(in io.Reader, out, errOut io.Writer, original string) (string, error) {
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

	cmd := exec.Command("sh", "-c", editor+` "`+strings.ReplaceAll(tmpFile.Name(), `"`, `\"`)+`"`)
	cmd.Stdin = in
	cmd.Stdout = out
	cmd.Stderr = errOut

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

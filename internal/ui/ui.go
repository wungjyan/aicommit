package ui

import (
	"fmt"
	"io"
	"os"
	"time"

	"golang.org/x/term"
)

// UI writes human-facing diagnostics to one injected stream. Command data is
// written by the command layer to stdout and never passes through this type.
type UI struct {
	writer  io.Writer
	color   bool
	spinner bool
}

// New creates a diagnostic UI. Color and animated spinner output are enabled
// only when the target writer is a terminal.
func New(writer io.Writer) *UI {
	return newUI(writer, isTerminal)
}

func newUI(writer io.Writer, terminal func(io.Writer) bool) *UI {
	if writer == nil {
		writer = io.Discard
	}
	isTerminal := terminal(writer)
	return &UI{writer: writer, color: isTerminal, spinner: isTerminal}
}

func isTerminal(writer io.Writer) bool {
	file, ok := writer.(*os.File)
	return ok && term.IsTerminal(int(file.Fd()))
}

// DisableColor suppresses ANSI color and style sequences without changing the
// selected writer or spinner behavior.
func (u *UI) DisableColor() {
	u.color = false
}

// Success writes a successful status message to the diagnostic stream.
func (u *UI) Success(msg string) {
	fmt.Fprintf(u.writer, "%s%s %s%s\n", u.green(), "✔", msg, u.reset())
}

// Error writes an error status message to the diagnostic stream.
func (u *UI) Error(msg string) {
	fmt.Fprintf(u.writer, "%s%s %s%s\n", u.red(), "✘", msg, u.reset())
}

// Warn writes a warning status message to the diagnostic stream.
func (u *UI) Warn(msg string) {
	fmt.Fprintf(u.writer, "%s%s %s%s\n", u.yellow(), "⚠", msg, u.reset())
}

// Info writes an informational status message to the diagnostic stream.
func (u *UI) Info(msg string) {
	fmt.Fprintf(u.writer, "%s%s %s%s\n", u.cyan(), "ℹ", msg, u.reset())
}

// Bold returns text styled for terminal prompts.
func (u *UI) Bold(s string) string {
	return u.bold() + s + u.reset()
}

// Highlight returns highlighted text for terminal prompts.
func (u *UI) Highlight(s string) string {
	return u.cyan() + s + u.reset()
}

// Spinner runs fn while rendering progress on the diagnostic stream. Non-TTY
// destinations receive a stable one-line status rather than terminal control
// sequences, preserving useful logs for automation.
func (u *UI) Spinner(label string, fn func() error) error {
	if !u.spinner {
		fmt.Fprintf(u.writer, "%s... ", label)
		err := fn()
		if err != nil {
			fmt.Fprintln(u.writer, "failed")
		} else {
			fmt.Fprintln(u.writer, "done")
		}
		return err
	}

	frames := []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}
	done := make(chan error, 1)
	go func() { done <- fn() }()

	for i := 0; ; i++ {
		select {
		case err := <-done:
			fmt.Fprint(u.writer, "\r\033[K")
			if err != nil {
				fmt.Fprintf(u.writer, "%s%s %s%s\n", u.red(), "✘", label, u.reset())
			} else {
				fmt.Fprintf(u.writer, "%s%s %s%s\n", u.green(), "✔", label, u.reset())
			}
			return err
		default:
			fmt.Fprintf(u.writer, "\r%s%s%s %s", u.cyan(), frames[i%len(frames)], u.reset(), label)
			time.Sleep(80 * time.Millisecond)
		}
	}
}

func (u *UI) reset() string {
	return u.style("\033[0m")
}

func (u *UI) red() string {
	return u.style("\033[31m")
}

func (u *UI) green() string {
	return u.style("\033[32m")
}

func (u *UI) yellow() string {
	return u.style("\033[33m")
}

func (u *UI) cyan() string {
	return u.style("\033[36m")
}

func (u *UI) bold() string {
	return u.style("\033[1m")
}

func (u *UI) style(code string) string {
	if !u.color {
		return ""
	}
	return code
}

package ui

import (
	"fmt"
	"os"
	"time"
)

// ANSI color codes — disabled automatically when stdout is not a terminal.
var (
	colorReset  = "\033[0m"
	colorRed    = "\033[31m"
	colorGreen  = "\033[32m"
	colorYellow = "\033[33m"
	colorCyan   = "\033[36m"
	colorBold   = "\033[1m"
	colorDim    = "\033[2m"
)

func init() {
	// Disable colors when not writing to a real terminal (e.g. piped output).
	if !isTerminal(os.Stdout) {
		colorReset = ""
		colorRed = ""
		colorGreen = ""
		colorYellow = ""
		colorCyan = ""
		colorBold = ""
		colorDim = ""
	}
}

// DisableColor removes ANSI color and style sequences for the current process.
// Command-level --no-color calls this before any interactive UI is rendered.
func DisableColor() {
	colorReset = ""
	colorRed = ""
	colorGreen = ""
	colorYellow = ""
	colorCyan = ""
	colorBold = ""
	colorDim = ""
}

func isTerminal(f *os.File) bool {
	fi, err := f.Stat()
	if err != nil {
		return false
	}
	return (fi.Mode() & os.ModeCharDevice) != 0
}

// Success prints a green ✔ message.
func Success(msg string) {
	fmt.Printf("%s✔ %s%s\n", colorGreen, msg, colorReset)
}

// Error prints a red ✘ message.
func Error(msg string) {
	fmt.Printf("%s✘ %s%s\n", colorRed, msg, colorReset)
}

// Warn prints a yellow ⚠ message.
func Warn(msg string) {
	fmt.Printf("%s⚠ %s%s\n", colorYellow, msg, colorReset)
}

// Info prints a cyan ℹ message.
func Info(msg string) {
	fmt.Printf("%sℹ %s%s\n", colorCyan, msg, colorReset)
}

// Bold returns text wrapped in bold ANSI codes.
func Bold(s string) string {
	return colorBold + s + colorReset
}

// Dim returns text wrapped in dim ANSI codes.
func Dim(s string) string {
	return colorDim + s + colorReset
}

// Highlight returns text in cyan.
func Highlight(s string) string {
	return colorCyan + s + colorReset
}

// Spinner runs a terminal spinner in the background while fn executes.
// It prints a final success or error line depending on the returned error.
func Spinner(label string, fn func() error) error {
	if !isTerminal(os.Stdout) {
		// No spinner in non-interactive mode — just run the function.
		fmt.Printf("%s... ", label)
		err := fn()
		if err != nil {
			fmt.Println("failed")
		} else {
			fmt.Println("done")
		}
		return err
	}

	frames := []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}
	done := make(chan error, 1)

	go func() {
		done <- fn()
	}()

	i := 0
	for {
		select {
		case err := <-done:
			// Clear the spinner line.
			fmt.Printf("\r\033[K")
			if err != nil {
				fmt.Printf("%s✘ %s%s\n", colorRed, label, colorReset)
			} else {
				fmt.Printf("%s✔ %s%s\n", colorGreen, label, colorReset)
			}
			return err
		default:
			fmt.Printf("\r%s%s%s %s", colorCyan, frames[i%len(frames)], colorReset, label)
			i++
			time.Sleep(80 * time.Millisecond)
		}
	}
}

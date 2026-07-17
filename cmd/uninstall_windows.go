//go:build windows

package cmd

import (
	"fmt"
	"os/exec"
	"strings"
)

// Windows cannot remove a running executable. Start a short-lived child that
// waits for this process to exit, then removes the file.
func removeCurrentExecutable(path string) error {
	quotedPath := strings.ReplaceAll(path, `"`, `""`)
	command := `ping 127.0.0.1 -n 2 > NUL & del /f /q "` + quotedPath + `"`
	if err := exec.Command("cmd", "/C", command).Start(); err != nil {
		return fmt.Errorf("schedule binary removal: %w", err)
	}
	return nil
}

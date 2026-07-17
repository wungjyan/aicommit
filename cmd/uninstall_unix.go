//go:build !windows

package cmd

import "os"

func removeCurrentExecutable(path string) error {
	return os.Remove(path)
}

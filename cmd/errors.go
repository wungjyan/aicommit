package cmd

import (
	"errors"
	"fmt"
)

// ErrUsage marks a command-usage error (bad flag combination, missing required
// field, invalid value). The exit-code mapping in a later commit maps it to the
// usage exit code; for now it is a normal returned error so commands fail
// rather than silently succeed.
var ErrUsage = errors.New("usage error")

// usageErrorf builds a usage error wrapping ErrUsage.
func usageErrorf(format string, args ...any) error {
	return fmt.Errorf("%s: %w", fmt.Sprintf(format, args...), ErrUsage)
}

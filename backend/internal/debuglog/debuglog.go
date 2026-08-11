package debuglog

import (
	"log"
	"os"
	"strings"
	"sync/atomic"
)

var enabled atomic.Bool

// ConfigureFromEnv enables debug logging when C7D5A6L_DEBUG is 1/true/yes/on.
func ConfigureFromEnv() {
	SetEnabled(envTruthy(os.Getenv("C7D5A6L_DEBUG")))
}

// SetEnabled turns debug logging on or off.
func SetEnabled(on bool) {
	enabled.Store(on)
}

// Enabled reports whether debug logging is on.
func Enabled() bool {
	return enabled.Load()
}

// Printf logs with a [debug] prefix when enabled.
func Printf(format string, args ...any) {
	if !enabled.Load() {
		return
	}
	log.Printf("[debug] "+format, args...)
}

// Str formats an optional string pointer for logs ("—" when nil).
func Str(p *string) string {
	if p == nil {
		return "—"
	}
	return *p
}

// Bool formats an optional bool pointer for logs ("—" when nil).
func Bool(p *bool) string {
	if p == nil {
		return "—"
	}
	if *p {
		return "true"
	}
	return "false"
}

func envTruthy(v string) bool {
	switch strings.ToLower(strings.TrimSpace(v)) {
	case "1", "true", "yes", "on":
		return true
	default:
		return false
	}
}

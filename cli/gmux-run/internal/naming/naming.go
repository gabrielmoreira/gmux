package naming

import (
	"crypto/rand"
	"fmt"
	"path/filepath"
	"strings"
)

// AbducoName generates a deterministic-ish abduco session name.
// Format: <kind>:<project>:<random-suffix>
func AbducoName(kind, cwd string) string {
	project := filepath.Base(cwd)
	// sanitize for abduco (no slashes, colons in project)
	project = strings.ReplaceAll(project, ":", "-")
	project = strings.ReplaceAll(project, "/", "-")
	suffix := shortID()
	return fmt.Sprintf("%s:%s:%s", kind, project, suffix)
}

// SessionID generates a unique session identifier.
func SessionID() string {
	return "sess-" + shortID()
}

func shortID() string {
	b := make([]byte, 4)
	_, _ = rand.Read(b)
	return fmt.Sprintf("%x", b)
}

package adapters

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// PiSessionDir computes pi's session directory for a given cwd.
// Pi encodes: strip leading /, replace remaining / with -, wrap in --.
// /home/mg/dev/gmux → --home-mg-dev-gmux--
func PiSessionDir(cwd string) string {
	home, _ := os.UserHomeDir()
	path := strings.TrimPrefix(cwd, "/")
	encoded := "--" + strings.ReplaceAll(path, "/", "-") + "--"
	return filepath.Join(home, ".pi", "agent", "sessions", encoded)
}

// PiSessionHeader is the first line of a pi JSONL session file.
type PiSessionHeader struct {
	Type    string `json:"type"`
	Version int    `json:"version"`
	ID      string `json:"id"`
	Cwd     string `json:"cwd"`
}

// ReadPiSessionHeader reads and parses the first line of a session file.
func ReadPiSessionHeader(path string) (*PiSessionHeader, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	buf := make([]byte, 1024)
	n, err := f.Read(buf)
	if err != nil || n == 0 {
		return nil, fmt.Errorf("empty file")
	}

	line := string(buf[:n])
	if idx := strings.IndexByte(line, '\n'); idx >= 0 {
		line = line[:idx]
	}

	var h PiSessionHeader
	if err := json.Unmarshal([]byte(line), &h); err != nil {
		return nil, err
	}
	if h.Type != "session" {
		return nil, fmt.Errorf("not a session header: type=%s", h.Type)
	}
	return &h, nil
}

// ExtractPiText reads a pi JSONL session file and extracts conversation
// text suitable for similarity matching. Returns concatenated text content
// from message entries (user, assistant, tool results).
//
// This is the pi-specific text extractor for ADR-0009 content similarity
// matching. It extracts "text" values from content arrays in message
// entries, ignoring JSON structure, keys, and non-text content types.
func ExtractPiText(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}

	var out strings.Builder
	for _, line := range strings.Split(strings.TrimSpace(string(data)), "\n") {
		if line == "" {
			continue
		}
		// Quick check: does this line contain text content?
		if !strings.Contains(line, `"text"`) {
			continue
		}
		// Parse and extract text values from content arrays
		var entry struct {
			Type    string `json:"type"`
			Message *struct {
				Content json.RawMessage `json:"content"`
			} `json:"message"`
		}
		if err := json.Unmarshal([]byte(line), &entry); err != nil {
			continue
		}
		if entry.Message == nil {
			continue
		}
		// Content can be a string or array of content blocks
		var blocks []struct {
			Type string `json:"type"`
			Text string `json:"text"`
		}
		if err := json.Unmarshal(entry.Message.Content, &blocks); err != nil {
			// Try as plain string
			var s string
			if err := json.Unmarshal(entry.Message.Content, &s); err == nil {
				out.WriteString(s)
				out.WriteByte(' ')
			}
			continue
		}
		for _, b := range blocks {
			if b.Type == "text" && b.Text != "" {
				out.WriteString(b.Text)
				out.WriteByte(' ')
			}
		}
	}
	return out.String(), nil
}

// ListSessionFiles returns all .jsonl files in a directory.
func ListSessionFiles(dir string) []string {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil
	}
	var files []string
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".jsonl") {
			files = append(files, filepath.Join(dir, e.Name()))
		}
	}
	return files
}

package paths

import (
	"os"
	"testing"
)

func TestNormalizePath(t *testing.T) {
	home, err := os.UserHomeDir()
	if err != nil {
		t.Skip("no home dir")
	}

	tests := []struct {
		input string
		want  string
	}{
		{"~", home},
		{"~/dev/gmux", home + "/dev/gmux"},
		{"/opt/data", "/opt/data"},
		{"", ""},
		// Already absolute: unchanged.
		{home + "/dev/gmux", home + "/dev/gmux"},
	}
	for _, tt := range tests {
		got := NormalizePath(tt.input)
		if got != tt.want {
			t.Errorf("NormalizePath(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestCanonicalizePath(t *testing.T) {
	home, err := os.UserHomeDir()
	if err != nil {
		t.Skip("no home dir")
	}

	tests := []struct {
		input string
		want  string
	}{
		{home, "~"},
		{home + "/dev/gmux", "~/dev/gmux"},
		{home + "/", "~"},
		{"/opt/data", "/opt/data"},
		{"/tmp/../tmp", "/tmp"},
		{"", ""},
		// Already canonical: passes through unchanged.
		{"~/dev/gmux", "~/dev/gmux"},
		{"~", "~"},
	}
	for _, tt := range tests {
		got := CanonicalizePath(tt.input)
		if got != tt.want {
			t.Errorf("CanonicalizePath(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestSessionSocketDirUsesOverride(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("GMUX_SOCKET_DIR", dir)

	if got := SessionSocketDir(); got != dir {
		t.Fatalf("SessionSocketDir() = %q, want %q", got, dir)
	}
	if got := SessionSocketPath("sess-123"); got != filepath.Join(dir, "sess-123.sock") {
		t.Fatalf("SessionSocketPath() = %q, want %q", got, filepath.Join(dir, "sess-123.sock"))
	}
}

func TestSessionSocketDirUsesStableUnixDefault(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("windows uses the temp directory for session sockets")
	}

	t.Setenv("GMUX_SOCKET_DIR", "")
	t.Setenv("TMPDIR", filepath.Join(t.TempDir(), "custom-tmp"))

	want := filepath.Join("/tmp", "gmux-sessions")
	if got := SessionSocketDir(); got != want {
		t.Fatalf("SessionSocketDir() = %q, want %q", got, want)
	}
	if got := SessionSocketPath("sess-123"); got != filepath.Join(want, "sess-123.sock") {
		t.Fatalf("SessionSocketPath() = %q, want %q", got, filepath.Join(want, "sess-123.sock"))
	}
}

func TestSessionSocketDirUsesTempDirOnWindows(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("windows-only path assertion")
	}

	t.Setenv("GMUX_SOCKET_DIR", "")

	want := filepath.Join(os.TempDir(), "gmux-sessions")
	if got := SessionSocketDir(); got != want {
		t.Fatalf("SessionSocketDir() = %q, want %q", got, want)
	}
}

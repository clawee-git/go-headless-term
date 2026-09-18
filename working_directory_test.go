package headlessterm

import (
	"testing"
)

func TestWorkingDirectory(t *testing.T) {
	cases := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "basic BEL terminator",
			input:    "\x1b]7;file://localhost/home/user\x07",
			expected: "file://localhost/home/user",
		},
		{
			name:     "ST terminator",
			input:    "\x1b]7;file://myhost/var/log\x1b\\",
			expected: "file://myhost/var/log",
		},
		{
			name:     "not set",
			input:    "",
			expected: "",
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			term := New(WithSize(24, 80))
			if c.input != "" {
				term.WriteString(c.input)
			}
			got := term.WorkingDirectory()
			if got != c.expected {
				t.Errorf("expected %q, got %q", c.expected, got)
			}
		})
	}

	t.Run("multiple updates", func(t *testing.T) {
		term := New(WithSize(24, 80))
		term.WriteString("\x1b]7;file://localhost/home/user\x07")
		if got := term.WorkingDirectory(); got != "file://localhost/home/user" {
			t.Errorf("expected file://localhost/home/user, got %q", got)
		}
		term.WriteString("\x1b]7;file://localhost/tmp\x07")
		if got := term.WorkingDirectory(); got != "file://localhost/tmp" {
			t.Errorf("expected file://localhost/tmp, got %q", got)
		}
	})
}

func TestWorkingDirectoryPath(t *testing.T) {
	cases := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "basic localhost",
			input:    "\x1b]7;file://localhost/home/user\x07",
			expected: "/home/user",
		},
		{
			name:     "with hostname",
			input:    "\x1b]7;file://mycomputer.local/var/log/system\x07",
			expected: "/var/log/system",
		},
		{
			name:     "empty hostname",
			input:    "\x1b]7;file:///home/user\x07",
			expected: "/home/user",
		},
		{
			name:     "not set",
			input:    "",
			expected: "",
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			term := New(WithSize(24, 80))
			if c.input != "" {
				term.WriteString(c.input)
			}
			got := term.WorkingDirectoryPath()
			if got != c.expected {
				t.Errorf("expected %q, got %q", c.expected, got)
			}
		})
	}
}

func TestWorkingDirectory_Middleware(t *testing.T) {
	var middlewareCalled bool
	var receivedURI string

	mw := &Middleware{
		SetWorkingDirectory: func(uri string, next func(string)) {
			middlewareCalled = true
			receivedURI = uri
			next(uri)
		},
	}

	term := New(WithSize(24, 80), WithMiddleware(mw))

	term.WriteString("\x1b]7;file://localhost/test\x07")

	if !middlewareCalled {
		t.Error("expected middleware to be called")
	}
	if receivedURI != "file://localhost/test" {
		t.Errorf("expected file://localhost/test, got %q", receivedURI)
	}
	if term.WorkingDirectory() != "file://localhost/test" {
		t.Errorf("expected working directory to be set")
	}
}

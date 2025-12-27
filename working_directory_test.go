package headlessterm

import (
	"testing"
)

func TestWorkingDirectory_Basic(t *testing.T) {
	term := New(WithSize(24, 80))

	// OSC 7 ; file://hostname/path BEL
	term.WriteString("\x1b]7;file://localhost/home/user\x07")

	uri := term.WorkingDirectory()
	expected := "file://localhost/home/user"
	if uri != expected {
		t.Errorf("expected %q, got %q", expected, uri)
	}
}

func TestWorkingDirectory_STTerminator(t *testing.T) {
	term := New(WithSize(24, 80))

	// OSC 7 ; file://hostname/path ST (ESC \)
	term.WriteString("\x1b]7;file://myhost/var/log\x1b\\")

	uri := term.WorkingDirectory()
	expected := "file://myhost/var/log"
	if uri != expected {
		t.Errorf("expected %q, got %q", expected, uri)
	}
}

func TestWorkingDirectory_Multiple(t *testing.T) {
	term := New(WithSize(24, 80))

	// Set first directory
	term.WriteString("\x1b]7;file://localhost/home/user\x07")
	uri := term.WorkingDirectory()
	if uri != "file://localhost/home/user" {
		t.Errorf("expected file://localhost/home/user, got %q", uri)
	}

	// Change directory
	term.WriteString("\x1b]7;file://localhost/tmp\x07")
	uri = term.WorkingDirectory()
	if uri != "file://localhost/tmp" {
		t.Errorf("expected file://localhost/tmp, got %q", uri)
	}
}

func TestWorkingDirectory_NotSet(t *testing.T) {
	term := New(WithSize(24, 80))

	uri := term.WorkingDirectory()
	if uri != "" {
		t.Errorf("expected empty string, got %q", uri)
	}
}

func TestWorkingDirectoryPath_Basic(t *testing.T) {
	term := New(WithSize(24, 80))

	term.WriteString("\x1b]7;file://localhost/home/user\x07")

	path := term.WorkingDirectoryPath()
	expected := "/home/user"
	if path != expected {
		t.Errorf("expected %q, got %q", expected, path)
	}
}

func TestWorkingDirectoryPath_WithHostname(t *testing.T) {
	term := New(WithSize(24, 80))

	term.WriteString("\x1b]7;file://mycomputer.local/var/log/system\x07")

	path := term.WorkingDirectoryPath()
	expected := "/var/log/system"
	if path != expected {
		t.Errorf("expected %q, got %q", expected, path)
	}
}

func TestWorkingDirectoryPath_EmptyHostname(t *testing.T) {
	term := New(WithSize(24, 80))

	// Some systems emit file:///path (empty hostname)
	term.WriteString("\x1b]7;file:///home/user\x07")

	path := term.WorkingDirectoryPath()
	expected := "/home/user"
	if path != expected {
		t.Errorf("expected %q, got %q", expected, path)
	}
}

func TestWorkingDirectoryPath_NotSet(t *testing.T) {
	term := New(WithSize(24, 80))

	path := term.WorkingDirectoryPath()
	if path != "" {
		t.Errorf("expected empty string, got %q", path)
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

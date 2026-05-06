package cmd

import (
	"bytes"
	"encoding/json"
	"errors"
	"strings"
	"testing"
)

func TestDecodeScreenshotDataURL(t *testing.T) {
	mimeType, data, err := decodeScreenshotDataURL("data:image/png;base64,aGVsbG8=")
	if err != nil {
		t.Fatalf("decodeScreenshotDataURL returned error: %v", err)
	}
	if mimeType != "image/png" {
		t.Fatalf("expected mime type image/png, got %q", mimeType)
	}
	if string(data) != "hello" {
		t.Fatalf("expected decoded payload hello, got %q", string(data))
	}
}

func TestResolvePathFromInitialCwd(t *testing.T) {
	oldInitialCwd := InitialCwd
	InitialCwd = `C:\tmp\wave-root`
	defer func() {
		InitialCwd = oldInitialCwd
	}()

	resolved, err := resolvePathFromInitialCwd(`artifacts\shot.png`)
	if err != nil {
		t.Fatalf("resolvePathFromInitialCwd returned error: %v", err)
	}
	expected := `C:\tmp\wave-root\artifacts\shot.png`
	if resolved != expected {
		t.Fatalf("expected %q, got %q", expected, resolved)
	}
}

func TestEmitAutomationActionLifecycleSuccess(t *testing.T) {
	var buf bytes.Buffer
	oldStdout := WrappedStdout
	WrappedStdout = &buf
	defer func() {
		WrappedStdout = oldStdout
	}()

	err := emitAutomationActionLifecycle("type", "block-1", map[string]any{"bytes": 4}, func() error {
		return nil
	})
	if err != nil {
		t.Fatalf("emitAutomationActionLifecycle returned error: %v", err)
	}

	lines := strings.Split(strings.TrimSpace(buf.String()), "\n")
	if len(lines) != 2 {
		t.Fatalf("expected 2 json lines, got %d: %q", len(lines), buf.String())
	}

	var started automationEnvelope
	if err := json.Unmarshal([]byte(lines[0]), &started); err != nil {
		t.Fatalf("unmarshal started envelope: %v", err)
	}
	if started.Type != "action_started" {
		t.Fatalf("expected first event action_started, got %q", started.Type)
	}

	var completed automationEnvelope
	if err := json.Unmarshal([]byte(lines[1]), &completed); err != nil {
		t.Fatalf("unmarshal completed envelope: %v", err)
	}
	if completed.Type != "action_completed" {
		t.Fatalf("expected second event action_completed, got %q", completed.Type)
	}
}

func TestEmitAutomationActionLifecycleFailure(t *testing.T) {
	var buf bytes.Buffer
	oldStdout := WrappedStdout
	WrappedStdout = &buf
	defer func() {
		WrappedStdout = oldStdout
	}()

	runErr := errors.New("no controller found")
	err := emitAutomationActionLifecycle("type", "block-1", nil, func() error {
		return runErr
	})
	if !errors.Is(err, runErr) {
		t.Fatalf("expected error %v, got %v", runErr, err)
	}

	lines := strings.Split(strings.TrimSpace(buf.String()), "\n")
	if len(lines) != 2 {
		t.Fatalf("expected 2 json lines, got %d: %q", len(lines), buf.String())
	}

	var failed automationEnvelope
	if err := json.Unmarshal([]byte(lines[1]), &failed); err != nil {
		t.Fatalf("unmarshal failed envelope: %v", err)
	}
	if failed.Type != "action_failed" {
		t.Fatalf("expected second event action_failed, got %q", failed.Type)
	}
}

func TestFirstAutomationValue(t *testing.T) {
	value := firstAutomationValue("", "  ", "window-1", "window-2")
	if value != "window-1" {
		t.Fatalf("expected first non-empty value, got %q", value)
	}
}

func TestFirstArg(t *testing.T) {
	if firstArg(nil) != "" {
		t.Fatalf("expected empty string for nil args")
	}
	if firstArg([]string{"abc", "def"}) != "abc" {
		t.Fatalf("expected first arg abc")
	}
}

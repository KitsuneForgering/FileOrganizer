package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/KitsuneForgering/FileOrganizer/internal/organizer"
)

func TestJSONProtocolPreviewAndApply(t *testing.T) {
	base := t.TempDir()
	source, dest := filepath.Join(base, "source"), filepath.Join(base, "dest")
	if err := os.Mkdir(source, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(source, "note.txt"), []byte("hello"), 0644); err != nil {
		t.Fatal(err)
	}
	request, err := json.Marshal(organizer.Request{Source: source, Destination: dest, Rules: []organizer.Rule{{Extensions: []string{"txt"}, Folder: "Notes"}}})
	if err != nil {
		t.Fatal(err)
	}
	for _, action := range []string{"preview", "apply"} {
		var out, stderr bytes.Buffer
		if code := run([]string{action}, bytes.NewReader(request), &out, &stderr); code != 0 {
			t.Fatalf("%s exited %d: %s", action, code, stderr.String())
		}
		var result organizer.Result
		if err := json.Unmarshal(out.Bytes(), &result); err != nil {
			t.Fatalf("%s returned invalid JSON: %v", action, err)
		}
		want := "ready"
		if action == "apply" {
			want = "moved"
		}
		if len(result.Moves) != 1 || result.Moves[0].Status != want {
			t.Fatalf("%s: %+v", action, result)
		}
	}
	if data, err := os.ReadFile(filepath.Join(dest, "Notes", "note.txt")); err != nil || string(data) != "hello" {
		t.Fatalf("file not moved through JSON protocol: %q, %v", data, err)
	}
}

func TestJSONProtocolRejectsBadInput(t *testing.T) {
	for _, tc := range []struct {
		name      string
		args      []string
		input     string
		code      int
		errorText string
	}{
		{"action", []string{"delete"}, `{}`, 2, "usage"},
		{"malformed", []string{"preview"}, `{`, 2, "EOF"},
		{"unknown field", []string{"preview"}, `{"surprise":true}`, 2, "unknown field"},
		{"second request", []string{"preview"}, `{} {}`, 2, "exactly one"},
		{"invalid request", []string{"apply"}, `{}`, 1, "source and destination"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var out, stderr bytes.Buffer
			if code := run(tc.args, strings.NewReader(tc.input), &out, &stderr); code != tc.code || !strings.Contains(stderr.String(), tc.errorText) {
				t.Fatalf("code=%d stderr=%q", code, stderr.String())
			}
			if out.Len() != 0 {
				t.Fatalf("failed command wrote stdout: %q", out.String())
			}
		})
	}
}

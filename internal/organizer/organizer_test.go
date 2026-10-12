package organizer

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestPreviewAndApply(t *testing.T) {
	base := t.TempDir()
	source := filepath.Join(base, "source")
	dest := filepath.Join(base, "dest")
	if err := os.Mkdir(source, 0755); err != nil {
		t.Fatal(err)
	}
	for name, content := range map[string]string{
		"report.PDF": "new", "photo.jpg": "image", ".hidden.pdf": "private", "other.bin": "other",
	} {
		if err := os.WriteFile(filepath.Join(source, name), []byte(content), 0644); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.MkdirAll(filepath.Join(dest, "Documents"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dest, "Documents", "report.PDF"), []byte("old"), 0644); err != nil {
		t.Fatal(err)
	}
	req := Request{Source: source, Destination: dest, Rules: []Rule{
		{Extensions: []string{"pdf"}, Folder: "Documents"},
		{Extensions: []string{"jpg"}, Folder: "Images"},
	}}
	preview, err := Preview(req)
	if err != nil {
		t.Fatal(err)
	}
	if len(preview.Moves) != 2 || preview.Skipped != 2 {
		t.Fatalf("unexpected preview: %+v", preview)
	}
	if preview.Moves[1].Status != "conflict" {
		t.Fatalf("expected conflict: %+v", preview.Moves)
	}
	result, err := Apply(req)
	if err != nil {
		t.Fatal(err)
	}
	if result.Moved != 1 || result.Skipped != 3 {
		t.Fatalf("unexpected result: %+v", result)
	}
	if got, err := os.ReadFile(filepath.Join(dest, "Documents", "report.PDF")); err != nil || string(got) != "old" {
		t.Fatalf("existing file changed: %q, %v", got, err)
	}
	if got, err := os.ReadFile(filepath.Join(dest, "Images", "photo.jpg")); err != nil || string(got) != "image" {
		t.Fatalf("moved file: %q, %v", got, err)
	}
	if _, err := os.Stat(filepath.Join(source, "photo.jpg")); !os.IsNotExist(err) {
		t.Fatalf("source remains: %v", err)
	}
}

func TestRejectTraversal(t *testing.T) {
	base := t.TempDir()
	req := Request{Source: base, Destination: filepath.Join(base, "dest"), Rules: []Rule{
		{Extensions: []string{"txt"}, Folder: "../outside"},
	}}
	if _, err := Preview(req); err == nil {
		t.Fatal("accepted folder traversal")
	}
}

func TestRejectSymlinkedDestinationFolder(t *testing.T) {
	base := t.TempDir()
	source := filepath.Join(base, "source")
	dest := filepath.Join(base, "dest")
	outside := filepath.Join(base, "outside")
	for _, dir := range []string{source, dest, outside} {
		if err := os.Mkdir(dir, 0755); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(filepath.Join(source, "note.txt"), []byte("data"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, filepath.Join(dest, "Documents")); err != nil {
		t.Fatal(err)
	}
	req := Request{Source: source, Destination: dest, Rules: []Rule{{Extensions: []string{"txt"}, Folder: "Documents"}}}
	result, err := Apply(req)
	if err != nil {
		t.Fatal(err)
	}
	if result.Moved != 0 || len(result.Errors) != 1 {
		t.Fatalf("unexpected result: %+v", result)
	}
	if _, err := os.Stat(filepath.Join(source, "note.txt")); err != nil {
		t.Fatalf("source lost: %v", err)
	}
	if _, err := os.Stat(filepath.Join(outside, "note.txt")); !os.IsNotExist(err) {
		t.Fatalf("file escaped: %v", err)
	}
}

func TestRuleOrderAndPreviewDoesNotMove(t *testing.T) {
	base := t.TempDir()
	source, dest := filepath.Join(base, "source"), filepath.Join(base, "dest")
	if err := os.Mkdir(source, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(source, "Report.PDF"), []byte("report"), 0644); err != nil {
		t.Fatal(err)
	}
	req := Request{Source: source, Destination: dest, Rules: []Rule{
		{Extensions: []string{" .PDF "}, Folder: "First"},
		{Extensions: []string{"pdf"}, Folder: "Second"},
	}}
	result, err := Preview(req)
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Moves) != 1 || result.Moves[0].Folder != "First" || result.Moves[0].Status != "ready" {
		t.Fatalf("wrong rule selected: %+v", result)
	}
	if _, err := os.Stat(filepath.Join(source, "Report.PDF")); err != nil {
		t.Fatalf("preview touched source: %v", err)
	}
	if _, err := os.Stat(dest); !os.IsNotExist(err) {
		t.Fatalf("preview created destination: %v", err)
	}
}

func TestApplyRescansAndPreservesMetadata(t *testing.T) {
	base := t.TempDir()
	source, dest := filepath.Join(base, "source"), filepath.Join(base, "dest")
	if err := os.Mkdir(source, 0755); err != nil {
		t.Fatal(err)
	}
	req := Request{Source: source, Destination: dest, Rules: []Rule{{Extensions: []string{"txt"}, Folder: "Notes/2026"}}}
	if _, err := Preview(req); err != nil {
		t.Fatal(err)
	}
	from := filepath.Join(source, "new.txt")
	if err := os.WriteFile(from, []byte("after preview"), 0600); err != nil {
		t.Fatal(err)
	}
	modified := time.Date(2024, time.January, 2, 3, 4, 5, 0, time.UTC)
	if err := os.Chtimes(from, modified, modified); err != nil {
		t.Fatal(err)
	}
	result, err := Apply(req)
	if err != nil {
		t.Fatal(err)
	}
	if result.Moved != 1 || result.Moves[0].Status != "moved" {
		t.Fatalf("file added after preview was not moved: %+v", result)
	}
	to := filepath.Join(dest, "Notes", "2026", "new.txt")
	data, err := os.ReadFile(to)
	if err != nil || string(data) != "after preview" {
		t.Fatalf("moved content: %q, %v", data, err)
	}
	info, err := os.Stat(to)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0600 || !info.ModTime().Equal(modified) {
		t.Fatalf("metadata changed: mode=%#o time=%s", info.Mode().Perm(), info.ModTime())
	}
}

func TestIgnoreSymlinksAndSubdirectories(t *testing.T) {
	base := t.TempDir()
	source, dest := filepath.Join(base, "source"), filepath.Join(base, "dest")
	if err := os.MkdirAll(filepath.Join(source, "subdir"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(source, "subdir", "nested.txt"), []byte("nested"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(filepath.Join(source, "subdir", "nested.txt"), filepath.Join(source, "alias.txt")); err != nil {
		t.Fatal(err)
	}
	req := Request{Source: source, Destination: dest, Rules: []Rule{{Extensions: []string{"txt"}, Folder: "Notes"}}}
	result, err := Apply(req)
	if err != nil {
		t.Fatal(err)
	}
	if result.Moved != 0 || result.Skipped != 2 {
		t.Fatalf("nonregular entries moved: %+v", result)
	}
	if _, err := os.Lstat(filepath.Join(source, "alias.txt")); err != nil {
		t.Fatal(err)
	}
}

func TestInvalidRequestsDoNotCreateDestination(t *testing.T) {
	base := t.TempDir()
	source := filepath.Join(base, "source")
	if err := os.Mkdir(source, 0755); err != nil {
		t.Fatal(err)
	}
	dest := filepath.Join(base, "dest")
	tests := []struct {
		name string
		req  Request
		want string
	}{
		{"same directory", Request{Source: source, Destination: source, Rules: []Rule{{Extensions: []string{"txt"}, Folder: "Notes"}}}, "must differ"},
		{"no rules", Request{Source: source, Destination: dest}, "at least one"},
		{"absolute folder", Request{Source: source, Destination: dest, Rules: []Rule{{Extensions: []string{"txt"}, Folder: "/tmp"}}}, "relative path"},
		{"empty extension", Request{Source: source, Destination: dest, Rules: []Rule{{Extensions: []string{""}, Folder: "Notes"}}}, "invalid extension"},
		{"wildcard extension", Request{Source: source, Destination: dest, Rules: []Rule{{Extensions: []string{"*"}, Folder: "Notes"}}}, "invalid extension"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := Apply(tc.req); err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("got %v, want %q", err, tc.want)
			}
			if _, err := os.Stat(dest); !os.IsNotExist(err) {
				t.Fatalf("invalid request created destination: %v", err)
			}
		})
	}
}

func TestAdditionalRuleConditions(t *testing.T) {
	base := t.TempDir()
	source, dest := filepath.Join(base, "source"), filepath.Join(base, "dest")
	if err := os.Mkdir(source, 0755); err != nil {
		t.Fatal(err)
	}
	files := map[string]string{
		"Invoice-2026.PDF": "%PDF-1.4\ninvoice",
		"photo-one.jpg":    "plain text with wrong extension",
		"IMG_42.png":       "\x89PNG\r\n\x1a\nimage content",
		"note.txt":         "hello",
	}
	for name, content := range files {
		if err := os.WriteFile(filepath.Join(source, name), []byte(content), 0644); err != nil {
			t.Fatal(err)
		}
	}
	req := Request{Source: source, Destination: dest, Rules: []Rule{
		{Extensions: []string{"pdf"}, Name: "Invoice-2026.PDF", MIMETypes: []string{"application/pdf"}, MinBytes: int64ptr(10), Folder: "Exact"},
		{Extensions: []string{"jpg"}, NameGlob: "photo-*.jpg", MIMETypes: []string{"image/*"}, Folder: "ShouldNotMatch"},
		{NameRegex: `^IMG_[0-9]+\.png$`, MIMETypes: []string{"image/*", "application/pdf"}, MaxBytes: int64ptr(100), Folder: "Images"},
		{NameGlob: "note.*", MinBytes: int64ptr(1), MaxBytes: int64ptr(5), Folder: "Notes"},
	}}
	result, err := Preview(req)
	if err != nil {
		t.Fatal(err)
	}
	got := make(map[string]string)
	for _, move := range result.Moves {
		got[move.Name] = move.Folder
	}
	for name, folder := range map[string]string{"Invoice-2026.PDF": "Exact", "IMG_42.png": "Images", "note.txt": "Notes"} {
		if got[name] != folder {
			t.Errorf("%s went to %q, want %q", name, got[name], folder)
		}
	}
	if _, matched := got["photo-one.jpg"]; matched {
		t.Error("MIME filter trusted extension instead of file content")
	}
	if result.Skipped != 1 {
		t.Fatalf("unexpected skip count: %+v", result)
	}
}

func TestRegexOnlyAndGlobOnlyRules(t *testing.T) {
	base := t.TempDir()
	source := filepath.Join(base, "source")
	if err := os.Mkdir(source, 0755); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"receipt-42.dat", "archive-2026.zip", "other.dat"} {
		if err := os.WriteFile(filepath.Join(source, name), []byte("x"), 0644); err != nil {
			t.Fatal(err)
		}
	}
	req := Request{Source: source, Destination: filepath.Join(base, "dest"), Rules: []Rule{
		{NameRegex: `^receipt-[0-9]+\.dat$`, Folder: "Receipts"},
		{NameGlob: "archive-*.zip", Folder: "Archives"},
	}}
	result, err := Preview(req)
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Moves) != 2 || result.Skipped != 1 {
		t.Fatalf("regex/glob-only rules did not match: %+v", result)
	}
}

func TestRejectInvalidAdditionalConditions(t *testing.T) {
	base := t.TempDir()
	min, max, negative := int64(20), int64(10), int64(-1)
	cases := []struct {
		name string
		rule Rule
		want string
	}{
		{"no condition", Rule{Folder: "Other"}, "matching condition"},
		{"invalid regex", Rule{NameRegex: "[", Folder: "Other"}, "name regex"},
		{"invalid glob", Rule{NameGlob: "[", Folder: "Other"}, "name glob"},
		{"invalid MIME", Rule{MIMETypes: []string{"image"}, Folder: "Other"}, "MIME type"},
		{"inverted size", Rule{MinBytes: &min, MaxBytes: &max, Folder: "Other"}, "minimum size"},
		{"negative size", Rule{MinBytes: &negative, Folder: "Other"}, "negative"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := Preview(Request{Source: base, Destination: filepath.Join(base, "dest"), Rules: []Rule{tc.rule}})
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("got %v, want %q", err, tc.want)
			}
		})
	}
}

func TestPartialAndApproximateNames(t *testing.T) {
	base := t.TempDir()
	source := filepath.Join(base, "source")
	if err := os.Mkdir(source, 0755); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"Fatura-março.pdf", "relatrio.pdf", "outro.pdf"} {
		if err := os.WriteFile(filepath.Join(source, name), []byte("data"), 0644); err != nil {
			t.Fatal(err)
		}
	}
	req := Request{Source: source, Destination: filepath.Join(base, "dest"), Rules: []Rule{
		{NameContains: "FATURA", Folder: "Invoices"},
		{NameFuzzy: "relatorio.pdf", Folder: "Reports"},
	}}
	result, err := Preview(req)
	if err != nil {
		t.Fatal(err)
	}
	got := make(map[string]string)
	for _, move := range result.Moves {
		got[move.Name] = move.Folder
	}
	if got["Fatura-março.pdf"] != "Invoices" || got["relatrio.pdf"] != "Reports" || result.Skipped != 1 {
		t.Fatalf("unexpected inexact matches: %+v", result)
	}
}

func TestEditDistanceBoundaries(t *testing.T) {
	for _, tc := range []struct {
		a, b  string
		limit int
		want  bool
	}{
		{"report.pdf", "report.pdf", 1, true},
		{"report.pdf", "report.pfd", 1, false},
		{"report.pdf", "report.pfd", 2, true},
		{"relatório.pdf", "relatorio.pdf", 1, true},
		{"unrelated.txt", "report.pdf", 3, false},
	} {
		if got := withinEditDistance(tc.a, tc.b, tc.limit); got != tc.want {
			t.Errorf("distance(%q, %q, %d) = %v, want %v", tc.a, tc.b, tc.limit, got, tc.want)
		}
	}
}

func TestRejectInvalidFuzzyThreshold(t *testing.T) {
	base := t.TempDir()
	for _, rule := range []Rule{
		{NameFuzzy: "file.pdf", MaxEdits: intptr(0), Folder: "Other"},
		{NameFuzzy: "file.pdf", MaxEdits: intptr(4), Folder: "Other"},
		{NameContains: "file", MaxEdits: intptr(1), Folder: "Other"},
	} {
		if _, err := Preview(Request{Source: base, Destination: filepath.Join(base, "dest"), Rules: []Rule{rule}}); err == nil {
			t.Fatalf("accepted invalid fuzzy threshold: %+v", rule)
		}
	}
}

func int64ptr(value int64) *int64 { return &value }
func intptr(value int) *int       { return &value }

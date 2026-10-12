package organizer

import (
	"errors"
	"fmt"
	"io"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// Rule order matters: the first matching rule wins.
type Rule struct {
	Extensions   []string `json:"extensions,omitempty"`
	Name         string   `json:"name,omitempty"`
	NameContains string   `json:"name_contains,omitempty"`
	NameGlob     string   `json:"name_glob,omitempty"`
	NameRegex    string   `json:"name_regex,omitempty"`
	NameFuzzy    string   `json:"name_fuzzy,omitempty"`
	MaxEdits     *int     `json:"max_edits,omitempty"`
	MIMETypes    []string `json:"mime_types,omitempty"`
	MinBytes     *int64   `json:"min_bytes,omitempty"`
	MaxBytes     *int64   `json:"max_bytes,omitempty"`
	Folder       string   `json:"folder"`
}

type compiledRule struct {
	Rule
	extensions map[string]bool
	regex      *regexp.Regexp
	mimeTypes  []string
}

type Request struct {
	Source      string `json:"source"`
	Destination string `json:"destination"`
	Rules       []Rule `json:"rules"`
}

type Move struct {
	Name   string `json:"name"`
	Folder string `json:"folder"`
	Status string `json:"status"`
}

type Result struct {
	Moves   []Move   `json:"moves"`
	Moved   int      `json:"moved"`
	Skipped int      `json:"skipped"`
	Errors  []string `json:"errors"`
}

func validate(req Request) (string, string, []compiledRule, error) {
	if req.Source == "" || req.Destination == "" {
		return "", "", nil, errors.New("source and destination are required")
	}
	source, err := filepath.Abs(req.Source)
	if err != nil {
		return "", "", nil, err
	}
	dest, err := filepath.Abs(req.Destination)
	if err != nil {
		return "", "", nil, err
	}
	info, err := os.Stat(source)
	if err != nil {
		return "", "", nil, fmt.Errorf("source: %w", err)
	}
	if !info.IsDir() {
		return "", "", nil, errors.New("source is not a directory")
	}
	if source == dest {
		return "", "", nil, errors.New("source and destination must differ")
	}
	if len(req.Rules) == 0 {
		return "", "", nil, errors.New("at least one rule is required")
	}
	compiled := make([]compiledRule, len(req.Rules))
	for i, rule := range req.Rules {
		folder := filepath.Clean(rule.Folder)
		if folder == "." || folder == ".." || filepath.IsAbs(folder) || strings.HasPrefix(folder, ".."+string(os.PathSeparator)) || folder != rule.Folder {
			return "", "", nil, fmt.Errorf("rule %d: folder must be a relative path below destination", i+1)
		}
		if len(rule.Extensions) == 0 && rule.Name == "" && rule.NameContains == "" && rule.NameGlob == "" && rule.NameRegex == "" && rule.NameFuzzy == "" && len(rule.MIMETypes) == 0 && rule.MinBytes == nil && rule.MaxBytes == nil {
			return "", "", nil, fmt.Errorf("rule %d: at least one matching condition is required", i+1)
		}
		if rule.NameFuzzy == "" && rule.MaxEdits != nil {
			return "", "", nil, fmt.Errorf("rule %d: max_edits requires name_fuzzy", i+1)
		}
		if rule.MaxEdits != nil && (*rule.MaxEdits < 1 || *rule.MaxEdits > 3) {
			return "", "", nil, fmt.Errorf("rule %d: max_edits must be between 1 and 3", i+1)
		}
		compiled[i] = compiledRule{Rule: rule, extensions: make(map[string]bool)}
		for _, ext := range rule.Extensions {
			ext = strings.ToLower(strings.TrimSpace(ext))
			if !strings.HasPrefix(ext, ".") {
				ext = "." + ext
			}
			if ext == "." || strings.ContainsAny(ext, "/\\*? ") {
				return "", "", nil, fmt.Errorf("rule %d: invalid extension %q", i+1, ext)
			}
			compiled[i].extensions[ext] = true
		}
		if rule.NameGlob != "" {
			if _, err := filepath.Match(rule.NameGlob, ""); err != nil {
				return "", "", nil, fmt.Errorf("rule %d: invalid name glob: %w", i+1, err)
			}
		}
		if rule.NameRegex != "" {
			regex, err := regexp.Compile(rule.NameRegex)
			if err != nil {
				return "", "", nil, fmt.Errorf("rule %d: invalid name regex: %w", i+1, err)
			}
			compiled[i].regex = regex
		}
		if rule.MinBytes != nil && *rule.MinBytes < 0 || rule.MaxBytes != nil && *rule.MaxBytes < 0 {
			return "", "", nil, fmt.Errorf("rule %d: size limits cannot be negative", i+1)
		}
		if rule.MinBytes != nil && rule.MaxBytes != nil && *rule.MinBytes > *rule.MaxBytes {
			return "", "", nil, fmt.Errorf("rule %d: minimum size exceeds maximum size", i+1)
		}
		for _, raw := range rule.MIMETypes {
			value := strings.ToLower(strings.TrimSpace(raw))
			if strings.HasSuffix(value, "/*") && !strings.ContainsAny(strings.TrimSuffix(value, "/*"), " */;") && strings.TrimSuffix(value, "/*") != "" {
				compiled[i].mimeTypes = append(compiled[i].mimeTypes, value)
				continue
			}
			parsed, _, err := mime.ParseMediaType(value)
			if err != nil || parsed != value || !strings.Contains(value, "/") {
				return "", "", nil, fmt.Errorf("rule %d: invalid MIME type %q", i+1, raw)
			}
			compiled[i].mimeTypes = append(compiled[i].mimeTypes, value)
		}
	}
	return source, dest, compiled, nil
}

func (rule compiledRule) matches(path, name string, size int64, detectedMIME *string) (bool, error) {
	if len(rule.extensions) > 0 && !rule.extensions[strings.ToLower(filepath.Ext(name))] {
		return false, nil
	}
	if rule.Name != "" && name != rule.Name {
		return false, nil
	}
	if rule.NameContains != "" && !strings.Contains(strings.ToLower(name), strings.ToLower(rule.NameContains)) {
		return false, nil
	}
	if rule.NameGlob != "" {
		matched, _ := filepath.Match(rule.NameGlob, name) // validated before scanning
		if !matched {
			return false, nil
		}
	}
	if rule.regex != nil && !rule.regex.MatchString(name) {
		return false, nil
	}
	if rule.NameFuzzy != "" {
		maxEdits := 1
		if rule.MaxEdits != nil {
			maxEdits = *rule.MaxEdits
		}
		if !withinEditDistance(strings.ToLower(name), strings.ToLower(rule.NameFuzzy), maxEdits) {
			return false, nil
		}
	}
	if rule.MinBytes != nil && size < *rule.MinBytes || rule.MaxBytes != nil && size > *rule.MaxBytes {
		return false, nil
	}
	if len(rule.mimeTypes) > 0 {
		if *detectedMIME == "" {
			value, err := detectMIME(path)
			if err != nil {
				return false, err
			}
			*detectedMIME = value
		}
		for _, pattern := range rule.mimeTypes {
			if pattern == *detectedMIME || strings.HasSuffix(pattern, "/*") && strings.HasPrefix(*detectedMIME, strings.TrimSuffix(pattern, "*")) {
				return true, nil
			}
		}
		return false, nil
	}
	return true, nil
}

// withinEditDistance compares full filenames using Unicode code points.
func withinEditDistance(a, b string, limit int) bool {
	left, right := []rune(a), []rune(b)
	if difference := len(left) - len(right); difference > limit || difference < -limit {
		return false
	}
	previous := make([]int, len(right)+1)
	current := make([]int, len(right)+1)
	for j := range previous {
		previous[j] = j
	}
	for i, leftRune := range left {
		current[0] = i + 1
		rowMin := current[0]
		for j, rightRune := range right {
			cost := 0
			if leftRune != rightRune {
				cost = 1
			}
			current[j+1] = min(previous[j+1]+1, current[j]+1, previous[j]+cost)
			if current[j+1] < rowMin {
				rowMin = current[j+1]
			}
		}
		if rowMin > limit {
			return false
		}
		previous, current = current, previous
	}
	return previous[len(right)] <= limit
}

func detectMIME(path string) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer file.Close()
	var header [512]byte
	n, err := io.ReadFull(file, header[:])
	if err != nil && err != io.EOF && err != io.ErrUnexpectedEOF {
		return "", err
	}
	value, _, err := mime.ParseMediaType(http.DetectContentType(header[:n]))
	return value, err
}

func Preview(req Request) (Result, error) {
	source, dest, compiled, err := validate(req)
	if err != nil {
		return Result{}, err
	}
	entries, err := os.ReadDir(source)
	if err != nil {
		return Result{}, err
	}
	result := Result{Moves: []Move{}, Errors: []string{}}
	for _, entry := range entries {
		name := entry.Name()
		if strings.HasPrefix(name, ".") || entry.Type()&os.ModeSymlink != 0 {
			result.Skipped++
			continue
		}
		info, err := entry.Info()
		if err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("%s: %v", name, err))
			result.Skipped++
			continue
		}
		if !info.Mode().IsRegular() {
			result.Skipped++
			continue
		}
		matched := false
		fileMIME := ""
		for i, rule := range req.Rules {
			applies, err := compiled[i].matches(filepath.Join(source, name), name, info.Size(), &fileMIME)
			if err != nil {
				result.Errors = append(result.Errors, fmt.Sprintf("%s: %v", name, err))
				matched = true
				result.Skipped++
				break
			}
			if !applies {
				continue
			}
			matched = true
			target := filepath.Join(dest, rule.Folder, name)
			status := "ready"
			if _, err := os.Lstat(target); err == nil {
				status = "conflict"
			} else if !errors.Is(err, os.ErrNotExist) {
				status = "error"
			}
			result.Moves = append(result.Moves, Move{Name: name, Folder: rule.Folder, Status: status})
			break
		}
		if !matched {
			result.Skipped++
		}
	}
	return result, nil
}

// Apply repeats the scan so files added after preview are handled by the current rules.
// Copy with O_EXCL prevents overwriting an existing target, including one created during the copy.
func Apply(req Request) (Result, error) {
	result, err := Preview(req)
	if err != nil {
		return Result{}, err
	}
	source, dest, _, _ := validate(req)
	for i := range result.Moves {
		move := &result.Moves[i]
		if move.Status != "ready" {
			result.Skipped++
			continue
		}
		from := filepath.Join(source, move.Name)
		to := filepath.Join(dest, move.Folder, move.Name)
		if err := moveFile(from, to, dest); err != nil {
			move.Status = "error"
			result.Errors = append(result.Errors, fmt.Sprintf("%s: %v", move.Name, err))
			result.Skipped++
			continue
		}
		move.Status = "moved"
		result.Moved++
	}
	return result, nil
}

func moveFile(from, to, destinationRoot string) error {
	in, err := os.Open(from)
	if err != nil {
		return err
	}
	defer in.Close()
	info, err := in.Stat()
	if err != nil {
		return err
	}
	if !info.Mode().IsRegular() {
		return errors.New("source is no longer a regular file")
	}
	if err := ensureTargetDir(destinationRoot, filepath.Dir(to)); err != nil {
		return err
	}
	out, err := os.OpenFile(to, os.O_WRONLY|os.O_CREATE|os.O_EXCL, info.Mode().Perm())
	if err != nil {
		return err
	}
	removeTarget := true
	defer func() {
		if removeTarget {
			_ = os.Remove(to)
		}
	}()
	if _, err = io.Copy(out, in); err != nil {
		_ = out.Close()
		return err
	}
	if err = out.Sync(); err != nil {
		_ = out.Close()
		return err
	}
	if err = out.Close(); err != nil {
		return err
	}
	if err = os.Chtimes(to, info.ModTime(), info.ModTime()); err != nil {
		return err
	}
	current, err := os.Lstat(from)
	if err != nil {
		return err
	}
	opened, err := in.Stat()
	if err != nil {
		return err
	}
	if !os.SameFile(info, current) || opened.Size() != info.Size() || !opened.ModTime().Equal(info.ModTime()) {
		return errors.New("source changed while being copied")
	}
	// Keep the copy if removing the source fails, so the user's data remains available.
	removeTarget = false
	if err = os.Remove(from); err != nil {
		return fmt.Errorf("copy created but source could not be removed: %w", err)
	}
	return nil
}

func ensureTargetDir(root, targetDir string) error {
	if err := os.MkdirAll(root, 0755); err != nil {
		return err
	}
	relative, err := filepath.Rel(root, targetDir)
	if err != nil || relative == ".." || strings.HasPrefix(relative, ".."+string(os.PathSeparator)) {
		return errors.New("target folder is outside destination")
	}
	current := root
	for _, part := range append([]string{""}, strings.Split(relative, string(os.PathSeparator))...) {
		if part != "" && part != "." {
			current = filepath.Join(current, part)
			if err := os.Mkdir(current, 0755); err != nil && !errors.Is(err, os.ErrExist) {
				return err
			}
		}
		info, err := os.Lstat(current)
		if err != nil {
			return err
		}
		if !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
			return errors.New("destination contains a symbolic link or non-directory")
		}
	}
	return nil
}

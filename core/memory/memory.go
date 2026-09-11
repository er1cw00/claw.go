package memory

import (
	"os"
	"path/filepath"

	//	"sort"
	"fmt"
	"strings"
	"time"

	"github.com/er1cw00/claw.go/base/logger"
)

// MemoryStore provides persistent agent memory.
// It supports daily notes (memory/YYYY-MM-DD.md) and long-term memory (MEMORY.md).
type MemoryStore struct {
	workspace  string
	memoryDir  string
	memoryFile string
}

// NewMemoryStore creates a MemoryStore rooted at the given workspace directory.
func NewMemoryStore(workspace string) *MemoryStore {
	memoryDir := filepath.Join(workspace, "memory")
	return &MemoryStore{
		workspace:  workspace,
		memoryDir:  memoryDir,
		memoryFile: filepath.Join(memoryDir, "MEMORY.md"),
	}
}

// ensureDir creates the memory directory if it does not exist.
func (ms *MemoryStore) ensureDir() error {
	return os.MkdirAll(ms.memoryDir, 0755)
}

// todayDate returns today's date formatted as YYYY-MM-DD.
func todayDate() string {
	return time.Now().Format("2006-01-02")
}

// ReadLongTerm reads the long-term memory file (MEMORY.md).
func (ms *MemoryStore) ReadLongTerm() (string, error) {
	content, err := os.ReadFile(ms.memoryFile)
	if err != nil {
		return "", err
	}
	return string(content), nil
}

// WriteLongTerm writes content to the long-term memory file (MEMORY.md).
func (ms *MemoryStore) WriteLongTerm(content string) error {
	if err := ms.ensureDir(); err != nil {
		return err
	}
	return os.WriteFile(ms.memoryFile, []byte(content), 0644)
}

// ReadToday reads today's daily notes file.
func (ms *MemoryStore) ReadToday() string {
	return ms.ReadDaily(todayDate())
}

// ReadDaily reads a specific daily notes file by date (YYYY-MM-DD).
func (ms *MemoryStore) ReadDaily(date string) string {
	path := filepath.Join(ms.memoryDir, date+".md")
	content, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	return string(content)
}

// WriteToday writes content to today's daily notes file.
func (ms *MemoryStore) WriteToday(content string) error {
	return ms.WriteDaily(todayDate(), content)
}

// WriteDaily writes content to a specific daily notes file by date (YYYY-MM-DD).
func (ms *MemoryStore) WriteDaily(date, content string) error {
	if err := ms.ensureDir(); err != nil {
		return err
	}
	path := filepath.Join(ms.memoryDir, date+".md")
	return os.WriteFile(path, []byte(content), 0644)
}

// AppendToday appends a line (with timestamp) to today's memory note file.
func (ms *MemoryStore) AppendToday(text string) error {
	if err := os.MkdirAll(ms.memoryDir, 0o755); err != nil {
		return err
	}
	name := time.Now().UTC().Format("2006-01-02") + ".md"
	path := filepath.Join(ms.memoryDir, name)
	f, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	defer func() { _ = f.Close() }()
	_, err = fmt.Fprintf(f, "[%s] %s\n", time.Now().UTC().Format(time.RFC3339), text)
	return err
}

// GetRecentMemories returns combined daily memory content from the last N days.
func (ms *MemoryStore) GetRecentMemories(days int) string {
	if days <= 0 {
		return ""
	}
	today := time.Now()
	var parts []string
	for i := 0; i < days; i++ {
		date := today.AddDate(0, 0, -i).Format("2006-01-02")
		content := ms.ReadDaily(date)
		if content != "" {
			parts = append(parts, content)
		}
	}
	return strings.Join(parts, "\n\n---\n\n")
}

// AppendHistory appends an entry to the long-term history file (HISTORY.md).
func (ms *MemoryStore) AppendHistory(entry string) error {
	if err := ms.ensureDir(); err != nil {
		return err
	}
	historyFile := filepath.Join(ms.memoryDir, "HISTORY.md")
	content, err := os.ReadFile(historyFile)
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	var sb strings.Builder
	if len(content) > 0 {
		sb.Write(content)
		sb.WriteString("\n\n")
	}
	sb.WriteString(entry)
	return os.WriteFile(historyFile, []byte(sb.String()), 0644)
}

func (ms *MemoryStore) GetMemoryContext() string {
	var parts []string

	longTerm, err := ms.ReadLongTerm()
	if err != nil {
		logger.Errorf("read long term memory fail, err:%v", err)
	}
	if longTerm != "" {
		parts = append(parts, "## Long-term Memory\n"+longTerm)
	}

	today := ms.ReadToday()
	if today != "" {
		parts = append(parts, "## Today's Notes\n"+today)
	}

	return strings.Join(parts, "\n\n")
}

// // ListMemoryFiles returns all daily memory files sorted by date (newest first).
//
//	func (ms *MemoryStore) ListMemoryFiles() []string {
//		entries, err := os.ReadDir(ms.memoryDir)
//		if err != nil {
//			return nil
//		}
//		var files []string
//		for _, entry := range entries {
//			if entry.IsDir() {
//				continue
//			}
//			name := entry.Name()
//			if matched, _ := filepath.Match("????-??-??.md", name); matched {
//				files = append(files, filepath.Join(ms.memoryDir, name))
//			}
//		}
//		sort.Sort(sort.Reverse(sort.StringSlice(files)))
//		return files
//	}
//
// ListFiles returns the filenames of all files in the memory directory.
func (s *MemoryStore) ListFiles() ([]string, error) {
	entries, err := os.ReadDir(s.memoryDir)
	if err != nil {
		if os.IsNotExist(err) {
			return []string{}, nil
		}
		return nil, err
	}
	names := make([]string, 0, len(entries))
	for _, e := range entries {
		if !e.IsDir() {
			names = append(names, e.Name())
		}
	}
	return names, nil
}

// ReadFile reads a named file from the memory directory.
// name must be "MEMORY.md" or a date file "YYYY-MM-DD.md".
// Returns ("", nil) if the file does not exist.
func (s *MemoryStore) ReadFile(name string) (string, error) {
	if !isValidMemoryFile(name) {
		return "", fmt.Errorf("invalid memory filename: %q", name)
	}
	b, err := os.ReadFile(filepath.Join(s.memoryDir, name))
	if err != nil {
		if os.IsNotExist(err) {
			return "", nil
		}
		return "", err
	}
	return string(b), nil
}

// WriteFile writes content to a named file in the memory directory.
// name must be "MEMORY.md" or a date file "YYYY-MM-DD.md".
func (s *MemoryStore) WriteFile(name, content string) error {
	if !isValidMemoryFile(name) {
		return fmt.Errorf("invalid memory filename: %q", name)
	}
	if err := os.MkdirAll(s.memoryDir, 0o755); err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(s.memoryDir, name), []byte(content), 0o644)
}

// DeleteFile deletes a dated memory file (YYYY-MM-DD.md only).
// Long-term memory (MEMORY.md) is protected and cannot be deleted via this method.
func (s *MemoryStore) DeleteFile(name string) error {
	// Only dated files may be deleted — never MEMORY.md.
	if len(name) != 13 || name[4] != '-' || name[7] != '-' || name[10:] != ".md" {
		return fmt.Errorf("delete_memory: only dated files (YYYY-MM-DD) can be deleted, got %q", name)
	}
	if _, err := time.Parse("2006-01-02", name[:10]); err != nil {
		return fmt.Errorf("delete_memory: only dated files (YYYY-MM-DD) can be deleted, got %q", name)
	}
	if err := os.Remove(filepath.Join(s.memoryDir, name)); err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("memory file not found: %q", name)
		}
		return err
	}
	return nil
}

func isValidMemoryFile(name string) bool {
	if name == "MEMORY.md" {
		return true
	}
	// Must be exactly "YYYY-MM-DD.md" (13 chars).
	if len(name) != 13 || name[4] != '-' || name[7] != '-' || name[10:] != ".md" {
		return false
	}
	_, err := time.Parse("2006-01-02", name[:10])
	return err == nil
}

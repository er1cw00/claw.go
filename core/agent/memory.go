package agent

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
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
func (ms *MemoryStore) ReadLongTerm() string {
	content, err := os.ReadFile(ms.memoryFile)
	if err != nil {
		return ""
	}
	return string(content)
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

// ListMemoryFiles returns all daily memory files sorted by date (newest first).
func (ms *MemoryStore) ListMemoryFiles() []string {
	entries, err := os.ReadDir(ms.memoryDir)
	if err != nil {
		return nil
	}
	var files []string
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if matched, _ := filepath.Match("????-??-??.md", name); matched {
			files = append(files, filepath.Join(ms.memoryDir, name))
		}
	}
	sort.Sort(sort.Reverse(sort.StringSlice(files)))
	return files
}

// GetMemoryContext returns formatted memory context including long-term and today's notes.
func (ms *MemoryStore) GetMemoryContext() string {
	var parts []string

	longTerm := ms.ReadLongTerm()
	if longTerm != "" {
		parts = append(parts, "## Long-term Memory\n"+longTerm)
	}

	today := ms.ReadToday()
	if today != "" {
		parts = append(parts, "## Today's Notes\n"+today)
	}

	return strings.Join(parts, "\n\n")
}

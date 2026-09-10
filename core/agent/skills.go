package agent

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"gopkg.in/yaml.v3"
)

// SkillEntry holds basic information about a discovered skill.
type SkillEntry struct {
	Name   string
	Path   string
	Source string
}

// SkillLoader discovers and loads agent skills from markdown files.
// Skills are stored as SKILL.md inside a directory named after the skill.
type SkillsLoader struct {
	workspace       string
	workspaceSkills string
	disabledSkills  map[string]struct{}
}

var (
	// stripSkillFrontmatter matches YAML frontmatter at the start of a markdown file.
	// stripSkillFrontmatter = regexp.MustCompile(
	// 	`^---\s*\r?\n(.*?)\r?\n---\s*\r?\n?`,
	// )
	stripSkillFrontmatter = regexp.MustCompile(
		`(?s)^---\s*\r?\n(.*?)\r?\n---\s*\r?\n?`,
	)
)

// NewSkillLoader creates a new SkillLoader.
func NewSkillsLoader(workspace string, disabledSkills []string) *SkillsLoader {
	disabled := make(map[string]struct{}, len(disabledSkills))
	for _, name := range disabledSkills {
		disabled[name] = struct{}{}
	}
	return &SkillsLoader{
		workspace:       workspace,
		workspaceSkills: filepath.Join(workspace, "skills"),
		disabledSkills:  disabled,
	}
}

func (sl *SkillsLoader) skillEntriesFromDir(base, source string, skipNames map[string]struct{}) []SkillEntry {
	if base == "" {
		return nil
	}
	entries, err := os.ReadDir(base)
	if err != nil {
		return nil
	}
	var result []SkillEntry
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		name := entry.Name()
		if _, ok := skipNames[name]; ok {
			continue
		}
		skillFile := filepath.Join(base, name, "SKILL.md")
		if _, err := os.Stat(skillFile); err != nil {
			continue
		}
		result = append(result, SkillEntry{
			Name:   name,
			Path:   skillFile,
			Source: source,
		})
	}
	return result
}

// ListSkills returns all available skills.
// If filterUnavailable is true, skills with unmet requirements are omitted.
func (sl *SkillsLoader) ListSkills(filterUnavailable bool) []SkillEntry {
	skills := sl.skillEntriesFromDir(sl.workspaceSkills, "workspace", nil)
	workspaceNames := make(map[string]struct{}, len(skills))
	for _, entry := range skills {
		workspaceNames[entry.Name] = struct{}{}
	}

	if len(sl.disabledSkills) > 0 {
		filtered := skills[:0]
		for _, skill := range skills {
			if _, ok := sl.disabledSkills[skill.Name]; !ok {
				filtered = append(filtered, skill)
			}
		}
		skills = filtered
	}

	if !filterUnavailable {
		return skills
	}

	filtered := skills[:0]
	for _, skill := range skills {
		meta := sl.GetSkillMeta(skill.Name)
		if sl.CheckRequirements(meta) {
			filtered = append(filtered, skill)
		}
	}
	return filtered
}

// LoadSkill returns the raw content of a skill's SKILL.md file, or nil if not found.
func (sl *SkillsLoader) LoadSkill(name string) []byte {
	roots := []string{sl.workspaceSkills}
	for _, root := range roots {
		path := filepath.Join(root, name, "SKILL.md")
		if content, err := os.ReadFile(path); err == nil {
			return content
		}
	}
	return nil
}

// LoadSkillsForContext loads the requested skills and returns formatted context content.
func (sl *SkillsLoader) LoadSkillsForContext(skillNames []string) string {
	var parts []string
	for _, name := range skillNames {
		content := sl.LoadSkill(name)
		if content == nil {
			continue
		}
		parts = append(parts, fmt.Sprintf("### Skill: %s\n\n%s", name, sl.StripFrontmatter(content)))
	}
	return strings.Join(parts, "\n\n---\n\n")
}

// BuildSkillsSummary builds a markdown summary of all skills.
func (sl *SkillsLoader) BuildSkillsSummary(exclude map[string]struct{}) string {
	allSkills := sl.ListSkills(false)
	if len(allSkills) == 0 {
		return ""
	}

	var lines []string
	for _, entry := range allSkills {
		if _, ok := exclude[entry.Name]; ok {
			continue
		}
		meta := sl.GetSkillMeta(entry.Name)
		available := sl.CheckRequirements(meta)
		desc := sl.GetSkillDescription(entry.Name)
		if available {
			lines = append(lines, fmt.Sprintf("- **%s** — %s  `%s`", entry.Name, desc, entry.Path))
		} else {
			missing := sl.GetMissingRequirements(meta)
			suffix := ""
			if missing != "" {
				suffix = fmt.Sprintf(" (unavailable: %s)", missing)
			} else {
				suffix = " (unavailable)"
			}
			lines = append(lines, fmt.Sprintf("- **%s** — %s%s  `%s`", entry.Name, desc, suffix, entry.Path))
		}
	}
	return strings.Join(lines, "\n")
}

// GetSkillAvailability returns whether a skill can run and why not when it cannot.
func (sl *SkillsLoader) GetSkillAvailability(name string) (bool, string) {
	meta := sl.GetSkillMeta(name)
	if sl.CheckRequirements(meta) {
		return true, ""
	}
	return false, sl.GetMissingRequirements(meta)
}

// GetSkillRequirements returns explicit command/env requirements and currently missing entries.
func (sl *SkillsLoader) GetSkillRequirements(name string) map[string][]string {
	requires := sl.GetSkillMeta(name).Get("requires")
	bins := getStringList(requires, "bins")
	env := getStringList(requires, "env")
	return map[string][]string{
		"bins":         bins,
		"env":          env,
		"missing_bins": filterMissingBins(bins),
		"missing_env":  filterMissingEnv(env),
	}
}

// GetSkillDescription returns the description from a skill's frontmatter.
func (sl *SkillsLoader) GetSkillDescription(name string) string {
	meta := sl.GetSkillMetadata(name)
	if meta != nil && meta.Description != "" {
		return meta.Description
	}
	return name
}

// StripFrontmatter removes YAML frontmatter from markdown content.
func (sl *SkillsLoader) StripFrontmatter(content []byte) string {
	if !bytesHasPrefix(content, []byte("---")) {
		return string(content)
	}
	loc := stripSkillFrontmatter.FindStringIndex(string(content))
	if loc == nil {
		return string(content)
	}
	return strings.TrimSpace(string(content[loc[1]:]))
}

// CheckRequirements returns true if all skill requirements are met.
func (sl *SkillsLoader) CheckRequirements(meta *SkillMeta) bool {
	if meta == nil {
		return true
	}
	requires := meta.Get("requires")
	for _, cmd := range getStringList(requires, "bins") {
		if !commandExists(cmd) {
			return false
		}
	}
	for _, env := range getStringList(requires, "env") {
		if os.Getenv(env) == "" {
			return false
		}
	}
	return true
}

// GetMissingRequirements returns a description of missing requirements.
func (sl *SkillsLoader) GetMissingRequirements(meta *SkillMeta) string {
	if meta == nil {
		return ""
	}
	requires := meta.Get("requires")
	var missing []string
	for _, cmd := range getStringList(requires, "bins") {
		if !commandExists(cmd) {
			missing = append(missing, fmt.Sprintf("CLI: %s", cmd))
		}
	}
	for _, env := range getStringList(requires, "env") {
		if os.Getenv(env) == "" {
			missing = append(missing, fmt.Sprintf("ENV: %s", env))
		}
	}
	return strings.Join(missing, ", ")
}

// GetSkillMeta returns the parsed nanobot/openclaw metadata for a skill.
func (sl *SkillsLoader) GetSkillMeta(name string) *SkillMeta {
	raw := sl.GetSkillMetadata(name)
	if raw == nil {
		return nil
	}
	return ParseNanobotMetadata(raw.Metadata)
}

// GetAlwaysSkills returns skills marked as always=true that meet requirements.
func (sl *SkillsLoader) GetAlwaysSkills() []string {
	var result []string
	for _, entry := range sl.ListSkills(true) {
		meta := sl.GetSkillMetadata(entry.Name)
		if meta == nil {
			continue
		}
		nano := ParseNanobotMetadata(meta.Metadata)
		if nano.GetBool("always") || meta.Always {
			result = append(result, entry.Name)
		}
	}
	return result
}

// SkillMetadata holds top-level frontmatter fields for a skill.
type SkillMetadata struct {
	Description string         `yaml:"description"`
	Metadata    map[string]any `yaml:"metadata"`
	Always      bool           `yaml:"always"`
}

// GetSkillMetadata parses and returns a skill's frontmatter metadata.
func (sl *SkillsLoader) GetSkillMetadata(name string) *SkillMetadata {
	content := sl.LoadSkill(name)
	if content == nil || !bytesHasPrefix(content, []byte("---")) {
		return nil
	}
	match := stripSkillFrontmatter.FindSubmatch(content)
	if match == nil {
		return nil
	}
	var meta SkillMetadata
	if err := yaml.Unmarshal(match[1], &meta); err != nil {
		return nil
	}
	return &meta
}

// SkillMeta wraps skill metadata and provides safe accessors.
type SkillMeta struct {
	data map[string]any
}

// Get returns a nested map value by key, or nil.
func (sm *SkillMeta) Get(key string) map[string]any {
	if sm == nil || sm.data == nil {
		return nil
	}
	value, ok := sm.data[key].(map[string]any)
	if !ok {
		return nil
	}
	return value
}

// GetBool returns a boolean value by key.
func (sm *SkillMeta) GetBool(key string) bool {
	if sm == nil || sm.data == nil {
		return false
	}
	switch v := sm.data[key].(type) {
	case bool:
		return v
	case string:
		return strings.ToLower(v) == "true"
	default:
		return false
	}
}

// ParseNanobotMetadata extracts nanobot/openclaw metadata from a frontmatter metadata field.
func ParseNanobotMetadata(raw any) *SkillMeta {
	var data map[string]any
	switch v := raw.(type) {
	case map[string]any:
		data = v
	case string:
		if err := yaml.Unmarshal([]byte(v), &data); err != nil {
			return nil
		}
	default:
		return nil
	}
	if data == nil {
		return nil
	}
	payload, ok := data["nanobot"].(map[string]any)
	if !ok {
		payload, ok = data["openclaw"].(map[string]any)
	}
	if !ok {
		return nil
	}
	return &SkillMeta{data: payload}
}

func bytesHasPrefix(s, prefix []byte) bool {
	return len(s) >= len(prefix) && string(s[:len(prefix)]) == string(prefix)
}

func getStringList(m map[string]any, key string) []string {
	if m == nil {
		return nil
	}
	raw, ok := m[key].([]any)
	if !ok {
		return nil
	}
	out := make([]string, 0, len(raw))
	for _, item := range raw {
		if s, ok := item.(string); ok {
			out = append(out, s)
		}
	}
	return out
}

func commandExists(cmd string) bool {
	_, err := os.Stat(cmd)
	if err == nil {
		return true
	}
	paths := strings.Split(os.Getenv("PATH"), string(os.PathListSeparator))
	for _, dir := range paths {
		if _, err := os.Stat(filepath.Join(dir, cmd)); err == nil {
			return true
		}
	}
	return false
}

func filterMissingBins(bins []string) []string {
	var out []string
	for _, cmd := range bins {
		if !commandExists(cmd) {
			out = append(out, cmd)
		}
	}
	return out
}

func filterMissingEnv(envs []string) []string {
	var out []string
	for _, env := range envs {
		if os.Getenv(env) == "" {
			out = append(out, env)
		}
	}
	return out
}

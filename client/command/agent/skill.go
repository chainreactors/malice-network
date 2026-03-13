package agent

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/carapace-sh/carapace"
	"github.com/chainreactors/malice-network/client/core"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

// Skill represents a loaded SKILL.md file with parsed frontmatter and body.
type Skill struct {
	Name        string `yaml:"name"`
	Description string `yaml:"description"`
	Body        string // Markdown content after frontmatter
	Dir         string // directory containing SKILL.md
}

// SkillInfo is a summary returned by DiscoverSkills for listing and completion.
type SkillInfo struct {
	Name        string
	Description string
	Source      string // "local" or "global"
}

// skillSearchPaths returns the local and global skills directories in priority order.
func skillSearchPaths() []struct {
	dir    string
	source string
} {
	paths := []struct {
		dir    string
		source string
	}{
		{filepath.Join(".", "skills"), "local"},
	}
	if home, err := os.UserHomeDir(); err == nil {
		paths = append(paths, struct {
			dir    string
			source string
		}{filepath.Join(home, ".config", "malice", "skills"), "global"})
	}
	return paths
}

// DiscoverSkills scans local and global skills directories, returning all available skills.
// Local skills override global skills with the same name.
func DiscoverSkills() []SkillInfo {
	seen := make(map[string]struct{})
	var skills []SkillInfo

	for _, sp := range skillSearchPaths() {
		entries, err := os.ReadDir(sp.dir)
		if err != nil {
			continue
		}
		for _, entry := range entries {
			if !entry.IsDir() {
				continue
			}
			skillFile := filepath.Join(sp.dir, entry.Name(), "SKILL.md")
			if _, err := os.Stat(skillFile); err != nil {
				continue
			}
			s, err := parseSkillFile(skillFile)
			if err != nil {
				continue
			}
			name := s.Name
			if name == "" {
				name = entry.Name()
			}
			if _, exists := seen[name]; exists {
				continue // local already registered, skip global duplicate
			}
			seen[name] = struct{}{}
			skills = append(skills, SkillInfo{
				Name:        name,
				Description: s.Description,
				Source:      sp.source,
			})
		}
	}
	return skills
}

// LoadSkill loads a skill by name, searching local then global directories.
func LoadSkill(name string) (*Skill, error) {
	for _, sp := range skillSearchPaths() {
		skillFile := filepath.Join(sp.dir, name, "SKILL.md")
		if _, err := os.Stat(skillFile); err != nil {
			continue
		}
		s, err := parseSkillFile(skillFile)
		if err != nil {
			return nil, fmt.Errorf("failed to parse skill %q: %w", name, err)
		}
		if s.Name == "" {
			s.Name = name
		}
		s.Dir = filepath.Join(sp.dir, name)
		return s, nil
	}
	return nil, fmt.Errorf("skill %q not found (searched ./skills/ and ~/.config/malice/skills/)", name)
}

// parseSkillFile reads a SKILL.md file and separates YAML frontmatter from body.
func parseSkillFile(path string) (*Skill, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	content := string(data)
	s := &Skill{}

	scanner := bufio.NewScanner(strings.NewReader(content))
	// Look for opening ---
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "---" {
			break
		}
		if line != "" {
			// No frontmatter, entire content is body
			s.Body = content
			return s, nil
		}
	}

	// Collect frontmatter lines until closing ---
	var fmLines []string
	foundClose := false
	for scanner.Scan() {
		line := scanner.Text()
		if strings.TrimSpace(line) == "---" {
			foundClose = true
			break
		}
		fmLines = append(fmLines, line)
	}

	if !foundClose {
		// No closing ---, treat entire content as body
		s.Body = content
		return s, nil
	}

	// Parse frontmatter YAML
	if len(fmLines) > 0 {
		fmData := strings.Join(fmLines, "\n")
		if err := yaml.Unmarshal([]byte(fmData), s); err != nil {
			return nil, fmt.Errorf("invalid frontmatter YAML: %w", err)
		}
	}

	// Remainder is body
	var bodyLines []string
	for scanner.Scan() {
		bodyLines = append(bodyLines, scanner.Text())
	}
	s.Body = strings.Join(bodyLines, "\n")

	return s, nil
}

var (
	reIndexedArgs = regexp.MustCompile(`\$ARGUMENTS\[(\d+)\]`)
	reShortArgs   = regexp.MustCompile(`\$(\d+)`)
)

// renderSkill performs argument substitution on the skill body.
func renderSkill(skill *Skill, args []string) string {
	joined := strings.Join(args, " ")
	body := skill.Body

	hasArgPlaceholder := strings.Contains(body, "$ARGUMENTS")

	// Replace $ARGUMENTS[N] with the Nth argument
	body = reIndexedArgs.ReplaceAllStringFunc(body, func(match string) string {
		sub := reIndexedArgs.FindStringSubmatch(match)
		if len(sub) < 2 {
			return match
		}
		idx, err := strconv.Atoi(sub[1])
		if err != nil || idx >= len(args) {
			return match
		}
		return args[idx]
	})

	// Replace $N shorthand with the Nth argument
	body = reShortArgs.ReplaceAllStringFunc(body, func(match string) string {
		sub := reShortArgs.FindStringSubmatch(match)
		if len(sub) < 2 {
			return match
		}
		idx, err := strconv.Atoi(sub[1])
		if err != nil || idx >= len(args) {
			return match
		}
		return args[idx]
	})

	// Replace $ARGUMENTS with joined string
	body = strings.ReplaceAll(body, "$ARGUMENTS", joined)

	// If no $ARGUMENTS placeholder existed and args were provided, append them
	if !hasArgPlaceholder && len(args) > 0 {
		body = body + "\nARGUMENTS: " + joined
	}

	return strings.TrimSpace(body)
}

// SkillCmd loads and executes a skill as a poison injection.
func SkillCmd(cmd *cobra.Command, con *core.Console, args []string) error {
	name := args[0]
	skillArgs := args[1:]

	skill, err := LoadSkill(name)
	if err != nil {
		return err
	}

	text := renderSkill(skill, skillArgs)

	session := con.GetInteractive()
	task, err := Poison(con.Rpc, session, text)
	if err != nil {
		return err
	}
	session.Console(task, "skill "+name)
	return nil
}

// SkillListCmd lists all discovered skills.
func SkillListCmd(cmd *cobra.Command, con *core.Console) error {
	skills := DiscoverSkills()
	if len(skills) == 0 {
		fmt.Println("No skills found. Place SKILL.md files in ./skills/<name>/ or ~/.config/malice/skills/<name>/")
		return nil
	}

	// Calculate column widths
	nameWidth := 4 // "NAME"
	descWidth := 11 // "DESCRIPTION"
	for _, s := range skills {
		if len(s.Name) > nameWidth {
			nameWidth = len(s.Name)
		}
		if len(s.Description) > descWidth {
			descWidth = len(s.Description)
		}
	}

	fmtStr := fmt.Sprintf("  %%-%ds  %%-%ds  %%s\n", nameWidth, descWidth)
	fmt.Printf(fmtStr, "NAME", "DESCRIPTION", "SOURCE")
	fmt.Printf(fmtStr, strings.Repeat("─", nameWidth), strings.Repeat("─", descWidth), "──────")
	for _, s := range skills {
		desc := s.Description
		if desc == "" {
			desc = "-"
		}
		fmt.Printf(fmtStr, s.Name, desc, s.Source)
	}
	return nil
}

// SkillNameCompleter returns a carapace.Action that completes skill names.
func SkillNameCompleter() carapace.Action {
	return carapace.ActionCallback(func(c carapace.Context) carapace.Action {
		skills := DiscoverSkills()
		results := make([]string, 0, len(skills)*2)
		for _, s := range skills {
			desc := s.Description
			if desc == "" {
				desc = "skill (" + s.Source + ")"
			}
			results = append(results, s.Name, desc)
		}
		return carapace.ActionValuesDescribed(results...).Tag("skills")
	})
}

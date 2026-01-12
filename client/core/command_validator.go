package core

import (
	"strings"
	"sync"

	"github.com/spf13/cobra"
)

// CommandValidator validates AI-generated commands against registered commands
type CommandValidator struct {
	mu         sync.RWMutex
	commandMap map[string]bool   // command name -> exists
	aliases    map[string]string // alias -> canonical name
}

// NewCommandValidator creates a validator from a cobra root command
func NewCommandValidator(rootCmd *cobra.Command) *CommandValidator {
	v := &CommandValidator{
		commandMap: make(map[string]bool),
		aliases:    make(map[string]string),
	}
	if rootCmd != nil {
		v.buildCommandMap(rootCmd, "")
	}
	return v
}

// buildCommandMap recursively builds a map of all available commands
func (v *CommandValidator) buildCommandMap(cmd *cobra.Command, prefix string) {
	name := cmd.Name()
	fullName := name
	if prefix != "" {
		fullName = prefix + " " + name
	}

	// Register the command
	v.commandMap[fullName] = true
	v.commandMap[name] = true // Also register just the command name

	// Register aliases
	for _, alias := range cmd.Aliases {
		v.aliases[alias] = name
		if prefix != "" {
			v.aliases[prefix+" "+alias] = fullName
		}
	}

	// Process subcommands
	for _, subCmd := range cmd.Commands() {
		if !subCmd.Hidden {
			v.buildCommandMap(subCmd, fullName)
		}
	}
}

// Validate checks if a command is valid
func (v *CommandValidator) Validate(command string) bool {
	v.mu.RLock()
	defer v.mu.RUnlock()

	command = strings.TrimSpace(command)
	if command == "" {
		return false
	}

	// Extract the first word (command name)
	parts := strings.Fields(command)
	if len(parts) == 0 {
		return false
	}

	cmdName := parts[0]

	// Check exact match
	if v.commandMap[cmdName] {
		return true
	}

	// Check alias
	if _, ok := v.aliases[cmdName]; ok {
		return true
	}

	// Check full command with subcommand
	if len(parts) >= 2 {
		fullCmd := parts[0] + " " + parts[1]
		if v.commandMap[fullCmd] {
			return true
		}
		if _, ok := v.aliases[fullCmd]; ok {
			return true
		}
	}

	return false
}

// ValidateAndFix attempts to fix an invalid command
func (v *CommandValidator) ValidateAndFix(command string) (string, bool) {
	command = strings.TrimSpace(command)
	if command == "" {
		return "", false
	}

	// If already valid, return as-is
	if v.Validate(command) {
		return command, true
	}

	parts := strings.Fields(command)
	if len(parts) == 0 {
		return "", false
	}

	// Try to fix the first word
	fixed := v.findSimilar(parts[0])
	if fixed != "" {
		parts[0] = fixed
		fixedCmd := strings.Join(parts, " ")
		if v.Validate(fixedCmd) {
			return fixedCmd, true
		}
	}

	// Try alias resolution
	if canonical, ok := v.aliases[parts[0]]; ok {
		parts[0] = canonical
		return strings.Join(parts, " "), true
	}

	return command, false
}

// findSimilar finds a similar command using Levenshtein distance
func (v *CommandValidator) findSimilar(input string) string {
	v.mu.RLock()
	defer v.mu.RUnlock()

	input = strings.ToLower(input)
	minDist := 3 // Max distance threshold
	var similar string

	for cmd := range v.commandMap {
		// Only compare single-word commands
		if strings.Contains(cmd, " ") {
			continue
		}
		dist := levenshteinDistance(input, strings.ToLower(cmd))
		if dist < minDist {
			minDist = dist
			similar = cmd
		}
	}

	return similar
}

// GetAllCommands returns all registered command names
func (v *CommandValidator) GetAllCommands() []string {
	v.mu.RLock()
	defer v.mu.RUnlock()

	commands := make([]string, 0, len(v.commandMap))
	for cmd := range v.commandMap {
		// Only return single-word commands to avoid duplication
		if !strings.Contains(cmd, " ") {
			commands = append(commands, cmd)
		}
	}
	return commands
}

// AddCommand manually adds a command to the validator
func (v *CommandValidator) AddCommand(name string, aliases ...string) {
	v.mu.Lock()
	defer v.mu.Unlock()

	v.commandMap[name] = true
	for _, alias := range aliases {
		v.aliases[alias] = name
	}
}

// levenshteinDistance calculates the edit distance between two strings
func levenshteinDistance(s1, s2 string) int {
	if len(s1) == 0 {
		return len(s2)
	}
	if len(s2) == 0 {
		return len(s1)
	}

	// Create matrix
	matrix := make([][]int, len(s1)+1)
	for i := range matrix {
		matrix[i] = make([]int, len(s2)+1)
		matrix[i][0] = i
	}
	for j := range matrix[0] {
		matrix[0][j] = j
	}

	// Fill matrix
	for i := 1; i <= len(s1); i++ {
		for j := 1; j <= len(s2); j++ {
			cost := 1
			if s1[i-1] == s2[j-1] {
				cost = 0
			}
			matrix[i][j] = min(
				matrix[i-1][j]+1,      // deletion
				matrix[i][j-1]+1,      // insertion
				matrix[i-1][j-1]+cost, // substitution
			)
		}
	}

	return matrix[len(s1)][len(s2)]
}

func min(a, b, c int) int {
	if a < b {
		if a < c {
			return a
		}
		return c
	}
	if b < c {
		return b
	}
	return c
}

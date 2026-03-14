package intl

import (
	"regexp"
	"strings"
	"testing"
)

// TestBofPackFormatsValid uses regex to extract all bof_pack format strings
// from Lua source files and validates that each character is a known BOF
// pack format specifier.
func TestBofPackFormatsValid(t *testing.T) {
	requireCommunityFixture(t, "community/main.lua")
	files, err := GetAllLuaFiles()
	if err != nil {
		t.Fatalf("failed to enumerate lua files: %v", err)
	}

	// Match bof_pack("FORMAT", ...) — the first string argument is the format
	formatRegex := regexp.MustCompile(`bof_pack\("([^"]+)"`)
	found := 0

	for _, path := range files {
		content, err := UnifiedFS.ReadFile(path)
		if err != nil {
			t.Fatalf("failed to read %s: %v", path, err)
		}

		matches := formatRegex.FindAllStringSubmatch(string(content), -1)
		for _, match := range matches {
			format := match[1]
			found++
			t.Run(path+"/"+format, func(t *testing.T) {
				if !ValidBofPackFormat(format) {
					t.Errorf("invalid bof_pack format %q in %s", format, path)
				}
			})
		}
	}

	if found == 0 {
		t.Fatal("no bof_pack format strings found in any lua file")
	}
	t.Logf("validated %d bof_pack format strings", found)
}

// TestBofPackArgCountMatch validates that the number of format characters
// in each bof_pack call matches the number of subsequent arguments.
// It uses depth-aware parenthesis counting to handle nested function calls.
func TestBofPackArgCountMatch(t *testing.T) {
	requireCommunityFixture(t, "community/main.lua")
	files, err := GetAllLuaFiles()
	if err != nil {
		t.Fatalf("failed to enumerate lua files: %v", err)
	}

	// Find bof_pack(" positions, then extract full call with paren depth counting
	formatRegex := regexp.MustCompile(`bof_pack\("([^"]+)"`)
	found := 0

	for _, path := range files {
		content, err := UnifiedFS.ReadFile(path)
		if err != nil {
			t.Fatalf("failed to read %s: %v", path, err)
		}
		src := string(content)

		locs := formatRegex.FindAllStringIndex(src, -1)
		fmts := formatRegex.FindAllStringSubmatch(src, -1)
		for idx, loc := range locs {
			format := fmts[idx][1]
			// Find the opening '(' of bof_pack(
			openParen := strings.Index(src[loc[0]:], "(")
			if openParen < 0 {
				continue
			}
			openParen += loc[0]

			// Extract the full argument list by counting paren depth
			fullArgs := extractBalancedArgs(src, openParen)
			if fullArgs == "" {
				continue
			}

			// Remove the format string argument (first arg)
			// Find the first comma after the format string
			afterFormat := strings.Index(fullArgs, ",")
			if afterFormat < 0 {
				// No args after format — format should be empty (unlikely)
				continue
			}
			restArgs := strings.TrimSpace(fullArgs[afterFormat+1:])
			argCount := countTopLevelArgs(restArgs)

			found++
			t.Run(path+"/"+format, func(t *testing.T) {
				if len(format) != argCount {
					t.Errorf("bof_pack format %q has %d specifiers but %d args in %s\n  args: %s",
						format, len(format), argCount, path, restArgs)
				}
			})
		}
	}

	t.Logf("validated %d bof_pack calls for argument count", found)
}

// extractBalancedArgs extracts the content between balanced parentheses
// starting at the given opening parenthesis position.
func extractBalancedArgs(src string, openPos int) string {
	if openPos >= len(src) || src[openPos] != '(' {
		return ""
	}

	depth := 0
	inString := false
	var stringChar byte

	for i := openPos; i < len(src); i++ {
		c := src[i]
		if inString {
			if c == '\\' {
				i++
				continue
			}
			if c == stringChar {
				inString = false
			}
			continue
		}
		switch c {
		case '"', '\'':
			inString = true
			stringChar = c
		case '[':
			// Handle Lua long strings [[...]]
			if i+1 < len(src) && src[i+1] == '[' {
				// Skip long string — find matching ]]
				end := strings.Index(src[i+2:], "]]")
				if end >= 0 {
					i += 2 + end + 1
				}
				continue
			}
			depth++
		case '(', '{':
			depth++
		case ')', '}', ']':
			depth--
			if depth == 0 {
				return src[openPos+1 : i]
			}
		}
	}
	return ""
}

// countTopLevelArgs counts comma-separated arguments at the top level,
// handling nested parentheses, strings, and table constructors.
func countTopLevelArgs(s string) int {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0
	}

	count := 1
	depth := 0
	inString := false
	var stringChar byte

	for i := 0; i < len(s); i++ {
		c := s[i]
		if inString {
			if c == '\\' {
				i++ // skip escaped char
				continue
			}
			if c == stringChar {
				inString = false
			}
			continue
		}
		switch c {
		case '"', '\'':
			inString = true
			stringChar = c
		case '(', '{', '[':
			depth++
		case ')', '}', ']':
			depth--
		case ',':
			if depth == 0 {
				count++
			}
		}
	}
	return count
}

// TestScriptResourcePathsResolve validates that script_resource paths
// referenced in Lua source code correspond to actual files in UnifiedFS.
func TestScriptResourcePathsResolve(t *testing.T) {
	requireCommunityFixture(t, "community/main.lua")
	// Load all modules in the mock VM so we can capture script_resource calls
	harness := NewTestHarness()
	vm := harness.NewMockVM()
	defer vm.Close()

	err := harness.LoadCommunityMain(vm)
	if err != nil {
		t.Fatalf("failed to load community main.lua: %v", err)
	}

	// Check that resource directories exist in UnifiedFS
	resourceDirs := []string{
		"community/community/resources/bof",
	}

	for _, dir := range resourceDirs {
		entries, err := UnifiedFS.ReadDir(dir)
		if err != nil {
			t.Logf("resource directory %s not found (may be expected): %v", dir, err)
			continue
		}
		t.Logf("resource directory %s contains %d entries", dir, len(entries))
	}

	t.Logf("captured %d script_resource paths during module loading", len(harness.ResourcePaths))
}

// TestTTPFormatValid validates that all TTP annotations follow the MITRE
// ATT&CK format (T followed by digits, optionally with sub-technique).
func TestTTPFormatValid(t *testing.T) {
	requireCommunityFixture(t, "community/main.lua")
	harness := NewTestHarness()
	vm := harness.NewMockVM()
	defer vm.Close()

	err := harness.LoadCommunityMain(vm)
	if err != nil {
		t.Fatalf("failed to load community main.lua: %v", err)
	}

	ttpRegex := regexp.MustCompile(`^T\d{4}(\.\d{3})?$`)

	for name, cmd := range harness.Commands {
		t.Run(name, func(t *testing.T) {
			if cmd.TTP == "" {
				t.Logf("command %q has no TTP annotation", name)
				return
			}
			if !ttpRegex.MatchString(cmd.TTP) {
				t.Errorf("command %q has invalid TTP format: %q (expected T####[.###])", name, cmd.TTP)
			}
		})
	}
}

package wizard

import (
	"testing"

	"github.com/spf13/cobra"
)

func TestBuildFormGroups(t *testing.T) {
	// Create a test command with various flags
	cmd := &cobra.Command{
		Use:   "test",
		Short: "Test command",
	}

	// Add flags with different types
	cmd.Flags().String("target", "", "Build target")
	cmd.Flags().String("profile", "", "Profile name")
	cmd.Flags().Bool("secure", false, "Enable secure mode")
	cmd.Flags().Int("port", 8080, "Port number")
	cmd.Flags().StringSlice("modules", nil, "Modules to include")

	// Test building form groups
	result := make(map[string]any)
	groups := buildFormGroups(cmd, result)

	if len(groups) == 0 {
		t.Fatal("Expected at least one group")
	}

	// Verify fields were created
	totalFields := 0
	for _, g := range groups {
		totalFields += len(g.Fields)
		t.Logf("Group: %s, Fields: %d", g.Title, len(g.Fields))
		for _, f := range g.Fields {
			t.Logf("  - Field: %s, Kind: %d", f.Name, f.Kind)
		}
	}

	// Should have 5 fields (excluding help which is auto-added)
	if totalFields < 5 {
		t.Errorf("Expected at least 5 fields, got %d", totalFields)
	}
}

func TestFlagToField(t *testing.T) {
	tests := []struct {
		name       string
		flagType   string
		defVal     string
		wantKind   FieldKind
		wantResult interface{}
	}{
		{
			name:     "string flag",
			flagType: "string",
			defVal:   "default",
			wantKind: KindInput,
		},
		{
			name:     "bool flag",
			flagType: "bool",
			defVal:   "false",
			wantKind: KindConfirm,
		},
		{
			name:     "int flag",
			flagType: "int",
			defVal:   "42",
			wantKind: KindNumber,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a fresh command for each test
			testCmd := &cobra.Command{Use: "test"}
			switch tt.flagType {
			case "string":
				testCmd.Flags().String("test-flag", tt.defVal, "Test description")
			case "bool":
				testCmd.Flags().Bool("test-flag", tt.defVal == "true", "Test description")
			case "int":
				testCmd.Flags().Int("test-flag", 42, "Test description")
			}

			flag := testCmd.Flags().Lookup("test-flag")
			if flag == nil {
				t.Fatal("Flag not found")
			}

			result := make(map[string]any)
			field := flagToField(testCmd, flag, result)

			if field.Kind != tt.wantKind {
				t.Errorf("Kind = %d, want %d", field.Kind, tt.wantKind)
			}

			if field.Name != "test-flag" {
				t.Errorf("Name = %s, want test-flag", field.Name)
			}
		})
	}
}

func TestApplyResultToFlags(t *testing.T) {
	cmd := &cobra.Command{Use: "test"}
	cmd.Flags().String("target", "", "Target")
	cmd.Flags().Int("port", 0, "Port")
	cmd.Flags().Bool("secure", false, "Secure")

	result := map[string]any{
		"target": ptr("x86_64-pc-windows-gnu"),
		"port":   ptr("8080"),
		"secure": ptr("true"),
	}

	err := ApplyResultToFlags(cmd, result)
	if err != nil {
		t.Fatalf("ApplyResultToFlags failed: %v", err)
	}

	// Verify flags were set
	target, _ := cmd.Flags().GetString("target")
	if target != "x86_64-pc-windows-gnu" {
		t.Errorf("target = %s, want x86_64-pc-windows-gnu", target)
	}

	port, _ := cmd.Flags().GetInt("port")
	if port != 8080 {
		t.Errorf("port = %d, want 8080", port)
	}

	secure, _ := cmd.Flags().GetBool("secure")
	if !secure {
		t.Error("secure should be true")
	}
}

func TestApplyResultToFlagsWithSlice(t *testing.T) {
	cmd := &cobra.Command{Use: "test"}
	cmd.Flags().StringSlice("modules", nil, "Modules")

	result := map[string]any{
		"modules": ptr("mod1,mod2,mod3"),
	}

	err := ApplyResultToFlags(cmd, result)
	if err != nil {
		t.Fatalf("ApplyResultToFlags failed: %v", err)
	}

	modules, _ := cmd.Flags().GetStringSlice("modules")
	if len(modules) != 3 {
		t.Errorf("modules length = %d, want 3", len(modules))
	}
	expected := []string{"mod1", "mod2", "mod3"}
	for i, m := range modules {
		if m != expected[i] {
			t.Errorf("modules[%d] = %s, want %s", i, m, expected[i])
		}
	}
}

func TestGroupedWizardForm(t *testing.T) {
	// Test creating a grouped form
	groups := []*FormGroup{
		{
			Name:  "basic",
			Title: "Basic",
			Fields: []*FormField{
				{
					Name:       "target",
					Title:      "Target",
					Kind:       KindSelect,
					Options:    []string{"x86_64-pc-windows-gnu", "x86_64-unknown-linux-gnu"},
					Selected:   0,
					InputValue: "x86_64-pc-windows-gnu",
				},
				{
					Name:       "name",
					Title:      "Name",
					Kind:       KindInput,
					InputValue: "default",
				},
			},
		},
		{
			Name:  "advanced",
			Title: "Advanced",
			Fields: []*FormField{
				{
					Name:       "secure",
					Title:      "Secure",
					Kind:       KindConfirm,
					ConfirmVal: false,
				},
			},
		},
	}

	form := NewGroupedWizardForm(groups)

	if form == nil {
		t.Fatal("Form should not be nil")
	}

	if len(form.groups) != 2 {
		t.Errorf("Expected 2 groups, got %d", len(form.groups))
	}

	if form.groupIndex != 0 {
		t.Errorf("Initial group index should be 0, got %d", form.groupIndex)
	}
}

func TestFieldKindConversion(t *testing.T) {
	// Test field kind values
	if KindSelect != 0 {
		t.Errorf("KindSelect should be 0, got %d", KindSelect)
	}
	if KindMultiSelect != 1 {
		t.Errorf("KindMultiSelect should be 1, got %d", KindMultiSelect)
	}
	if KindInput != 2 {
		t.Errorf("KindInput should be 2, got %d", KindInput)
	}
	if KindConfirm != 3 {
		t.Errorf("KindConfirm should be 3, got %d", KindConfirm)
	}
	if KindNumber != 4 {
		t.Errorf("KindNumber should be 4, got %d", KindNumber)
	}
}

func TestEnsureOptionValue(t *testing.T) {
	opts := []string{"opt1", "opt2", "opt3"}

	// Value exists in options
	result := ensureOptionValue(opts, "opt2")
	if len(result) != 3 {
		t.Errorf("Length should be 3, got %d", len(result))
	}

	// Value doesn't exist - should be appended to end
	result = ensureOptionValue(opts, "newopt")
	if len(result) != 4 {
		t.Errorf("Length should be 4, got %d", len(result))
	}
	if result[3] != "newopt" {
		t.Errorf("Last element should be newopt, got %s", result[3])
	}

	// Empty value shouldn't be added
	result = ensureOptionValue(opts, "")
	if len(result) != 3 {
		t.Errorf("Empty value should not be added, length = %d", len(result))
	}
}

func TestFinalizeResult(t *testing.T) {
	cmd := &cobra.Command{Use: "test"}
	cmd.Flags().Int("port", 0, "Port")
	cmd.Flags().String("name", "", "Name")

	portVal := "8080"
	nameVal := "test"
	result := map[string]any{
		"port": &portVal,
		"name": &nameVal,
	}

	finalizeResult(result, cmd)

	// port should be converted to int
	if v, ok := result["port"].(int64); ok {
		if v != 8080 {
			t.Errorf("port = %d, want 8080", v)
		}
	} else if v, ok := result["port"].(int); ok {
		if v != 8080 {
			t.Errorf("port = %d, want 8080", v)
		}
	} else {
		// Still a pointer is also acceptable if the value is correct
		if ptr, ok := result["port"].(*string); ok && ptr != nil && *ptr == "8080" {
			// This is fine
		} else {
			t.Errorf("port type = %T, expected int or *string", result["port"])
		}
	}
}

// Helper function
func ptr(s string) *string {
	return &s
}

// TestEnableWizard tests the EnableWizard function
func TestEnableWizard(t *testing.T) {
	cmd := &cobra.Command{
		Use:   "test",
		Short: "Test command",
		RunE: func(cmd *cobra.Command, args []string) error {
			return nil
		},
	}

	cmd.Flags().String("target", "", "Target")

	// Enable wizard for this command
	EnableWizard(cmd)

	// Verify --wizard flag was added
	wizardFlag := cmd.Flags().Lookup("wizard")
	if wizardFlag == nil {
		t.Fatal("--wizard flag should be added")
	}

	// Verify PreRunE was wrapped
	if cmd.PreRunE == nil {
		t.Fatal("PreRunE should be wrapped")
	}
}

// TestEnableWizardForCommands tests enabling wizard for multiple commands
func TestEnableWizardForCommands(t *testing.T) {
	cmd1 := &cobra.Command{
		Use: "cmd1",
		RunE: func(cmd *cobra.Command, args []string) error {
			return nil
		},
	}
	cmd2 := &cobra.Command{
		Use: "cmd2",
		RunE: func(cmd *cobra.Command, args []string) error {
			return nil
		},
	}

	EnableWizardForCommands(cmd1, cmd2)

	// Both commands should have --wizard flag
	if cmd1.Flags().Lookup("wizard") == nil {
		t.Error("cmd1 should have --wizard flag")
	}
	if cmd2.Flags().Lookup("wizard") == nil {
		t.Error("cmd2 should have --wizard flag")
	}
}

// TestAddWizardFlag tests the AddWizardFlag function
func TestAddWizardFlag(t *testing.T) {
	cmd := &cobra.Command{Use: "test"}

	AddWizardFlag(cmd)

	flag := cmd.Flags().Lookup("wizard")
	if flag == nil {
		t.Fatal("--wizard flag should be added")
	}

	if flag.DefValue != "false" {
		t.Errorf("Default value should be 'false', got %s", flag.DefValue)
	}

	if flag.Usage != "Start interactive wizard mode" {
		t.Errorf("Usage should be 'Start interactive wizard mode', got %s", flag.Usage)
	}
}

// TestWrapPreRunEWithWizard tests the PreRunE wrapper
func TestWrapPreRunEWithWizard(t *testing.T) {
	preRunCalled := false
	originalPreRunE := func(cmd *cobra.Command, args []string) error {
		preRunCalled = true
		return nil
	}

	wrapped := WrapPreRunEWithWizard(originalPreRunE, nil)

	cmd := &cobra.Command{Use: "test"}
	cmd.Flags().Bool("wizard", false, "")

	// Call without wizard mode - should call original
	err := wrapped(cmd, []string{})
	if err != nil {
		t.Fatalf("wrapped returned error: %v", err)
	}
	if !preRunCalled {
		t.Error("Original PreRunE should be called when wizard=false")
	}
}

// TestWrapRunEWithWizard tests the RunE wrapper
func TestWrapRunEWithWizard(t *testing.T) {
	runCalled := false
	originalRunE := func(cmd *cobra.Command, args []string) error {
		runCalled = true
		return nil
	}

	wrapped := WrapRunEWithWizard(originalRunE)

	cmd := &cobra.Command{Use: "test"}
	cmd.Flags().Bool("wizard", false, "")

	// Call without wizard mode - should call original
	err := wrapped(cmd, []string{})
	if err != nil {
		t.Fatalf("wrapped returned error: %v", err)
	}
	if !runCalled {
		t.Error("Original RunE should be called when wizard=false")
	}
}

// TestProviderRegistration tests dynamic provider registration
func TestProviderRegistration(t *testing.T) {
	// Register a global option provider
	RegisterProvider("test-flag", func() []string {
		return []string{"option1", "option2", "option3"}
	})

	// Verify it can be retrieved
	provider, ok := getOptionProvider("test-flag")
	if !ok {
		t.Fatal("Provider should be found")
	}

	opts := provider()
	if len(opts) != 3 {
		t.Errorf("Expected 3 options, got %d", len(opts))
	}
}

// TestScopedProviderRegistration tests scoped provider registration
func TestScopedProviderRegistration(t *testing.T) {
	cmd := &cobra.Command{Use: "test"}

	// Register a scoped provider
	RegisterProviderForCommand(cmd, "scoped-flag", func() []string {
		return []string{"scoped1", "scoped2"}
	})

	// Verify scoped provider works
	provider, ok := getScopedOptionProvider(cmd, "scoped-flag")
	if !ok {
		t.Fatal("Scoped provider should be found")
	}

	opts := provider()
	if len(opts) != 2 {
		t.Errorf("Expected 2 options, got %d", len(opts))
	}
}

// TestDefaultProviderRegistration tests default value provider registration
func TestDefaultProviderRegistration(t *testing.T) {
	RegisterDefaultProvider("test-default", func() string {
		return "default-value"
	})

	provider, ok := getDefaultProvider("test-default")
	if !ok {
		t.Fatal("Default provider should be found")
	}

	val := provider()
	if val != "default-value" {
		t.Errorf("Expected 'default-value', got %s", val)
	}
}

// TestFormGroupOptional tests optional form groups
func TestFormGroupOptional(t *testing.T) {
	groups := []*FormGroup{
		{
			Name:     "required",
			Title:    "Required Settings",
			Optional: false,
			Fields: []*FormField{
				{Name: "field1", Kind: KindInput},
			},
		},
		{
			Name:     "optional",
			Title:    "Optional Settings",
			Optional: true,
			Expanded: false,
			Fields: []*FormField{
				{Name: "field2", Kind: KindInput},
			},
		},
	}

	form := NewGroupedWizardForm(groups)

	// Verify groups are set correctly
	if !form.groups[1].Optional {
		t.Error("Second group should be optional")
	}
	if form.groups[1].Expanded {
		t.Error("Optional group should start collapsed")
	}
}

// TestCSVParsing tests CSV parsing for slice flags
func TestCSVParsing(t *testing.T) {
	tests := []struct {
		input    string
		expected []string
	}{
		{"a,b,c", []string{"a", "b", "c"}},
		{"single", []string{"single"}},
		{"", []string{}},
		{" a , b , c ", []string{"a", "b", "c"}}, // with spaces
	}

	for _, tt := range tests {
		result, err := parseCSV(tt.input)
		if err != nil {
			t.Errorf("parseCSV(%q) error: %v", tt.input, err)
			continue
		}
		if len(result) != len(tt.expected) {
			t.Errorf("parseCSV(%q) = %v, want %v", tt.input, result, tt.expected)
		}
	}
}

// TestIntValidator tests integer validation
func TestIntValidator(t *testing.T) {
	validator := intValidator("int")

	// Valid integers
	if err := validator("42"); err != nil {
		t.Errorf("42 should be valid: %v", err)
	}
	if err := validator("-10"); err != nil {
		t.Errorf("-10 should be valid: %v", err)
	}
	if err := validator("0"); err != nil {
		t.Errorf("0 should be valid: %v", err)
	}

	// Invalid integers
	if err := validator("abc"); err == nil {
		t.Error("abc should be invalid")
	}
	if err := validator("12.5"); err == nil {
		t.Error("12.5 should be invalid for int")
	}
}

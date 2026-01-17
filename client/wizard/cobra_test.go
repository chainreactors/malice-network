package wizard

import (
	"testing"

	"github.com/spf13/cobra"
)

func TestCobraToWizard(t *testing.T) {
	// Create a test command with various flag types
	cmd := &cobra.Command{
		Use:   "test",
		Short: "Test command",
		Long:  "This is a test command for wizard conversion",
	}

	// Add flags of different types
	cmd.Flags().String("name", "", "Name of the target")
	cmd.Flags().Int("port", 8080, "Port number")
	cmd.Flags().Bool("verbose", false, "Enable verbose output")
	cmd.Flags().StringSlice("tags", nil, "Tags for the target")
	cmd.Flags().Float64("timeout", 30.0, "Timeout in seconds")

	// Mark some as required
	cmd.MarkFlagRequired("name")

	// Add group annotations
	nameFlag := cmd.Flags().Lookup("name")
	nameFlag.Annotations = map[string][]string{
		"ui:group": {"Basic"},
		"ui:order": {"1"},
	}

	portFlag := cmd.Flags().Lookup("port")
	portFlag.Annotations = map[string][]string{
		"ui:group": {"Network"},
		"ui:order": {"2"},
	}

	// Convert to wizard
	wiz := CobraToWizard(cmd)

	// Verify wizard was created
	if wiz == nil {
		t.Fatal("CobraToWizard returned nil")
	}

	// Verify title
	if wiz.Title != "Test command" {
		t.Errorf("Expected title 'Test command', got '%s'", wiz.Title)
	}

	// Verify description
	if wiz.Description != "This is a test command for wizard conversion" {
		t.Errorf("Expected description to be set, got '%s'", wiz.Description)
	}

	// Verify groups were created (Basic, Network, and "General" for ungrouped)
	if len(wiz.Groups) == 0 {
		t.Error("Expected groups to be created")
	}

	// Verify fields were converted
	if len(wiz.Fields) != 5 { // name, port, verbose, tags, timeout (help is skipped)
		t.Errorf("Expected 5 fields, got %d", len(wiz.Fields))
	}

	// Verify field types
	nameField := wiz.GetField("name")
	if nameField == nil {
		t.Fatal("Expected 'name' field to exist")
	}
	if nameField.Type != FieldInput {
		t.Errorf("Expected name field to be FieldInput, got %d", nameField.Type)
	}

	portField := wiz.GetField("port")
	if portField == nil {
		t.Fatal("Expected 'port' field to exist")
	}
	if portField.Type != FieldNumber {
		t.Errorf("Expected port field to be FieldNumber, got %d", portField.Type)
	}
	if portField.Default != 8080 {
		t.Errorf("Expected port default to be 8080, got %v", portField.Default)
	}

	verboseField := wiz.GetField("verbose")
	if verboseField == nil {
		t.Fatal("Expected 'verbose' field to exist")
	}
	if verboseField.Type != FieldConfirm {
		t.Errorf("Expected verbose field to be FieldConfirm, got %d", verboseField.Type)
	}

	timeoutField := wiz.GetField("timeout")
	if timeoutField == nil {
		t.Fatal("Expected 'timeout' field to exist")
	}
	if timeoutField.Type != FieldInput {
		t.Errorf("Expected timeout field to be FieldInput (float), got %d", timeoutField.Type)
	}
	if def, ok := timeoutField.Default.(string); !ok || def == "" {
		t.Errorf("Expected timeout default to be a non-empty string, got %T/%v", timeoutField.Default, timeoutField.Default)
	}

	tagsField := wiz.GetField("tags")
	if tagsField == nil {
		t.Fatal("Expected 'tags' field to exist")
	}
	if tagsField.Type != FieldInput {
		t.Errorf("Expected tags field to be FieldInput (slice), got %d", tagsField.Type)
	}
	if def, ok := tagsField.Default.(string); !ok || def != "" {
		t.Errorf("Expected tags default to be empty string, got %T/%v", tagsField.Default, tagsField.Default)
	}
}

func TestCobraToWizardWithOptions(t *testing.T) {
	cmd := &cobra.Command{
		Use:   "select-test",
		Short: "Test select options",
	}

	cmd.Flags().String("format", "json", "Output format")

	formatFlag := cmd.Flags().Lookup("format")
	formatFlag.Annotations = map[string][]string{
		"ui:options": {"json", "yaml", "xml"},
	}

	wiz := CobraToWizard(cmd)
	if wiz == nil {
		t.Fatal("CobraToWizard returned nil")
	}

	formatField := wiz.GetField("format")
	if formatField == nil {
		t.Fatal("Expected 'format' field to exist")
	}

	if formatField.Type != FieldSelect {
		t.Errorf("Expected format field to be FieldSelect, got %d", formatField.Type)
	}

	if len(formatField.Options) != 3 {
		t.Errorf("Expected 3 options, got %d", len(formatField.Options))
	}
}

func TestApplyWizardResultToFlags(t *testing.T) {
	cmd := &cobra.Command{
		Use: "apply-test",
	}

	cmd.Flags().String("name", "", "Name")
	cmd.Flags().Int("port", 0, "Port")
	cmd.Flags().Bool("verbose", false, "Verbose")

	result := NewWizardResult("test")
	result.Set("name", "test-value")
	result.Set("port", 9090)
	result.Set("verbose", true)

	err := ApplyWizardResultToFlags(cmd, result)
	if err != nil {
		t.Fatalf("ApplyWizardResultToFlags failed: %v", err)
	}

	// Verify values were applied
	nameVal, _ := cmd.Flags().GetString("name")
	if nameVal != "test-value" {
		t.Errorf("Expected name to be 'test-value', got '%s'", nameVal)
	}

	portVal, _ := cmd.Flags().GetInt("port")
	if portVal != 9090 {
		t.Errorf("Expected port to be 9090, got %d", portVal)
	}

	verboseVal, _ := cmd.Flags().GetBool("verbose")
	if !verboseVal {
		t.Error("Expected verbose to be true")
	}
}

func TestApplyWizardResultToFlags_UnchangedDoesNotFlipChanged(t *testing.T) {
	cmd := &cobra.Command{
		Use: "apply-unchanged-test",
	}

	cmd.Flags().Int("port", 8080, "Port")

	result := NewWizardResult("test")
	result.Set("port", 8080)

	err := ApplyWizardResultToFlags(cmd, result)
	if err != nil {
		t.Fatalf("ApplyWizardResultToFlags failed: %v", err)
	}

	if cmd.Flags().Changed("port") {
		t.Fatalf("expected port flag to remain unchanged when wizard value equals current")
	}
}

func TestApplyWizardResultToFlags_SliceReplaceNotAppend(t *testing.T) {
	cmd := &cobra.Command{
		Use: "apply-slice-test",
	}

	cmd.Flags().StringSlice("tags", []string{}, "Tags")
	_ = cmd.Flags().Set("tags", "a") // simulate CLI-provided value (flag becomes changed)

	result := NewWizardResult("test")
	result.Set("tags", "b,c")

	err := ApplyWizardResultToFlags(cmd, result)
	if err != nil {
		t.Fatalf("ApplyWizardResultToFlags failed: %v", err)
	}

	tags, _ := cmd.Flags().GetStringSlice("tags")
	if len(tags) != 2 || tags[0] != "b" || tags[1] != "c" {
		t.Fatalf("expected tags to be replaced with [b c], got %#v", tags)
	}
}

func TestSkipFlags(t *testing.T) {
	cmd := &cobra.Command{
		Use: "skip-test",
	}

	cmd.Flags().String("name", "", "Name")
	cmd.Flags().Bool("help", false, "Help")     // Should be skipped
	cmd.Flags().Bool("wizard", false, "Wizard") // Should be skipped

	wiz := CobraToWizard(cmd)
	if wiz == nil {
		t.Fatal("CobraToWizard returned nil")
	}

	// Should only have 'name' field
	if len(wiz.Fields) != 1 {
		t.Errorf("Expected 1 field, got %d", len(wiz.Fields))
	}

	if wiz.GetField("help") != nil {
		t.Error("Expected 'help' field to be skipped")
	}

	if wiz.GetField("wizard") != nil {
		t.Error("Expected 'wizard' field to be skipped")
	}
}

package plugin

import (
	"encoding/json"
	"testing"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

// TestCommandSchema_GenerateSchema tests basic schema generation
func TestCommandSchema_GenerateSchema(t *testing.T) {
	// Create a test command with flags
	cmd := &cobra.Command{
		Use:   "test",
		Short: "Test command",
		Long:  "This is a test command for schema generation",
	}

	// Add various types of flags
	cmd.Flags().String("name", "", "Name parameter")
	cmd.Flags().Int("count", 10, "Count parameter")
	cmd.Flags().Bool("verbose", false, "Verbose output")
	cmd.Flags().StringSlice("tags", []string{}, "Tags list")

	// Create Command wrapper
	testCmd := &Command{
		Name:    "test",
		Command: cmd,
	}

	// Generate schema
	schema, err := testCmd.GenerateSchema()
	if err != nil {
		t.Fatalf("Failed to generate schema: %v", err)
	}

	// Verify basic structure
	if schema.Type != "object" {
		t.Errorf("Expected type 'object', got '%s'", schema.Type)
	}

	if schema.Title != "test" {
		t.Errorf("Expected title 'test', got '%s'", schema.Title)
	}

	// Verify properties
	if len(schema.Properties) != 4 {
		t.Errorf("Expected 4 properties, got %d", len(schema.Properties))
	}

	// Check string property
	if prop, ok := schema.Properties["name"]; ok {
		if prop.Type != "string" {
			t.Errorf("Expected 'name' type to be 'string', got '%s'", prop.Type)
		}
	} else {
		t.Error("Property 'name' not found")
	}

	// Check integer property
	if prop, ok := schema.Properties["count"]; ok {
		if prop.Type != "integer" {
			t.Errorf("Expected 'count' type to be 'integer', got '%s'", prop.Type)
		}
		if prop.Default != 10 {
			t.Errorf("Expected 'count' default to be 10, got %v", prop.Default)
		}
	} else {
		t.Error("Property 'count' not found")
	}

	// Check boolean property
	if prop, ok := schema.Properties["verbose"]; ok {
		if prop.Type != "boolean" {
			t.Errorf("Expected 'verbose' type to be 'boolean', got '%s'", prop.Type)
		}
		if prop.Default != false {
			t.Errorf("Expected 'verbose' default to be false, got %v", prop.Default)
		}
	} else {
		t.Error("Property 'verbose' not found")
	}

	// Check array property
	if prop, ok := schema.Properties["tags"]; ok {
		if prop.Type != "array" {
			t.Errorf("Expected 'tags' type to be 'array', got '%s'", prop.Type)
		}
	} else {
		t.Error("Property 'tags' not found")
	}
}

// TestCommandSchema_WithAnnotations tests schema generation with UI annotations
func TestCommandSchema_WithAnnotations(t *testing.T) {
	cmd := &cobra.Command{
		Use:   "execute",
		Short: "Execute command",
		Annotations: map[string]string{
			"mal":   "basic",
			"ttp":   "T1059",
			"opsec": "3",
		},
	}

	// Add flag with annotations
	flags := pflag.NewFlagSet("test", pflag.ContinueOnError)
	flags.String("command", "", "Command to execute")

	// Set annotations on the flag
	flag := flags.Lookup("command")
	if flag != nil {
		flag.Annotations = map[string][]string{
			"ui:widget":      {"textarea"},
			"ui:group":       {"Basic"},
			"ui:order":       {"1"},
			"ui:placeholder": {"whoami"},
			"ui:required":    {"true"},
		}
	}

	cmd.Flags().AddFlagSet(flags)

	testCmd := &Command{
		Name:    "execute",
		Command: cmd,
		Example: "execute whoami",
	}

	schema, err := testCmd.GenerateSchema()
	if err != nil {
		t.Fatalf("Failed to generate schema: %v", err)
	}

	// Verify metadata
	if schema.XMetadata == nil {
		t.Fatal("Expected metadata to be present")
	}

	if schema.XMetadata.PluginName != "basic" {
		t.Errorf("Expected plugin name 'basic', got '%s'", schema.XMetadata.PluginName)
	}

	if schema.XMetadata.TTP != "T1059" {
		t.Errorf("Expected TTP 'T1059', got '%s'", schema.XMetadata.TTP)
	}

	if schema.XMetadata.Opsec != 3 {
		t.Errorf("Expected Opsec 3, got %d", schema.XMetadata.Opsec)
	}

	// Verify UI annotations
	if prop, ok := schema.Properties["command"]; ok {
		if widget, ok := prop.AdditionalProperties["ui:widget"]; !ok || widget != "textarea" {
			t.Errorf("Expected ui:widget 'textarea', got %v", widget)
		}

		if group, ok := prop.AdditionalProperties["ui:group"]; !ok || group != "Basic" {
			t.Errorf("Expected ui:group 'Basic', got %v", group)
		}

		if order, ok := prop.AdditionalProperties["ui:order"]; !ok || order != 1 {
			t.Errorf("Expected ui:order 1, got %v", order)
		}

		if placeholder, ok := prop.AdditionalProperties["ui:placeholder"]; !ok || placeholder != "whoami" {
			t.Errorf("Expected ui:placeholder 'whoami', got %v", placeholder)
		}
	} else {
		t.Error("Property 'command' not found")
	}

	// Verify required field
	found := false
	for _, req := range schema.Required {
		if req == "command" {
			found = true
			break
		}
	}
	if !found {
		t.Error("Expected 'command' to be in required fields")
	}
}

// TestCommandSchema_ToJSON tests JSON serialization
func TestCommandSchema_ToJSON(t *testing.T) {
	cmd := &cobra.Command{
		Use:   "test",
		Short: "Test command",
	}

	cmd.Flags().String("name", "default", "Name parameter")

	testCmd := &Command{
		Name:    "test",
		Command: cmd,
	}

	schema, err := testCmd.GenerateSchema()
	if err != nil {
		t.Fatalf("Failed to generate schema: %v", err)
	}

	jsonStr, err := schema.ToJSON()
	if err != nil {
		t.Fatalf("Failed to convert schema to JSON: %v", err)
	}

	println(jsonStr)
	// Verify it's valid JSON
	var result map[string]interface{}
	if err := json.Unmarshal([]byte(jsonStr), &result); err != nil {
		t.Fatalf("Generated JSON is invalid: %v", err)
	}

	// Verify structure
	if result["type"] != "object" {
		t.Errorf("Expected type 'object' in JSON, got %v", result["type"])
	}

	if properties, ok := result["properties"].(map[string]interface{}); ok {
		if _, ok := properties["name"]; !ok {
			t.Error("Property 'name' not found in JSON")
		}
	} else {
		t.Error("Properties not found in JSON")
	}
}

// TestPropertySchema_MarshalJSON tests custom JSON marshaling with additional properties
func TestPropertySchema_MarshalJSON(t *testing.T) {
	prop := &PropertySchema{
		Type:        "string",
		Title:       "test",
		Description: "Test property",
		Default:     "default value",
		AdditionalProperties: map[string]interface{}{
			"ui:widget":      "textarea",
			"ui:placeholder": "Enter text",
			"ui:order":       1,
		},
	}

	data, err := json.Marshal(prop)
	if err != nil {
		t.Fatalf("Failed to marshal PropertySchema: %v", err)
	}

	var result map[string]interface{}
	if err := json.Unmarshal(data, &result); err != nil {
		t.Fatalf("Failed to unmarshal JSON: %v", err)
	}

	// Verify standard fields
	if result["type"] != "string" {
		t.Errorf("Expected type 'string', got %v", result["type"])
	}

	// Verify additional properties are included
	if result["ui:widget"] != "textarea" {
		t.Errorf("Expected ui:widget 'textarea', got %v", result["ui:widget"])
	}

	if result["ui:placeholder"] != "Enter text" {
		t.Errorf("Expected ui:placeholder 'Enter text', got %v", result["ui:placeholder"])
	}

	// Verify numeric value is preserved
	if order, ok := result["ui:order"].(float64); !ok || int(order) != 1 {
		t.Errorf("Expected ui:order 1, got %v", result["ui:order"])
	}
}

// TestGenerateSchemasFromCommands tests schema generation from cobra commands
func TestGenerateSchemasFromCommands(t *testing.T) {
	// Create test commands
	cmd1 := &cobra.Command{
		Use:   "cmd1",
		Short: "Command 1",
	}
	cmd1.Flags().String("param1", "", "Parameter 1")

	cmd2 := &cobra.Command{
		Use:   "cmd2",
		Short: "Command 2",
	}
	cmd2.Flags().Int("param2", 0, "Parameter 2")

	// Generate schemas using unified API: []*cobra.Command -> schemas
	schemas, err := GenerateSchemasFromCommands([]*cobra.Command{cmd1, cmd2})
	if err != nil {
		t.Fatalf("Failed to generate schemas: %v", err)
	}

	// Verify commands
	if len(schemas) != 2 {
		t.Errorf("Expected 2 commands, got %d", len(schemas))
	}

	if _, ok := schemas["cmd1"]; !ok {
		t.Error("Command 'cmd1' not found")
	}

	if _, ok := schemas["cmd2"]; !ok {
		t.Error("Command 'cmd2' not found")
	}
}

// TestSchemasToJSON tests schemas JSON serialization
func TestSchemasToJSON(t *testing.T) {
	cmd := &cobra.Command{
		Use:   "test",
		Short: "Test command",
	}
	cmd.Flags().String("param", "", "Parameter")

	// Generate schemas using unified API: []*cobra.Command -> schemas
	schemas, err := GenerateSchemasFromCommands([]*cobra.Command{cmd})
	if err != nil {
		t.Fatalf("Failed to generate schemas: %v", err)
	}

	// Convert to JSON
	jsonData, err := json.MarshalIndent(schemas, "", "  ")
	if err != nil {
		t.Fatalf("Failed to convert to JSON: %v", err)
	}

	// Verify it's valid JSON
	var result map[string]interface{}
	if err := json.Unmarshal(jsonData, &result); err != nil {
		t.Fatalf("Generated JSON is invalid: %v", err)
	}

	// Verify structure
	if _, ok := result["test"]; !ok {
		t.Error("Command 'test' not found in JSON")
	}
}

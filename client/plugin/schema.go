package plugin

import (
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

// CommandSchema represents the JSON Schema for a command (including UI information)
type CommandSchema struct {
	Type        string                     `json:"type"`
	Title       string                     `json:"title,omitempty"`
	Description string                     `json:"description,omitempty"`
	Properties  map[string]*PropertySchema `json:"properties"`
	Required    []string                   `json:"required,omitempty"`
	XMetadata   *CommandMetadata           `json:"x-metadata,omitempty"`
}

// PropertySchema represents a property in the JSON Schema
type PropertySchema struct {
	Type        string      `json:"type"`
	Title       string      `json:"title,omitempty"`
	Description string      `json:"description,omitempty"`
	Default     interface{} `json:"default,omitempty"`
	Enum        []string    `json:"enum,omitempty"`
	Pattern     string      `json:"pattern,omitempty"`
	MinLength   *int        `json:"minLength,omitempty"`
	MaxLength   *int        `json:"maxLength,omitempty"`
	Minimum     *float64    `json:"minimum,omitempty"`
	Maximum     *float64    `json:"maximum,omitempty"`

	// Additional properties for UI hints (populated from flag annotations)
	AdditionalProperties map[string]interface{} `json:"-"`
}

// MarshalJSON implements custom JSON marshaling to include additional properties
func (ps *PropertySchema) MarshalJSON() ([]byte, error) {
	m := make(map[string]interface{})

	// Standard JSON Schema fields
	m["type"] = ps.Type
	if ps.Title != "" {
		m["title"] = ps.Title
	}
	if ps.Description != "" {
		m["description"] = ps.Description
	}
	if ps.Default != nil {
		m["default"] = ps.Default
	}
	if len(ps.Enum) > 0 {
		m["enum"] = ps.Enum
	}
	if ps.Pattern != "" {
		m["pattern"] = ps.Pattern
	}
	if ps.MinLength != nil {
		m["minLength"] = ps.MinLength
	}
	if ps.MaxLength != nil {
		m["maxLength"] = ps.MaxLength
	}
	if ps.Minimum != nil {
		m["minimum"] = ps.Minimum
	}
	if ps.Maximum != nil {
		m["maximum"] = ps.Maximum
	}

	// Add UI hints from annotations
	for k, v := range ps.AdditionalProperties {
		m[k] = v
	}

	return json.Marshal(m)
}

// CommandMetadata represents metadata for a command
type CommandMetadata struct {
	Name        string            `json:"name"`
	PluginName  string            `json:"plugin,omitempty"`
	Source      string            `json:"source,omitempty"` // golang/alias/extension/mal
	TTP         string            `json:"ttp,omitempty"`
	Opsec       int               `json:"opsec,omitempty"`
	Example     string            `json:"example,omitempty"`
	Annotations map[string]string `json:"annotations,omitempty"`
}

// GenerateSchema generates JSON Schema for a command
func (cmd *Command) GenerateSchema() (*CommandSchema, error) {
	if cmd.Command == nil {
		return nil, fmt.Errorf("command is nil")
	}

	schema := &CommandSchema{
		Type:        "object",
		Title:       cmd.Command.Use,
		Description: cmd.Command.Short,
		Properties:  make(map[string]*PropertySchema),
		Required:    []string{},
		XMetadata:   extractMetadata(cmd),
	}

	// Extract schema from flags
	cmd.Command.Flags().VisitAll(func(flag *pflag.Flag) {
		propSchema := createPropertySchema(flag)
		schema.Properties[flag.Name] = propSchema

		// Determine if required
		if isRequired(flag) {
			schema.Required = append(schema.Required, flag.Name)
		}
	})

	return schema, nil
}

// extractMetadata extracts command metadata from annotations
func extractMetadata(cmd *Command) *CommandMetadata {
	metadata := &CommandMetadata{
		Name:        cmd.Name,
		Example:     cmd.Example,
		Annotations: cmd.Command.Annotations,
	}

	if mal, ok := cmd.Command.Annotations["mal"]; ok {
		metadata.PluginName = mal
	}
	if source, ok := cmd.Command.Annotations["source"]; ok {
		metadata.Source = source
	}
	if ttp, ok := cmd.Command.Annotations["ttp"]; ok {
		metadata.TTP = ttp
	}
	if opsec, ok := cmd.Command.Annotations["opsec"]; ok {
		if level, err := strconv.Atoi(opsec); err == nil {
			metadata.Opsec = level
		}
	}

	return metadata
}

// createPropertySchema creates a PropertySchema from a pflag.Flag with default UI hints
func createPropertySchema(flag *pflag.Flag) *PropertySchema {
	propSchema := &PropertySchema{
		Title:                flag.Name,
		Description:          flag.Usage,
		AdditionalProperties: make(map[string]interface{}),
	}

	// Set type and default value
	setTypeAndDefault(propSchema, flag)

	// Apply default UI hints based on type
	applyDefaultUIHints(propSchema, flag)

	// Extract custom annotations (overrides defaults)
	extractAnnotations(propSchema, flag)

	return propSchema
}

// setTypeAndDefault sets the JSON Schema type and default value
func setTypeAndDefault(propSchema *PropertySchema, flag *pflag.Flag) {
	switch flag.Value.Type() {
	case "bool":
		propSchema.Type = "boolean"
		if flag.DefValue != "" {
			propSchema.Default = flag.DefValue == "true"
		}
	case "int", "int32", "int64":
		propSchema.Type = "integer"
		if flag.DefValue != "" {
			if val, err := strconv.Atoi(flag.DefValue); err == nil {
				propSchema.Default = val
			}
		}
	case "float", "float32", "float64":
		propSchema.Type = "number"
		if flag.DefValue != "" {
			if val, err := strconv.ParseFloat(flag.DefValue, 64); err == nil {
				propSchema.Default = val
			}
		}
	case "stringSlice", "stringArray":
		propSchema.Type = "array"
	default: // string
		propSchema.Type = "string"
		if flag.DefValue != "" {
			propSchema.Default = flag.DefValue
		}
	}
}

// applyDefaultUIHints applies default UI hints based on field type and characteristics
func applyDefaultUIHints(propSchema *PropertySchema, flag *pflag.Flag) {
	switch propSchema.Type {
	case "boolean":
		propSchema.AdditionalProperties["ui:widget"] = "checkbox"

	case "integer":
		propSchema.AdditionalProperties["ui:widget"] = "updown"

	case "number":
		propSchema.AdditionalProperties["ui:widget"] = "updown"

	case "array":
		propSchema.AdditionalProperties["ui:widget"] = "tags"

	case "string":
		// Default to text, but use textarea for long descriptions or specific names
		if len(flag.Usage) > 50 ||
			flag.Name == "command" ||
			flag.Name == "script" ||
			flag.Name == "code" {
			propSchema.AdditionalProperties["ui:widget"] = "textarea"
		} else {
			propSchema.AdditionalProperties["ui:widget"] = "text"
		}
	}
}

// extractAnnotations extracts and applies custom annotations from flag
func extractAnnotations(propSchema *PropertySchema, flag *pflag.Flag) {
	for key, values := range flag.Annotations {
		if len(values) == 0 {
			continue
		}

		if len(values) == 1 {
			value := values[0]
			// Parse numeric values for specific keys
			if key == "ui:order" || key == "ui:min" || key == "ui:max" {
				if numVal, err := strconv.Atoi(value); err == nil {
					propSchema.AdditionalProperties[key] = numVal
					continue
				}
				if floatVal, err := strconv.ParseFloat(value, 64); err == nil {
					propSchema.AdditionalProperties[key] = floatVal
					continue
				}
			}
			// Store as string
			propSchema.AdditionalProperties[key] = value
		} else {
			// Multiple values, store as array
			propSchema.AdditionalProperties[key] = values
		}
	}
}

// isRequired determines if a flag is required
func isRequired(flag *pflag.Flag) bool {
	// Check for explicit ui:required annotation
	if requiredAnnotation, ok := flag.Annotations["ui:required"]; ok && len(requiredAnnotation) > 0 {
		return requiredAnnotation[0] == "true"
	}
	// Default: required if no default value and not optional
	return flag.NoOptDefVal == "" && flag.DefValue == ""
}

// ToJSON exports CommandSchema as JSON string
func (schema *CommandSchema) ToJSON() (string, error) {
	data, err := json.MarshalIndent(schema, "", "  ")
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// GenerateSchemasFromCommands generates schemas from a list of cobra commands
// This is the unified API for schema generation from []*cobra.Command
func GenerateSchemasFromCommands(commands []*cobra.Command) (map[string]*CommandSchema, error) {
	schemas := make(map[string]*CommandSchema)

	for _, cmd := range commands {
		if cmd == nil {
			continue
		}

		// Create Command wrapper
		pluginCmd := &Command{
			Name:    cmd.Name(),
			Command: cmd,
			Example: cmd.Example,
		}

		// Generate schema
		schema, err := pluginCmd.GenerateSchema()
		if err != nil {
			continue
		}

		schemas[cmd.Name()] = schema
	}

	return schemas, nil
}

// Lua API functions for setting flag annotations
// These functions are registered as builtin functions with ui_ prefix

// SetFlagUI sets multiple UI hints at once
// Usage: ui_set(flag, {widget="textarea", group="Basic", placeholder="Enter text"})
func SetFlagUI(flag *pflag.Flag, options map[string]string) error {
	if flag == nil {
		return fmt.Errorf("flag is nil")
	}

	if flag.Annotations == nil {
		flag.Annotations = make(map[string][]string)
	}

	for key, value := range options {
		// Add ui: prefix if not present
		if key != "" && key[0] != 'u' {
			key = "ui:" + key
		}
		flag.Annotations[key] = []string{value}
	}

	return nil
}

// SetFlagWidget sets the widget type
// Usage: ui_widget(flag, "textarea")
func SetFlagWidget(flag *pflag.Flag, widget string) error {
	if flag == nil {
		return fmt.Errorf("flag is nil")
	}
	return setFlagAnnotation(flag, "ui:widget", widget)
}

// SetFlagGroup sets the field group
// Usage: ui_group(flag, "Basic Settings")
func SetFlagGroup(flag *pflag.Flag, group string) error {
	if flag == nil {
		return fmt.Errorf("flag is nil")
	}
	return setFlagAnnotation(flag, "ui:group", group)
}

// SetFlagPlaceholder sets the placeholder text
// Usage: ui_placeholder(flag, "Enter command")
func SetFlagPlaceholder(flag *pflag.Flag, placeholder string) error {
	if flag == nil {
		return fmt.Errorf("flag is nil")
	}
	return setFlagAnnotation(flag, "ui:placeholder", placeholder)
}

// SetFlagRequired sets whether the field is required
// Usage: ui_required(flag, true)
func SetFlagRequired(flag *pflag.Flag, required bool) error {
	if flag == nil {
		return fmt.Errorf("flag is nil")
	}
	if required {
		return setFlagAnnotation(flag, "ui:required", "true")
	}
	return setFlagAnnotation(flag, "ui:required", "false")
}

// SetFlagRange sets the numeric range (min, max)
// Usage: ui_range(flag, 1, 100)
func SetFlagRange(flag *pflag.Flag, min, max float64) error {
	if flag == nil {
		return fmt.Errorf("flag is nil")
	}
	if err := setFlagAnnotation(flag, "ui:min", fmt.Sprintf("%v", min)); err != nil {
		return err
	}
	return setFlagAnnotation(flag, "ui:max", fmt.Sprintf("%v", max))
}

// SetFlagOrder sets the field order
// Usage: ui_order(flag, 1)
func SetFlagOrder(flag *pflag.Flag, order int) error {
	if flag == nil {
		return fmt.Errorf("flag is nil")
	}
	return setFlagAnnotation(flag, "ui:order", fmt.Sprintf("%d", order))
}

// setFlagAnnotation is a helper function to set a single annotation on a flag
func setFlagAnnotation(flag *pflag.Flag, key, value string) error {
	if flag.Annotations == nil {
		flag.Annotations = make(map[string][]string)
	}
	flag.Annotations[key] = []string{value}
	return nil
}

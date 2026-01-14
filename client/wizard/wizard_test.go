package wizard

import (
	"testing"
)

func TestWizardBuilder(t *testing.T) {
	wiz := NewWizard("test_wizard", "Test Wizard")
	wiz.WithDescription("Test description").
		Input("name", "Enter name", "default_name").
		Select("option", "Select option", []string{"a", "b", "c"}).
		Number("count", "Enter count", 10).
		Confirm("proceed", "Proceed?", true)

	if wiz.ID != "test_wizard" {
		t.Errorf("Expected ID 'test_wizard', got '%s'", wiz.ID)
	}

	if wiz.Title != "Test Wizard" {
		t.Errorf("Expected Title 'Test Wizard', got '%s'", wiz.Title)
	}

	if wiz.Description != "Test description" {
		t.Errorf("Expected Description 'Test description', got '%s'", wiz.Description)
	}

	if len(wiz.Fields) != 4 {
		t.Errorf("Expected 4 fields, got %d", len(wiz.Fields))
	}

	// Check field types
	expectedTypes := []FieldType{FieldInput, FieldSelect, FieldNumber, FieldConfirm}
	for i, f := range wiz.Fields {
		if f.Type != expectedTypes[i] {
			t.Errorf("Field %d: Expected type %d, got %d", i, expectedTypes[i], f.Type)
		}
	}
}

func TestWizardResult(t *testing.T) {
	result := NewWizardResult("test")

	// Test string value
	strVal := "test_string"
	result.Values["str"] = &strVal
	if result.GetString("str") != "test_string" {
		t.Errorf("GetString failed")
	}

	// Test bool value
	boolVal := true
	result.Values["bool"] = &boolVal
	if !result.GetBool("bool") {
		t.Errorf("GetBool failed")
	}

	// Test int value (from string)
	intStrVal := "42"
	result.Values["int"] = &intStrVal
	if result.GetInt("int") != 42 {
		t.Errorf("GetInt failed, got %d", result.GetInt("int"))
	}
}

func TestWizardTemplates(t *testing.T) {
	templates := ListTemplates()
	if len(templates) == 0 {
		t.Error("Expected some templates, got none")
	}

	// Test getting a known template
	wiz, ok := GetTemplate("listener_setup")
	if !ok {
		t.Error("Expected to find 'listener_setup' template")
	}
	if wiz == nil {
		t.Error("Template wizard is nil")
	}
}

func TestAllTemplatesRegistered(t *testing.T) {
	expectedTemplates := []string{
		// Existing
		"listener_setup",
		"tcp_pipeline",
		"http_pipeline",
		"profile_create",
		// Build
		"build_beacon",
		"build_pulse",
		"build_prelude",
		"build_module",
		// Pipeline
		"bind_pipeline",
		"rem_pipeline",
		// Certificate
		"cert_generate",
		"cert_import",
		// Config
		"github_config",
		"notify_config",
		// Composite
		"infrastructure_setup",
	}

	templates := ListTemplates()
	if len(templates) != len(expectedTemplates) {
		t.Errorf("Expected %d templates, got %d", len(expectedTemplates), len(templates))
	}

	for _, name := range expectedTemplates {
		wiz, ok := GetTemplate(name)
		if !ok {
			t.Errorf("Template '%s' not found", name)
			continue
		}
		if wiz == nil {
			t.Errorf("Template '%s' is nil", name)
			continue
		}
		if wiz.ID != name {
			t.Errorf("Template '%s' has wrong ID: %s", name, wiz.ID)
		}
		if wiz.Title == "" {
			t.Errorf("Template '%s' has empty title", name)
		}
		if len(wiz.Fields) == 0 {
			t.Errorf("Template '%s' has no fields", name)
		}
	}
}

func TestWizardClone(t *testing.T) {
	wiz := NewWizard("original", "Original").
		Input("field1", "Field 1", "").
		Select("field2", "Field 2", []string{"a", "b"})

	clone := wiz.Clone()

	if clone.ID != wiz.ID {
		t.Error("Clone ID mismatch")
	}

	if len(clone.Fields) != len(wiz.Fields) {
		t.Error("Clone fields count mismatch")
	}

	// Modify clone and ensure original is unchanged
	clone.ID = "modified"
	if wiz.ID == "modified" {
		t.Error("Modifying clone affected original")
	}

	// Verify parent rebinding: clone's fields should point to clone, not original
	for i, f := range clone.Fields {
		if f.parent != clone {
			t.Errorf("Clone field %d has wrong parent (expected clone, got original)", i)
		}
	}

	// Verify original's fields still point to original
	for i, f := range wiz.Fields {
		if f.parent != wiz {
			t.Errorf("Original field %d has wrong parent after clone", i)
		}
	}

	// Test that chaining on clone affects clone, not original
	clone.Field().SetRequired()
	if wiz.Fields[len(wiz.Fields)-1].Required {
		t.Error("SetRequired on clone affected original")
	}
	if !clone.Fields[len(clone.Fields)-1].Required {
		t.Error("SetRequired on clone did not set Required")
	}
}

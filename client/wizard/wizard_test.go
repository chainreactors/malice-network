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
}

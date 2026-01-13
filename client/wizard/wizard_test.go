package wizard

import (
	"errors"
	"reflect"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/huh"
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

func TestHorizontalMultiSelect(t *testing.T) {
	options := []string{"nano", "full", "base", "extend"}
	var selected []string

	m := NewHorizontalMultiSelect(options).
		Title("Modules").
		Value(&selected).
		Key("modules")

	// Test initial state
	if m.title != "Modules" {
		t.Errorf("Expected title 'Modules', got '%s'", m.title)
	}
	if len(m.options) != 4 {
		t.Errorf("Expected 4 options, got %d", len(m.options))
	}

	// Test selection toggle
	m.selected[0] = true
	m.selected[2] = true
	vals := m.getSelectedValues()
	if len(vals) != 2 {
		t.Errorf("Expected 2 selected values, got %d", len(vals))
	}
	if vals[0] != "nano" || vals[1] != "base" {
		t.Errorf("Selected values mismatch: %v", vals)
	}

	// Test updateValue syncs to pointer
	m.updateValue()
	if len(selected) != 2 {
		t.Errorf("Value pointer not updated, got %d items", len(selected))
	}

	// Test GetValue
	val := m.GetValue()
	if valSlice, ok := val.([]string); !ok || len(valSlice) != 2 {
		t.Errorf("GetValue failed: %v", val)
	}

	// Test GetKey
	if m.GetKey() != "modules" {
		t.Errorf("GetKey failed: %s", m.GetKey())
	}

	// Test View renders without panic
	view := m.View()
	if view == "" {
		t.Error("View returned empty string")
	}
	if !strings.Contains(view, "Modules") {
		t.Error("View missing title")
	}
	// Single-line carousel style: only shows current option (cursor at 0 = "nano")
	if !strings.Contains(view, "nano") {
		t.Error("View missing current option")
	}
	// Should show position indicator
	if !strings.Contains(view, "(1/4)") {
		t.Error("View missing position indicator")
	}
}

func TestHorizontalMultiSelectValidationBlocksNavigation(t *testing.T) {
	options := []string{"nano", "full", "base"}
	var selected []string

	m := NewHorizontalMultiSelect(options).
		Title("Modules").
		Value(&selected).
		Validate(func(values []string) error {
			if len(values) == 0 {
				return errors.New("required")
			}
			return nil
		})

	// Enter with empty selection should not advance.
	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd != nil {
		t.Fatalf("expected no navigation command on validation error")
	}
	if m.err == nil {
		t.Fatalf("expected validation error to be set")
	}

	// Select one item then Enter should advance.
	m.Update(tea.KeyMsg{Type: tea.KeySpace})
	_, cmd = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd == nil {
		t.Fatalf("expected navigation command after valid selection")
	}
	if got, want := reflect.TypeOf(cmd()), reflect.TypeOf(huh.NextField()); got != want {
		t.Fatalf("expected NextField msg, got %v", got)
	}

	// shift+tab should go back (and still validate).
	_, cmd = m.Update(tea.KeyMsg{Type: tea.KeyShiftTab})
	if cmd == nil {
		t.Fatalf("expected navigation command for shift+tab")
	}
	if got, want := reflect.TypeOf(cmd()), reflect.TypeOf(huh.PrevField()); got != want {
		t.Fatalf("expected PrevField msg, got %v", got)
	}
}

func TestHorizontalSelectValidationBlocksNavigation(t *testing.T) {
	options := []string{"a", "b"}
	val := ""

	m := NewHorizontalSelect(options).
		Title("Build type").
		Value(&val).
		Validate(func(s string) error {
			if s == "a" {
				return errors.New("invalid")
			}
			return nil
		})

	// Enter on invalid selection should not advance.
	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd != nil {
		t.Fatalf("expected no navigation command on validation error")
	}
	if m.err == nil {
		t.Fatalf("expected validation error to be set")
	}

	// Move to valid option then Enter should advance.
	m.Update(tea.KeyMsg{Type: tea.KeyRight})
	_, cmd = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd == nil {
		t.Fatalf("expected navigation command after valid selection")
	}
	if got, want := reflect.TypeOf(cmd()), reflect.TypeOf(huh.NextField()); got != want {
		t.Fatalf("expected NextField msg, got %v", got)
	}
}

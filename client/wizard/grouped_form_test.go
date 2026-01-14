package wizard

import (
	"fmt"
	"testing"
)

func TestGroupedWizardFormStructure(t *testing.T) {
	// Get build_beacon wizard
	w, ok := GetTemplate("build_beacon")
	if !ok {
		t.Fatal("build_beacon template not found")
	}

	if !w.IsGrouped() {
		t.Fatal("build_beacon should be grouped")
	}

	// Print group structure
	fmt.Println("=== Build Beacon Wizard Groups ===")
	for i, g := range w.Groups {
		fmt.Printf("\n[%d] %s - %s\n", i+1, g.Title, g.Description)
		for j, f := range g.Fields {
			fmt.Printf("    %d.%d %s (%v)\n", i+1, j+1, f.Title, f.Type)
		}
	}
	fmt.Printf("\nTotal: %d groups, %d fields\n", len(w.Groups), len(w.Fields))
}

func TestFormGroupConversion(t *testing.T) {
	// Create a simple grouped wizard
	w := NewWizard("test", "Test Wizard").
		NewGroup("g1", "Group 1").
		WithDescription("First group").
		Select("field1", "Field 1", []string{"a", "b", "c"}).Field().EndGroup().
		Input("field2", "Field 2", "default").Field().EndGroup().
		End().
		NewGroup("g2", "Group 2").
		WithDescription("Second group").
		Confirm("field3", "Field 3", true).Field().EndGroup().
		Number("field4", "Field 4", 42).Field().EndGroup().
		End()

	if len(w.Groups) != 2 {
		t.Errorf("Expected 2 groups, got %d", len(w.Groups))
	}

	if len(w.Groups[0].Fields) != 2 {
		t.Errorf("Expected 2 fields in group 1, got %d", len(w.Groups[0].Fields))
	}

	if len(w.Groups[1].Fields) != 2 {
		t.Errorf("Expected 2 fields in group 2, got %d", len(w.Groups[1].Fields))
	}

	if len(w.Fields) != 4 {
		t.Errorf("Expected 4 total fields, got %d", len(w.Fields))
	}
}

func TestGroupedFormInit(t *testing.T) {
	groups := []*FormGroup{
		{
			Name:        "g1",
			Title:       "Group 1",
			Description: "First group",
			Fields: []*FormField{
				{
					Name:    "f1",
					Title:   "Field 1",
					Kind:    KindSelect,
					Options: []string{"opt1", "opt2", "opt3"},
				},
			},
		},
		{
			Name:        "g2",
			Title:       "Group 2",
			Description: "Second group",
			Fields: []*FormField{
				{
					Name:  "f2",
					Title: "Field 2",
					Kind:  KindInput,
				},
			},
		},
	}

	form := NewGroupedWizardForm(groups)
	form.Init()

	if form.groupIndex != 0 {
		t.Errorf("Expected groupIndex 0, got %d", form.groupIndex)
	}

	if form.fieldIndex != 0 {
		t.Errorf("Expected fieldIndex 0, got %d", form.fieldIndex)
	}
}

func TestIsGroupComplete_RequiredSelectEmpty(t *testing.T) {
	form := NewGroupedWizardForm([]*FormGroup{{
		Name:  "g1",
		Title: "Group 1",
		Fields: []*FormField{{
			Name:     "f1",
			Title:    "Field 1",
			Kind:     KindSelect,
			Options:  []string{""},
			Selected: 0,
			Required: true,
		}},
	}})

	if form.isGroupComplete(0) {
		t.Fatal("expected group to be incomplete when required select is empty")
	}
}

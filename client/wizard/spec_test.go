package wizard

import (
	"testing"

	lua "github.com/yuin/gopher-lua"
)

func TestNewWizardFromSpec(t *testing.T) {
	spec := &WizardSpec{
		ID:          "spec_wizard",
		Title:       "Spec Wizard",
		Description: "From spec",
		Fields: []FieldSpec{
			{Name: "host", Title: "Host", Type: "input", Default: "0.0.0.0", Required: true},
			{Name: "protocol", Title: "Protocol", Type: "select", Options: []string{"tcp", "http"}, Default: "http"},
			{Name: "modules", Title: "Modules", Type: "multiselect", Options: []string{"a", "b"}, Default: []interface{}{"a"}},
			{Name: "tls", Title: "TLS", Type: "confirm", Default: true},
			{Name: "port", Title: "Port", Type: "number", Default: float64(443)},
		},
	}

	wiz, err := NewWizardFromSpec(spec)
	if err != nil {
		t.Fatalf("NewWizardFromSpec failed: %v", err)
	}
	if wiz.ID != "spec_wizard" {
		t.Fatalf("expected ID spec_wizard, got %q", wiz.ID)
	}
	if wiz.Description != "From spec" {
		t.Fatalf("expected description %q, got %q", "From spec", wiz.Description)
	}
	if len(wiz.Fields) != 5 {
		t.Fatalf("expected 5 fields, got %d", len(wiz.Fields))
	}
	if wiz.Fields[0].Required != true {
		t.Fatalf("expected host.required=true")
	}
	if _, ok := wiz.Fields[4].Default.(int); !ok {
		t.Fatalf("expected port.default to be int, got %T", wiz.Fields[4].Default)
	}
}

func TestRegisterTemplateFromSpec_SetsID(t *testing.T) {
	name := "spec_template_test"
	t.Cleanup(func() {
		templatesMu.Lock()
		delete(Templates, name)
		templatesMu.Unlock()
	})

	spec := &WizardSpec{
		Title: "Template",
		Fields: []FieldSpec{
			{Name: "x", Title: "X", Type: "input"},
		},
	}
	if err := RegisterTemplateFromSpec(name, spec); err != nil {
		t.Fatalf("RegisterTemplateFromSpec failed: %v", err)
	}

	wiz, ok := GetTemplate(name)
	if !ok || wiz == nil {
		t.Fatalf("expected to load registered template")
	}
	if wiz.ID != name {
		t.Fatalf("expected template wizard ID to default to %q, got %q", name, wiz.ID)
	}
}

func TestNewWizardFromFile_EmbedPath(t *testing.T) {
	wiz, err := NewWizardFromFile("embed://community/resources/testdata/wizard_spec_test.yaml")
	if err != nil {
		t.Fatalf("NewWizardFromFile(embed://) failed: %v", err)
	}
	if wiz.ID != "embed_wizard_spec_test" {
		t.Fatalf("expected ID embed_wizard_spec_test, got %q", wiz.ID)
	}
	if wiz.Title == "" {
		t.Fatalf("expected non-empty title")
	}
	if len(wiz.Fields) != 2 {
		t.Fatalf("expected 2 fields, got %d", len(wiz.Fields))
	}
	if !wiz.Fields[0].Required {
		t.Fatalf("expected host.required=true")
	}
	if wiz.Fields[1].Type != FieldNumber {
		t.Fatalf("expected port.type=FieldNumber, got %v", wiz.Fields[1].Type)
	}
	if _, ok := wiz.Fields[1].Default.(int); !ok {
		t.Fatalf("expected port.default to be int, got %T", wiz.Fields[1].Default)
	}
}

func TestLuaWizardBuilderOptionsAndOrder(t *testing.T) {
	L := lua.NewState()
	defer L.Close()

	SetupMetatable(L)
	fns := make(map[string]lua.LGFunction)
	RegisterLuaFunctions(fns)
	for name, fn := range fns {
		L.SetGlobal(name, L.NewFunction(fn))
	}

	if err := L.DoString(`
wiz = wizard("lua_wiz", "Lua Wizard")
wiz:input("host", "Host", { required = true, desc = "host desc" })
wiz:select("protocol", "Protocol", {"tcp", "http", "https"}, "http", { required = true })
wiz:multiselect("mods", "Modules", {"a", "b"}, {"a"}, { required = true })
`); err != nil {
		t.Fatalf("lua script failed: %v", err)
	}

	ud, ok := L.GetGlobal("wiz").(*lua.LUserData)
	if !ok {
		t.Fatalf("expected wiz userdata, got %T", L.GetGlobal("wiz"))
	}
	wiz, ok := ud.Value.(*Wizard)
	if !ok {
		t.Fatalf("expected *Wizard userdata value, got %T", ud.Value)
	}

	if len(wiz.Fields) != 3 {
		t.Fatalf("expected 3 fields, got %d", len(wiz.Fields))
	}
	if wiz.Fields[0].Description != "host desc" || !wiz.Fields[0].Required {
		t.Fatalf("expected input opts applied, got desc=%q required=%v", wiz.Fields[0].Description, wiz.Fields[0].Required)
	}
	if got := wiz.Fields[1].Options; len(got) != 3 || got[0] != "tcp" || got[1] != "http" || got[2] != "https" {
		t.Fatalf("expected select option order preserved, got %#v", got)
	}
	if wiz.Fields[2].Default == nil {
		t.Fatalf("expected multiselect defaults set")
	}
}

func TestLuaWizardFromSpecAndRegisterTemplate(t *testing.T) {
	L := lua.NewState()
	defer L.Close()

	SetupMetatable(L)
	fns := make(map[string]lua.LGFunction)
	RegisterLuaFunctions(fns)
	for name, fn := range fns {
		L.SetGlobal(name, L.NewFunction(fn))
	}

	name := "lua_spec_template_test"
	t.Cleanup(func() {
		templatesMu.Lock()
		delete(Templates, name)
		templatesMu.Unlock()
	})

	if err := L.DoString(`
wiz2 = wizard_from_spec({
  id = "from_spec",
  title = "From Spec",
  fields = {
    { name = "port", title = "Port", type = "number", default = 80, required = true },
  },
})
wizard_register_template("` + name + `", {
  title = "Registered Template",
  fields = {
    { name = "host", title = "Host", type = "input", default = "127.0.0.1" },
  },
})
`); err != nil {
		t.Fatalf("lua script failed: %v", err)
	}

	ud, ok := L.GetGlobal("wiz2").(*lua.LUserData)
	if !ok {
		t.Fatalf("expected wiz2 userdata, got %T", L.GetGlobal("wiz2"))
	}
	wiz, ok := ud.Value.(*Wizard)
	if !ok {
		t.Fatalf("expected *Wizard userdata value, got %T", ud.Value)
	}
	if wiz.ID != "from_spec" || len(wiz.Fields) != 1 {
		t.Fatalf("unexpected wizard from spec: id=%q fields=%d", wiz.ID, len(wiz.Fields))
	}

	templ, ok := GetTemplate(name)
	if !ok || templ == nil {
		t.Fatalf("expected registered template")
	}
	if templ.ID != name {
		t.Fatalf("expected template ID %q, got %q", name, templ.ID)
	}
}

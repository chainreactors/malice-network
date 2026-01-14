package wizard

import (
	"errors"
	"strings"
	"sync"

	"github.com/chainreactors/malice-network/helper/intermediate"
	"github.com/chainreactors/mals"
	lua "github.com/yuin/gopher-lua"
)

const luaWizardTypeName = "wizard"

// RegisterLuaFunctions registers wizard functions to the VM functions map
func RegisterLuaFunctions(vmFns map[string]lua.LGFunction) {
	vmFns["wizard"] = luaWizardNew
	vmFns["wizard_template"] = luaWizardTemplate
	vmFns["wizard_templates"] = luaWizardListTemplates
	vmFns["wizard_from_spec"] = luaWizardFromSpec
	vmFns["wizard_from_file"] = luaWizardFromFile
	vmFns["wizard_register_template"] = luaWizardRegisterTemplate
}

// SetupMetatable sets up the wizard metatable in a Lua VM
func SetupMetatable(L *lua.LState) {
	mt := L.NewTypeMetatable(luaWizardTypeName)
	L.SetField(mt, "__index", L.SetFuncs(L.NewTable(), wizardMethods))
}

// wizardMethods contains all methods available on wizard userdata
var wizardMethods = map[string]lua.LGFunction{
	"input":       wizardInput,
	"text":        wizardText,
	"select":      wizardSelect,
	"multiselect": wizardMultiSelect,
	"confirm":     wizardConfirm,
	"number":      wizardNumber,
	"filepath":    wizardFilePath,
	"description": wizardDescription,
	"clone":       wizardClone,
	"run":         wizardRun,
	"get_field":   wizardGetField,
	"field_count": wizardFieldCount,
}

// luaWizardNew creates a new wizard (Lua: wizard(id, title))
func luaWizardNew(L *lua.LState) int {
	id := L.CheckString(1)
	title := L.OptString(2, "")

	wiz := NewWizard(id, title)

	ud := L.NewUserData()
	ud.Value = wiz
	L.SetMetatable(ud, L.GetTypeMetatable(luaWizardTypeName))
	L.Push(ud)
	return 1
}

// luaWizardTemplate loads a predefined template (Lua: wizard_template(name))
func luaWizardTemplate(L *lua.LState) int {
	name := L.CheckString(1)

	wiz, ok := GetTemplate(name)
	if !ok {
		L.Push(lua.LNil)
		L.Push(lua.LString("template not found: " + name))
		return 2
	}

	ud := L.NewUserData()
	ud.Value = wiz
	L.SetMetatable(ud, L.GetTypeMetatable(luaWizardTypeName))
	L.Push(ud)
	return 1
}

// luaWizardListTemplates returns available template names (Lua: wizard_templates())
func luaWizardListTemplates(L *lua.LState) int {
	names := ListTemplates()
	tbl := L.NewTable()
	for i, name := range names {
		tbl.RawSetInt(i+1, lua.LString(name))
	}
	L.Push(tbl)
	return 1
}

// luaWizardFromSpec builds a wizard from a spec table (Lua: wizard_from_spec(spec))
func luaWizardFromSpec(L *lua.LState) int {
	raw := mals.ConvertLuaValueToGo(L.CheckTable(1))
	specMap, ok := raw.(map[string]interface{})
	if !ok {
		L.Push(lua.LNil)
		L.Push(lua.LString("spec must be a table"))
		return 2
	}
	spec, err := SpecFromMap(specMap)
	if err != nil {
		L.Push(lua.LNil)
		L.Push(lua.LString(err.Error()))
		return 2
	}
	wiz, err := NewWizardFromSpec(spec)
	if err != nil {
		L.Push(lua.LNil)
		L.Push(lua.LString(err.Error()))
		return 2
	}

	ud := L.NewUserData()
	ud.Value = wiz
	L.SetMetatable(ud, L.GetTypeMetatable(luaWizardTypeName))
	L.Push(ud)
	return 1
}

// luaWizardFromFile loads a wizard spec from file (Lua: wizard_from_file(path))
func luaWizardFromFile(L *lua.LState) int {
	path := L.CheckString(1)
	wiz, err := NewWizardFromFile(path)
	if err != nil {
		L.Push(lua.LNil)
		L.Push(lua.LString(err.Error()))
		return 2
	}

	ud := L.NewUserData()
	ud.Value = wiz
	L.SetMetatable(ud, L.GetTypeMetatable(luaWizardTypeName))
	L.Push(ud)
	return 1
}

// luaWizardRegisterTemplate registers a template from a wizard or a spec table.
// Lua:
//
//	wizard_register_template(name, wiz)
//	wizard_register_template(name, specTable)
func luaWizardRegisterTemplate(L *lua.LState) int {
	name := L.CheckString(1)
	if name == "" {
		L.Push(lua.LNil)
		L.Push(lua.LString("template name is required"))
		return 2
	}

	switch v := L.Get(2).(type) {
	case *lua.LUserData:
		wiz, ok := v.Value.(*Wizard)
		if !ok {
			L.Push(lua.LNil)
			L.Push(lua.LString("second arg must be wizard userdata or spec table"))
			return 2
		}
		base := wiz.Clone()
		RegisterTemplate(name, func() *Wizard { return base.Clone() })
		L.Push(lua.LTrue)
		return 1
	case *lua.LTable:
		raw := mals.ConvertLuaValueToGo(v)
		specMap, ok := raw.(map[string]interface{})
		if !ok {
			L.Push(lua.LNil)
			L.Push(lua.LString("spec must be a table"))
			return 2
		}
		spec, err := SpecFromMap(specMap)
		if err != nil {
			L.Push(lua.LNil)
			L.Push(lua.LString(err.Error()))
			return 2
		}
		if spec.ID == "" {
			spec.ID = name
		}
		if err := RegisterTemplateFromSpec(name, spec); err != nil {
			L.Push(lua.LNil)
			L.Push(lua.LString(err.Error()))
			return 2
		}
		L.Push(lua.LTrue)
		return 1
	default:
		L.Push(lua.LNil)
		L.Push(lua.LString("second arg must be wizard userdata or spec table"))
		return 2
	}
}

// checkWizard extracts wizard from userdata
func checkWizard(L *lua.LState, n int) *Wizard {
	ud := L.CheckUserData(n)
	if v, ok := ud.Value.(*Wizard); ok {
		return v
	}
	L.ArgError(n, "wizard expected")
	return nil
}

type luaFieldOptions struct {
	Description string
	Required    bool
}

// luaFieldArgs holds parsed arguments for field creation methods
type luaFieldArgs struct {
	Name    string
	Title   string
	Default lua.LValue
	Options luaFieldOptions
	OptsIdx int
}

// parseLuaFieldArgs extracts name, title, optional default value, and options table
// from Lua stack starting at position 2 (position 1 is self).
// defaultIdx is the position where the default value is expected (usually 4).
func parseLuaFieldArgs(L *lua.LState, defaultIdx int) luaFieldArgs {
	args := luaFieldArgs{
		Name:  L.CheckString(2),
		Title: L.CheckString(3),
	}

	if L.GetTop() >= defaultIdx {
		v := L.Get(defaultIdx)
		if v.Type() == lua.LTTable {
			args.OptsIdx = defaultIdx
		} else {
			args.Default = v
			if L.GetTop() >= defaultIdx+1 && L.Get(defaultIdx+1).Type() == lua.LTTable {
				args.OptsIdx = defaultIdx + 1
			}
		}
	}

	args.Options = parseLuaFieldOptions(L, args.OptsIdx)
	return args
}

func parseLuaFieldOptions(L *lua.LState, idx int) luaFieldOptions {
	if idx <= 0 || L.Get(idx).Type() != lua.LTTable {
		return luaFieldOptions{}
	}
	tbl := L.CheckTable(idx)

	desc := lua.LVAsString(L.GetField(tbl, "description"))
	if desc == "" {
		desc = lua.LVAsString(L.GetField(tbl, "desc"))
	}
	required := false
	if v := L.GetField(tbl, "required"); v != lua.LNil {
		required = lua.LVAsBool(v)
	}

	return luaFieldOptions{
		Description: desc,
		Required:    required,
	}
}

func luaStringSlice(L *lua.LState, tbl *lua.LTable) ([]string, error) {
	out := make([]string, 0, tbl.Len())
	for i := 1; i <= tbl.Len(); i++ {
		v := tbl.RawGetInt(i)
		if v == lua.LNil {
			continue
		}
		out = append(out, lua.LVAsString(v))
	}
	return out, nil
}

// wizardInput adds an input field (Lua: wiz:input(name, title, default))
func wizardInput(L *lua.LState) int {
	wiz := checkWizard(L, 1)
	args := parseLuaFieldArgs(L, 4)

	defaultVal := ""
	if args.Default != nil && args.Default != lua.LNil {
		s, ok := args.Default.(lua.LString)
		if !ok {
			L.TypeError(4, lua.LTString)
			return 0
		}
		defaultVal = string(s)
	}

	wiz.AddField(&WizardField{
		Name:        args.Name,
		Title:       args.Title,
		Description: args.Options.Description,
		Type:        FieldInput,
		Default:     defaultVal,
		Required:    args.Options.Required,
	})

	L.Push(L.Get(1))
	return 1
}

// wizardText adds a text field (Lua: wiz:text(name, title, default))
func wizardText(L *lua.LState) int {
	wiz := checkWizard(L, 1)
	args := parseLuaFieldArgs(L, 4)

	defaultVal := ""
	if args.Default != nil && args.Default != lua.LNil {
		s, ok := args.Default.(lua.LString)
		if !ok {
			L.TypeError(4, lua.LTString)
			return 0
		}
		defaultVal = string(s)
	}

	wiz.AddField(&WizardField{
		Name:        args.Name,
		Title:       args.Title,
		Description: args.Options.Description,
		Type:        FieldText,
		Default:     defaultVal,
		Required:    args.Options.Required,
	})

	L.Push(L.Get(1))
	return 1
}

// wizardSelect adds a select field (Lua: wiz:select(name, title, options))
func wizardSelect(L *lua.LState) int {
	wiz := checkWizard(L, 1)
	name := L.CheckString(2)
	title := L.CheckString(3)

	optionsTbl := L.CheckTable(4)
	options, err := luaStringSlice(L, optionsTbl)
	if err != nil {
		L.Push(lua.LNil)
		L.Push(lua.LString(err.Error()))
		return 2
	}

	defaultVal := ""
	optsIdx := 0
	if L.GetTop() >= 5 {
		if L.Get(5).Type() == lua.LTTable {
			optsIdx = 5
		} else {
			defaultVal = L.OptString(5, "")
			if L.GetTop() >= 6 && L.Get(6).Type() == lua.LTTable {
				optsIdx = 6
			}
		}
	}
	opts := parseLuaFieldOptions(L, optsIdx)
	wiz.AddField(&WizardField{
		Name:        name,
		Title:       title,
		Description: opts.Description,
		Type:        FieldSelect,
		Options:     options,
		Default:     defaultVal,
		Required:    opts.Required,
	})

	L.Push(L.Get(1))
	return 1
}

// wizardMultiSelect adds a multi-select field (Lua: wiz:multiselect(name, title, options))
func wizardMultiSelect(L *lua.LState) int {
	wiz := checkWizard(L, 1)
	name := L.CheckString(2)
	title := L.CheckString(3)

	optionsTbl := L.CheckTable(4)
	options, err := luaStringSlice(L, optionsTbl)
	if err != nil {
		L.Push(lua.LNil)
		L.Push(lua.LString(err.Error()))
		return 2
	}

	var defaults []string
	optsIdx := 0
	if L.GetTop() >= 5 && L.Get(5).Type() == lua.LTTable {
		t := L.CheckTable(5)
		if L.GetField(t, "required") != lua.LNil || L.GetField(t, "description") != lua.LNil || L.GetField(t, "desc") != lua.LNil {
			optsIdx = 5
		} else {
			defaults, err = luaStringSlice(L, t)
			if err != nil {
				L.Push(lua.LNil)
				L.Push(lua.LString(err.Error()))
				return 2
			}
			if L.GetTop() >= 6 && L.Get(6).Type() == lua.LTTable {
				optsIdx = 6
			}
		}
	}
	opts := parseLuaFieldOptions(L, optsIdx)

	field := &WizardField{
		Name:        name,
		Title:       title,
		Description: opts.Description,
		Type:        FieldMultiSelect,
		Options:     options,
		Required:    opts.Required,
	}
	if defaults != nil {
		field.Default = defaults
	}
	wiz.AddField(field)

	L.Push(L.Get(1))
	return 1
}

// wizardConfirm adds a confirm field (Lua: wiz:confirm(name, title, default))
func wizardConfirm(L *lua.LState) int {
	wiz := checkWizard(L, 1)
	args := parseLuaFieldArgs(L, 4)

	defaultVal := false
	if args.Default != nil && args.Default != lua.LNil {
		b, ok := args.Default.(lua.LBool)
		if !ok {
			L.TypeError(4, lua.LTBool)
			return 0
		}
		defaultVal = bool(b)
	}

	wiz.AddField(&WizardField{
		Name:        args.Name,
		Title:       args.Title,
		Description: args.Options.Description,
		Type:        FieldConfirm,
		Default:     defaultVal,
		Required:    args.Options.Required,
	})

	L.Push(L.Get(1))
	return 1
}

// wizardNumber adds a number field (Lua: wiz:number(name, title, default))
func wizardNumber(L *lua.LState) int {
	wiz := checkWizard(L, 1)
	args := parseLuaFieldArgs(L, 4)

	defaultVal := 0
	if args.Default != nil && args.Default != lua.LNil {
		n, ok := args.Default.(lua.LNumber)
		if !ok {
			L.TypeError(4, lua.LTNumber)
			return 0
		}
		defaultVal = int(n)
	}

	wiz.AddField(&WizardField{
		Name:        args.Name,
		Title:       args.Title,
		Description: args.Options.Description,
		Type:        FieldNumber,
		Default:     defaultVal,
		Required:    args.Options.Required,
	})

	L.Push(L.Get(1))
	return 1
}

// wizardFilePath adds a file path field (Lua: wiz:filepath(name, title))
func wizardFilePath(L *lua.LState) int {
	wiz := checkWizard(L, 1)
	args := parseLuaFieldArgs(L, 4)

	defaultVal := ""
	if args.Default != nil && args.Default != lua.LNil {
		s, ok := args.Default.(lua.LString)
		if !ok {
			L.TypeError(4, lua.LTString)
			return 0
		}
		defaultVal = string(s)
	}

	wiz.AddField(&WizardField{
		Name:        args.Name,
		Title:       args.Title,
		Description: args.Options.Description,
		Type:        FieldFilePath,
		Default:     defaultVal,
		Required:    args.Options.Required,
	})

	L.Push(L.Get(1))
	return 1
}

// wizardDescription sets the wizard description (Lua: wiz:description(desc))
func wizardDescription(L *lua.LState) int {
	wiz := checkWizard(L, 1)
	desc := L.CheckString(2)

	wiz.WithDescription(desc)

	L.Push(L.Get(1))
	return 1
}

// wizardClone creates a copy of the wizard (Lua: wiz:clone())
func wizardClone(L *lua.LState) int {
	wiz := checkWizard(L, 1)
	clone := wiz.Clone()

	ud := L.NewUserData()
	ud.Value = clone
	L.SetMetatable(ud, L.GetTypeMetatable(luaWizardTypeName))
	L.Push(ud)
	return 1
}

// wizardRun executes the wizard (Lua: wiz:run())
func wizardRun(L *lua.LState) int {
	wiz := checkWizard(L, 1)

	runner := NewRunner(wiz)
	result, err := runner.Run()

	if err != nil {
		L.Push(lua.LNil)
		L.Push(lua.LString(err.Error()))
		return 2
	}

	// Convert result to Lua table
	resultTable := L.NewTable()
	for k, v := range result.ToMap() {
		resultTable.RawSetString(k, convertToLuaValue(L, v))
	}

	L.Push(resultTable)
	return 1
}

// wizardGetField gets field info (Lua: wiz:get_field(name))
func wizardGetField(L *lua.LState) int {
	wiz := checkWizard(L, 1)
	name := L.CheckString(2)

	for _, f := range wiz.Fields {
		if f.Name == name {
			tbl := L.NewTable()
			tbl.RawSetString("name", lua.LString(f.Name))
			tbl.RawSetString("title", lua.LString(f.Title))
			tbl.RawSetString("description", lua.LString(f.Description))
			tbl.RawSetString("type", lua.LNumber(float64(f.Type)))
			tbl.RawSetString("required", lua.LBool(f.Required))
			if f.Default != nil {
				tbl.RawSetString("default", convertToLuaValue(L, f.Default))
			}
			if len(f.Options) > 0 {
				optTbl := L.NewTable()
				for i, opt := range f.Options {
					optTbl.RawSetInt(i+1, lua.LString(opt))
				}
				tbl.RawSetString("options", optTbl)
			}
			L.Push(tbl)
			return 1
		}
	}

	L.Push(lua.LNil)
	return 1
}

// wizardFieldCount returns the number of fields (Lua: wiz:field_count())
func wizardFieldCount(L *lua.LState) int {
	wiz := checkWizard(L, 1)
	L.Push(lua.LNumber(len(wiz.Fields)))
	return 1
}

// convertToLuaValue converts a Go value to Lua value
func convertToLuaValue(L *lua.LState, v interface{}) lua.LValue {
	switch val := v.(type) {
	case string:
		return lua.LString(val)
	case bool:
		return lua.LBool(val)
	case int:
		return lua.LNumber(float64(val))
	case float64:
		return lua.LNumber(val)
	case []string:
		tbl := L.NewTable()
		for i, s := range val {
			tbl.RawSetInt(i+1, lua.LString(s))
		}
		return tbl
	default:
		return lua.LNil
	}
}

var registerBuiltinsOnce sync.Once

// RegisterBuiltinFunctions registers wizard functions as builtin functions
// This allows the functions to be available in all Lua VMs
func RegisterBuiltinFunctions() {
	registerBuiltinsOnce.Do(func() {
		// Register wizard constructor
		intermediate.RegisterFunction("wizard", func(id string, title string) (*Wizard, error) {
			return NewWizard(id, title), nil
		})

		intermediate.AddHelper("wizard", &mals.Helper{
			Group:   intermediate.ClientGroup,
			Short:   "Create a new interactive wizard form",
			Input:   []string{"id: wizard identifier", "title: wizard title"},
			Output:  []string{"wizard"},
			Example: `local wiz = wizard("my_wizard", "My Wizard")`,
		})

		// Register template loader
		intermediate.RegisterFunction("wizard_template", func(name string) (*Wizard, error) {
			wiz, ok := GetTemplate(name)
			if !ok {
				return nil, nil
			}
			return wiz, nil
		})

		intermediate.AddHelper("wizard_template", &mals.Helper{
			Group:   intermediate.ClientGroup,
			Short:   "Load a predefined wizard template",
			Input:   []string{"name: template name"},
			Output:  []string{"wizard"},
			Example: `local wiz = wizard_template("listener_setup")`,
		})

		// Register template list
		intermediate.RegisterFunction("wizard_templates", func() ([]string, error) {
			return ListTemplates(), nil
		})

		intermediate.AddHelper("wizard_templates", &mals.Helper{
			Group:   intermediate.ClientGroup,
			Short:   "List available wizard templates",
			Output:  []string{"template names"},
			Example: `local templates = wizard_templates()`,
		})

		// Config-driven helpers
		intermediate.RegisterFunction("wizard_from_file", func(path string) (*Wizard, error) {
			return NewWizardFromFile(path)
		})

		intermediate.AddHelper("wizard_from_file", &mals.Helper{
			Group:  intermediate.ClientGroup,
			Short:  "Load a wizard from a JSON/YAML spec file",
			Input:  []string{"path: .json/.yaml/.yml wizard spec file"},
			Output: []string{"wizard"},
			Example: `
-- Prefer loading from plugin resources so embedded/external plugins both work:
wiz = wizard_from_file(script_resource("wizards/priv_esc.yaml"))
`,
		})

		intermediate.RegisterFunction("wizard_from_spec", func(spec map[string]interface{}) (*Wizard, error) {
			ws, err := SpecFromMap(spec)
			if err != nil {
				return nil, err
			}
			return NewWizardFromSpec(ws)
		})

		intermediate.AddHelper("wizard_from_spec", &mals.Helper{
			Group:  intermediate.ClientGroup,
			Short:  "Build a wizard from a Lua spec table",
			Input:  []string{"spec: table"},
			Output: []string{"wizard"},
			Example: `
wiz = wizard_from_spec({
  id = "my_wizard",
  title = "My Wizard",
  fields = {
    { name = "host", title = "Host", type = "input", default = "0.0.0.0", required = true },
  },
})
`,
		})

		intermediate.RegisterFunction("wizard_register_template", func(name string, spec map[string]interface{}) (bool, error) {
			if strings.TrimSpace(name) == "" {
				return false, errors.New("template name is required")
			}
			ws, err := SpecFromMap(spec)
			if err != nil {
				return false, err
			}
			if ws.ID == "" {
				ws.ID = name
			}
			if err := RegisterTemplateFromSpec(name, ws); err != nil {
				return false, err
			}
			return true, nil
		})

		intermediate.AddHelper("wizard_register_template", &mals.Helper{
			Group:  intermediate.ClientGroup,
			Short:  "Register a wizard template from a spec table",
			Input:  []string{"name: template name", "spec: table"},
			Output: []string{"ok: boolean"},
			Example: `
wizard_register_template("priv_esc", {
  title = "Privilege Escalation",
  fields = {
    { name = "method", title = "Method", type = "select", options = {"uac","token"} },
  },
})
`,
		})
	})
}

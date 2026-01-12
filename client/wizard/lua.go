package wizard

import (
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
}

// SetupMetatable sets up the wizard metatable in a Lua VM
func SetupMetatable(L *lua.LState) {
	mt := L.NewTypeMetatable(luaWizardTypeName)
	L.SetField(mt, "__index", L.SetFuncs(L.NewTable(), wizardMethods))
}

// wizardMethods contains all methods available on wizard userdata
var wizardMethods = map[string]lua.LGFunction{
	"input":        wizardInput,
	"text":         wizardText,
	"select":       wizardSelect,
	"multiselect":  wizardMultiSelect,
	"confirm":      wizardConfirm,
	"number":       wizardNumber,
	"filepath":     wizardFilePath,
	"description":  wizardDescription,
	"run":          wizardRun,
	"get_field":    wizardGetField,
	"field_count":  wizardFieldCount,
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

// checkWizard extracts wizard from userdata
func checkWizard(L *lua.LState, n int) *Wizard {
	ud := L.CheckUserData(n)
	if v, ok := ud.Value.(*Wizard); ok {
		return v
	}
	L.ArgError(n, "wizard expected")
	return nil
}

// wizardInput adds an input field (Lua: wiz:input(name, title, default))
func wizardInput(L *lua.LState) int {
	wiz := checkWizard(L, 1)
	name := L.CheckString(2)
	title := L.CheckString(3)
	defaultVal := L.OptString(4, "")

	wiz.Input(name, title, defaultVal)

	L.Push(L.Get(1)) // Return self for chaining
	return 1
}

// wizardText adds a text field (Lua: wiz:text(name, title, default))
func wizardText(L *lua.LState) int {
	wiz := checkWizard(L, 1)
	name := L.CheckString(2)
	title := L.CheckString(3)
	defaultVal := L.OptString(4, "")

	wiz.Text(name, title, defaultVal)

	L.Push(L.Get(1))
	return 1
}

// wizardSelect adds a select field (Lua: wiz:select(name, title, options))
func wizardSelect(L *lua.LState) int {
	wiz := checkWizard(L, 1)
	name := L.CheckString(2)
	title := L.CheckString(3)

	options := make([]string, 0)
	tbl := L.CheckTable(4)
	tbl.ForEach(func(_, v lua.LValue) {
		options = append(options, v.String())
	})

	wiz.Select(name, title, options)

	L.Push(L.Get(1))
	return 1
}

// wizardMultiSelect adds a multi-select field (Lua: wiz:multiselect(name, title, options))
func wizardMultiSelect(L *lua.LState) int {
	wiz := checkWizard(L, 1)
	name := L.CheckString(2)
	title := L.CheckString(3)

	options := make([]string, 0)
	tbl := L.CheckTable(4)
	tbl.ForEach(func(_, v lua.LValue) {
		options = append(options, v.String())
	})

	wiz.MultiSelect(name, title, options)

	L.Push(L.Get(1))
	return 1
}

// wizardConfirm adds a confirm field (Lua: wiz:confirm(name, title, default))
func wizardConfirm(L *lua.LState) int {
	wiz := checkWizard(L, 1)
	name := L.CheckString(2)
	title := L.CheckString(3)
	defaultVal := L.OptBool(4, false)

	wiz.Confirm(name, title, defaultVal)

	L.Push(L.Get(1))
	return 1
}

// wizardNumber adds a number field (Lua: wiz:number(name, title, default))
func wizardNumber(L *lua.LState) int {
	wiz := checkWizard(L, 1)
	name := L.CheckString(2)
	title := L.CheckString(3)
	defaultVal := L.OptInt(4, 0)

	wiz.Number(name, title, defaultVal)

	L.Push(L.Get(1))
	return 1
}

// wizardFilePath adds a file path field (Lua: wiz:filepath(name, title))
func wizardFilePath(L *lua.LState) int {
	wiz := checkWizard(L, 1)
	name := L.CheckString(2)
	title := L.CheckString(3)

	wiz.FilePath(name, title)

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

// RegisterBuiltinFunctions registers wizard functions as builtin functions
// This allows the functions to be available in all Lua VMs
func RegisterBuiltinFunctions() {
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
}

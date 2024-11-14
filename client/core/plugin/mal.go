package plugin

var (
	MalType = "lua"
)

type MalManiFest struct {
	Name         string   `json:"name" yaml:"name"`
	Type         string   `json:"type" yaml:"type"` // lua, tcl
	Author       string   `json:"author" yaml:"author"`
	Version      string   `json:"version" yaml:"version"`
	EntryFile    string   `json:"entry" yaml:"entry"`
	Lib          bool     `json:"lib" yaml:"lib"`
	DependModule []string `json:"depend_module" yaml:"depend_modules"`
	DependArmory []string `json:"depend_armory" yaml:"depend_armory"`
}

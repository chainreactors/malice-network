package assets

var (
	maliceProfile = "malice.yaml"
)

type Profile struct {
	Aliases    []string `yaml:"aliases"`
	Extensions []string `yaml:"extensions"`
	Modules    []string `yaml:"modules"`
}

package assets

var (
	maliceProfile = "malice.yaml"
)

type Profile struct {
	ResourceDir string   `yaml:"resources" default:""`
	TempDir     string   `yaml:"tmp" default:""`
	Aliases     []string `yaml:"aliases"`
	Extensions  []string `yaml:"extensions"`
	Mals        []string `yaml:"mals"`
	//Modules     []string  `yaml:"modules"`
	Settings *Settings `yaml:"settings"`
}

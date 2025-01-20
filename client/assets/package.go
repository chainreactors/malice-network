package assets

import (
	"encoding/json"
	"github.com/chainreactors/mals/m"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path/filepath"
)

const (
	armoryConfigFileName = "armories.json"
	DefaultArmoryName    = "Default"
)

var (
	// DefaultArmoryPublicKey - The default public key for the armory
	DefaultArmoryPublicKey = "RWSBpxpRWDrD7Fe+VvRE3c2VEDC2NK80rlNCj+BX0gz44Xw07r6KQD9L"
	// DefaultArmoryRepoURL - The default repo url for the armory
	DefaultArmoryRepoURL = "https://api.github.com/repos/sliverarmory/armory/releases"

	DefaultArmoryConfig = &ArmoryConfig{
		PublicKey: DefaultArmoryPublicKey,
		RepoURL:   DefaultArmoryRepoURL,
		Name:      DefaultArmoryName,
		Enabled:   true,
	}
)

func GetMalsConfig() []*m.MalConfig {
	malConfigPath := filepath.Join(GetRootAppDir(), m.ManifestFileName)
	if _, err := os.Stat(malConfigPath); os.IsNotExist(err) {
		return nil
	}
	data, err := os.ReadFile(malConfigPath)
	if err != nil {
		return nil
	}
	var malConfigs []*m.MalConfig
	err = json.Unmarshal(data, &malConfigs)
	if err != nil {
		return nil
	}
	for _, malConfig := range malConfigs {
		if malConfig.AuthorizationCmd != "" {
			malConfig.Authorization = ExecuteAuthorizationCmd(malConfig.AuthorizationCmd)
		}
	}
	return append(malConfigs, m.DefaultMalConfig)
}

// ArmoryConfig - The armory config file
type ArmoryConfig struct {
	PublicKey        string `json:"public_key"`
	RepoURL          string `json:"repo_url"`
	Authorization    string `json:"authorization"`
	AuthorizationCmd string `json:"authorization_cmd"`
	Name             string `json:"name"`
	Enabled          bool   `json:"enabled"`
}

// GetArmoriesConfig - The parsed armory config file
func GetArmoriesConfig() []*ArmoryConfig {
	armoryConfigPath := filepath.Join(GetRootAppDir(), armoryConfigFileName)
	if _, err := os.Stat(armoryConfigPath); os.IsNotExist(err) {
		return []*ArmoryConfig{DefaultArmoryConfig}
	}
	data, err := ioutil.ReadFile(armoryConfigPath)
	if err != nil {
		return []*ArmoryConfig{DefaultArmoryConfig}
	}
	var armoryConfigs []*ArmoryConfig
	err = json.Unmarshal(data, &armoryConfigs)
	if err != nil {
		return []*ArmoryConfig{DefaultArmoryConfig}
	}
	for _, armoryConfig := range armoryConfigs {
		if armoryConfig.AuthorizationCmd != "" {
			armoryConfig.Authorization = ExecuteAuthorizationCmd(armoryConfig.AuthorizationCmd)
		}
	}
	return append(armoryConfigs, DefaultArmoryConfig)
}

func ExecuteAuthorizationCmd(cmd string) string {
	if cmd == "" {
		return ""
	}
	out, err := exec.Command(cmd).CombinedOutput()
	if err != nil {
		log.Printf("Failed to execute authorization_cmd '%s': %v", cmd, err)
		return ""
	}
	return string(out)
}

package assets

import (
	"encoding/json"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path/filepath"
)

const (
	armoryConfigFileName = "armories.json"
)

var (
	// DefaultArmoryPublicKey - The default public key for the armory
	DefaultArmoryPublicKey string
	// DefaultArmoryRepoURL - The default repo url for the armory
	DefaultArmoryRepoURL string

	defaultArmoryConfig = &ArmoryConfig{
		PublicKey: DefaultArmoryPublicKey,
		RepoURL:   DefaultArmoryRepoURL,
	}
)

// ArmoryConfig - The armory config file
type ArmoryConfig struct {
	PublicKey        string `json:"public_key"`
	RepoURL          string `json:"repo_url"`
	Authorization    string `json:"authorization"`
	AuthorizationCmd string `json:"authorization_cmd"`
}

// GetArmoriesConfig - The parsed armory config file
func GetArmoriesConfig() []*ArmoryConfig {
	armoryConfigPath := filepath.Join(GetRootAppDir(), armoryConfigFileName)
	if _, err := os.Stat(armoryConfigPath); os.IsNotExist(err) {
		return []*ArmoryConfig{defaultArmoryConfig}
	}
	data, err := ioutil.ReadFile(armoryConfigPath)
	if err != nil {
		return []*ArmoryConfig{defaultArmoryConfig}
	}
	var armoryConfigs []*ArmoryConfig
	err = json.Unmarshal(data, &armoryConfigs)
	if err != nil {
		return []*ArmoryConfig{defaultArmoryConfig}
	}
	for _, armoryConfig := range armoryConfigs {
		if armoryConfig.AuthorizationCmd != "" {
			armoryConfig.Authorization = executeAuthorizationCmd(armoryConfig)
		}
	}
	return append(armoryConfigs, defaultArmoryConfig)
}

func executeAuthorizationCmd(armoryConfig *ArmoryConfig) string {
	if armoryConfig.AuthorizationCmd == "" {
		return ""
	}
	out, err := exec.Command(armoryConfig.AuthorizationCmd).CombinedOutput()
	if err != nil {
		log.Printf("Failed to execute authorization_cmd '%s': %v", armoryConfig.AuthorizationCmd, err)
		return ""
	}
	return string(out)
}

package armory

import (
	"github.com/chainreactors/malice-network/client/assets"
	"slices"
)

func getCurrentArmoryConfiguration() []*assets.ArmoryConfig {
	configs := []*assets.ArmoryConfig{}
	armoryNames := []string{}
	// If the default armory is in the configuration, force it to be last
	var defaultConfig *assets.ArmoryConfig

	currentArmories.Range(func(key, value interface{}) bool {
		armoryEntry := value.(assets.ArmoryConfig)
		// Skip over the default armory for now
		if armoryEntry.Name != assets.DefaultArmoryName {
			configs = append(configs, &armoryEntry)
			armoryNames = append(armoryNames, armoryEntry.Name)
		} else {
			defaultConfig = &armoryEntry
		}
		return true
	})

	if !armoriesInitialized {
		/*
			Armories are initialized on the first call to the armory command
			If armories are added or removed before the first call, we want
			to make sure we still load in the ones from the configuration
			file.
		*/
		persistentConfigs := assets.GetArmoriesConfig()
		for _, config := range persistentConfigs {
			if !slices.Contains(armoryNames, config.Name) {
				if config.Name == assets.DefaultArmoryName {
					if defaultArmoryRemoved {
						continue
					} else if defaultConfig != nil {
						// The user potentially changed something about the default config
						configs = append(configs, defaultConfig)
						currentArmories.Store(config.PublicKey, *defaultConfig)
						continue
					}
				}
				configs = append(configs, config)
				currentArmories.Store(config.PublicKey, *config)
			}
		}
		return configs
	}
	if !defaultArmoryRemoved {
		if defaultConfig != nil {
			configs = append(configs, defaultConfig)
		} else {
			configs = append(configs, assets.DefaultArmoryConfig)
		}
	}

	for _, armoryConfig := range configs {
		if armoryConfig.AuthorizationCmd != "" {
			armoryConfig.Authorization = assets.ExecuteAuthorizationCmd(armoryConfig.AuthorizationCmd)
		}
	}
	return configs
}

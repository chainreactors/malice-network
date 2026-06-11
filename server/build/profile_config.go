package build

import "github.com/chainreactors/IoM-go/proto/client/clientpb"

func needsProfileFiles(config *clientpb.BuildConfig) bool {
	if config == nil || config.ProfileName == "" {
		return false
	}
	return config.MaleficConfig == nil || config.PreludeConfig == nil || config.Resources == nil
}

func mergeProfileFiles(config *clientpb.BuildConfig, implant, prelude []byte, resources *clientpb.BuildResources) {
	if config == nil {
		return
	}
	if config.MaleficConfig == nil {
		config.MaleficConfig = implant
	}
	if config.PreludeConfig == nil {
		config.PreludeConfig = prelude
	}
	if config.Resources == nil {
		config.Resources = resources
	}
}

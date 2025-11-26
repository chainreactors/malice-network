package build

import (
	"context"
	"fmt"
	"os"

	"github.com/chainreactors/IoM-go/consts"
	"github.com/chainreactors/IoM-go/proto/client/clientpb"

	"github.com/chainreactors/malice-network/server/internal/configs"
	"github.com/docker/docker/client"
)

// CheckSource
func CheckSource(ctx context.Context, req *clientpb.BuildConfig) (*clientpb.BuildConfig, error) {
	switch req.Source {
	case consts.ArtifactFromGithubAction:
		return checkGithubActionSource(req)
	case consts.ArtifactFromDocker:
		return checkDockerSource(ctx, req)
	case consts.ArtifactFromSaas:
		return checkSaasSource(req)
	default:
		return AutoCheckSource(ctx, req)
	}
}

func AutoCheckSource(ctx context.Context, req *clientpb.BuildConfig) (*clientpb.BuildConfig, error) {
	buildConfig, err := checkDockerSource(ctx, req)
	if err == nil {
		buildConfig.Source = consts.ArtifactFromDocker
		return buildConfig, err
	}

	buildConfig, err = checkSaasSource(req)
	if err == nil {
		buildConfig.Source = consts.ArtifactFromSaas
		return buildConfig, err
	}

	buildConfig, err = checkGithubActionSource(req)
	if err == nil {
		buildConfig.Source = consts.ArtifactFromGithubAction
		return buildConfig, err
	}

	return nil, fmt.Errorf("no available source")
}

func checkGithubActionSource(req *clientpb.BuildConfig) (*clientpb.BuildConfig, error) {
	var actionConfig *clientpb.GithubActionBuildConfig

	if sourceConfig, ok := req.GetSourceConfig().(*clientpb.BuildConfig_GithubAction); ok && sourceConfig != nil && sourceConfig.GithubAction != nil {
		actionConfig = sourceConfig.GithubAction
	} else {
		config := configs.GetGithubConfig()
		if config == nil {
			return nil, fmt.Errorf("github action config not found in server settings")
		}

		actionConfig = config.ToProtobuf()
		req.SourceConfig = &clientpb.BuildConfig_GithubAction{
			GithubAction: actionConfig,
		}
	}

	// check config
	if err := validateGithubActionConfig(actionConfig); err != nil {
		return nil, fmt.Errorf("invalid github action config: %w", err)
	}

	// check workflow
	if err := GetWorkflowStatus(actionConfig); err != nil {
		return nil, fmt.Errorf("github workflow not available: %w", err)
	}

	req.Source = consts.ArtifactFromGithubAction
	return req, nil
}

func checkDockerSource(ctx context.Context, req *clientpb.BuildConfig) (*clientpb.BuildConfig, error) {
	cli, err := GetDockerClient()
	if err != nil {
		return nil, fmt.Errorf("docker client unavailable: %w", err)
	}

	if _, err := cli.Ping(ctx); err != nil {
		return nil, fmt.Errorf("docker daemon not responding: %w", err)
	}

	if err := ensureDirExists(configs.SourceCodePath); err != nil {
		return nil, fmt.Errorf("source code path unavailable: %w", err)
	}

	if req.Target != "" {
		image := GetImage(req.Target)
		if _, _, err := cli.ImageInspectWithRaw(ctx, image); err != nil {
			if client.IsErrNotFound(err) {
				return nil, fmt.Errorf("docker image %s for target %s not found", image, req.Target)
			}
			return nil, fmt.Errorf("failed to inspect docker image %s: %w", image, err)
		}
	}

	req.Source = consts.ArtifactFromDocker
	return req, nil
}

func ensureDirExists(path string) error {
	info, err := os.Stat(path)
	if err != nil {
		return err
	}

	if !info.IsDir() {
		return fmt.Errorf("%s is not a directory", path)
	}

	return nil
}

func checkSaasSource(req *clientpb.BuildConfig) (*clientpb.BuildConfig, error) {
	saasConfig := configs.GetSaasConfig()
	if saasConfig == nil {
		return nil, fmt.Errorf("saas config not found in server settings")
	}

	if !saasConfig.Enable {
		return nil, fmt.Errorf("saas build is disabled")
	}

	if saasConfig.Url == "" || saasConfig.Token == "" {
		return nil, fmt.Errorf("incomplete saas configuration")
	}

	req.Source = consts.ArtifactFromSaas
	return req, nil
}

func validateGithubActionConfig(config *clientpb.GithubActionBuildConfig) error {
	if config == nil {
		return fmt.Errorf("config is nil")
	}

	if config.Owner == "" {
		return fmt.Errorf("owner is required")
	}

	if config.Repo == "" {
		return fmt.Errorf("repository is required")
	}

	if config.Token == "" {
		return fmt.Errorf("token is required")
	}

	if config.WorkflowId == "" {
		return fmt.Errorf("workflow ID is required")
	}

	return nil
}

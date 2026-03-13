package build

import (
	"testing"

	"github.com/chainreactors/malice-network/client/assets"
	"github.com/chainreactors/malice-network/client/command/common"
	"github.com/gookit/config/v2"
	yamlDriver "github.com/gookit/config/v2/yaml"
	"github.com/spf13/cobra"
)

func TestResolveGithubActionConfigFallsBackToLocalSettings(t *testing.T) {
	initBuildConfigTest(t)

	if err := assets.SaveSettings(&assets.Settings{
		Github: &assets.GithubSetting{
			Owner:    "chainreactors",
			Repo:     "malice-network",
			Token:    "gh-token",
			Workflow: "generate.yml",
		},
	}); err != nil {
		t.Fatalf("SaveSettings failed: %v", err)
	}

	cmd := &cobra.Command{Use: "build"}
	common.GithubFlagSet(cmd.Flags())

	got := resolveGithubActionConfig(cmd)
	if got == nil {
		t.Fatal("expected github action config from local settings")
	}
	if got.Owner != "chainreactors" || got.Repo != "malice-network" || got.Token != "gh-token" || got.WorkflowId != "generate.yml" {
		t.Fatalf("unexpected github action config: %#v", got)
	}
}

func TestResolveGithubActionConfigPrefersExplicitFlags(t *testing.T) {
	initBuildConfigTest(t)

	if err := assets.SaveSettings(&assets.Settings{
		Github: &assets.GithubSetting{
			Owner:    "saved-owner",
			Repo:     "saved-repo",
			Token:    "saved-token",
			Workflow: "saved.yml",
		},
	}); err != nil {
		t.Fatalf("SaveSettings failed: %v", err)
	}

	cmd := &cobra.Command{Use: "build"}
	common.GithubFlagSet(cmd.Flags())
	if err := cmd.ParseFlags([]string{
		"--github-owner", "flag-owner",
		"--github-repo", "flag-repo",
		"--github-token", "flag-token",
		"--github-workflowFile", "flag.yml",
	}); err != nil {
		t.Fatalf("ParseFlags failed: %v", err)
	}

	got := resolveGithubActionConfig(cmd)
	if got == nil {
		t.Fatal("expected github action config from flags")
	}
	if got.Owner != "flag-owner" || got.Repo != "flag-repo" || got.Token != "flag-token" || got.WorkflowId != "flag.yml" {
		t.Fatalf("unexpected github action config: %#v", got)
	}
}

func initBuildConfigTest(t *testing.T) {
	t.Helper()

	config.Reset()
	config.WithOptions(func(opt *config.Options) {
		opt.DecoderConfig.TagName = "config"
		opt.ParseDefault = true
	}, config.WithHookFunc(assets.HookFn))
	config.AddDriver(yamlDriver.Driver)

	root := t.TempDir()
	oldMaliceDirName := assets.MaliceDirName
	assets.MaliceDirName = root
	t.Cleanup(func() {
		assets.MaliceDirName = oldMaliceDirName
		config.Reset()
	})
}

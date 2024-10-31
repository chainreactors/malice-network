package build

import (
	"context"
	"fmt"
	"github.com/chainreactors/logs"
	"github.com/chainreactors/malice-network/helper/proto/client/clientpb"
	"github.com/chainreactors/malice-network/server/internal/configs"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
	"path/filepath"
	"time"
)

var (
	NameSpace                   = "ghcr.io/chainreactors"
	Tag                         = "nightly-2024-08-16-latest"
	ContainerSourceCodePath     = "/root/src"
	ContainerCargoRegistryCache = "/root/cargo/registry"
	CargoGitCache               = "/root/cargo/git"
)

func BuildPE(cli *client.Client, req *clientpb.Generate) error {

	SourceCodeVolume := fmt.Sprintf("%s:%s", configs.SourceCodePath, ContainerSourceCodePath)
	CargoRegistryCacheVolume := fmt.Sprintf("%s:%s", filepath.Join(configs.CargoCachePath, "registry"), ContainerCargoRegistryCache)
	CargoGitCacheVolume := fmt.Sprintf("%s:%s", filepath.Join(configs.CargoCachePath, "git"), CargoGitCache)

	timeout := 20 * time.Minute
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	resp, err := cli.ContainerCreate(ctx, &container.Config{
		Image: fmt.Sprintf("%s/%s:%s", NameSpace, req.Target, Tag),
		Cmd:   []string{"cargo", "make", "--disable-check-for-updates", "malefic"},
		Env:   []string{"TARGET_TRIPLE=" + req.Target + ""},
	}, &container.HostConfig{
		AutoRemove: true,
		Binds: []string{
			SourceCodeVolume,
			CargoRegistryCacheVolume,
			CargoGitCacheVolume,
		},
	}, nil, nil, "test-container")
	if err != nil {
		return err
	}

	if err := cli.ContainerStart(ctx, resp.ID, container.StartOptions{}); err != nil {
		logs.Log.Errorf("Error starting container: %v", err)
	}

	logs.Log.Infof("Container %s started successfully.", resp.ID)

	statusCh, errCh := cli.ContainerWait(ctx, resp.ID, container.WaitConditionNotRunning)
	select {
	case err := <-errCh:
		if err != nil {
			return err
		}
	case <-statusCh:
		logs.Log.Infof("Container %s has stopped and will be automatically removed.", resp.ID)
	}

	return nil
}

func BuildDLL(cli *client.Client) error {
	ctx := context.Background()
	resp, err := cli.ContainerCreate(ctx, &container.Config{
		//Image: "ghcr.io/chainreactors/x86_64-unknown-linux-gnu",
		Image: "nginx",
	}, nil, nil, nil, "test")
	if err != nil {
		return err
	}
	logs.Log.Infof("docker resp %s", resp)
	if err := cli.ContainerStart(ctx, resp.ID, container.StartOptions{}); err != nil {
		logs.Log.Errorf("Error starting container: %v", err)
	}
	return nil
}

//func buildDLL() {
//	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
//
//}

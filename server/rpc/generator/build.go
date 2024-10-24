package generator

import (
	"context"
	"github.com/chainreactors/logs"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
)

func BuildPE(cli *client.Client) error {
	ctx := context.Background()

	resp, err := cli.ContainerCreate(ctx, &container.Config{
		Image: "nginx",
		Cmd:   []string{"sh", "-c", "echo hello world && sleep 5"},
	}, &container.HostConfig{
		AutoRemove: true,
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

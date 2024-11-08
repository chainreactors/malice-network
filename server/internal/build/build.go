package build

import (
	"context"
	"fmt"
	"github.com/chainreactors/logs"
	"github.com/chainreactors/malice-network/helper/proto/client/clientpb"
	"github.com/chainreactors/malice-network/server/internal/configs"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

var (
	NameSpace                   = "ghcr.io/chainreactors"
	Tag                         = "nightly-2024-08-16-latest"
	ContainerSourceCodePath     = "/root/src"
	ContainerCargoRegistryCache = "/root/cargo/registry"
	CargoGitCache               = "/root/cargo/git"
	exePath                     = "malefic-mutant.exe"
	command                     = "generate"
	funcNameOption              = "--function-name"
	userDataPathOption          = "--user-data-path"
)

func BuildBeacon(cli *client.Client, req *clientpb.Generate) error {

	SourceCodeVolume := fmt.Sprintf("%s:%s", configs.SourceCodePath, ContainerSourceCodePath)
	CargoRegistryCacheVolume := fmt.Sprintf("%s:%s", filepath.Join(configs.CargoCachePath, "registry"), ContainerCargoRegistryCache)
	CargoGitCacheVolume := fmt.Sprintf("%s:%s", filepath.Join(configs.CargoCachePath, "git"), CargoGitCache)

	timeout := 20 * time.Minute
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	resp, err := cli.ContainerCreate(ctx, &container.Config{
		Image: fmt.Sprintf("%s/%s:%s", NameSpace, req.Target, Tag),
		//Cmd:   []string{"cargo", "make", "--disable-check-for-updates", "malefic"},
		Cmd: []string{"sh", "-c",
			fmt.Sprintf("malefic-mutant config beacon && cargo build --target %s --release -p malefic",
				req.Target)},
		//"cargo run -p malefic-mutant stage0 professional x86_64 source && cargo build --release -p malefic-pulse"},
		Env: []string{"TARGET_TRIPLE=" + req.Target + ""},
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

func BuildBind(cli *client.Client, req *clientpb.Generate) error {

	SourceCodeVolume := fmt.Sprintf("%s:%s", configs.SourceCodePath, ContainerSourceCodePath)
	CargoRegistryCacheVolume := fmt.Sprintf("%s:%s", filepath.Join(configs.CargoCachePath, "registry"), ContainerCargoRegistryCache)
	CargoGitCacheVolume := fmt.Sprintf("%s:%s", filepath.Join(configs.CargoCachePath, "git"), CargoGitCache)

	timeout := 20 * time.Minute
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	resp, err := cli.ContainerCreate(ctx, &container.Config{
		Image: fmt.Sprintf("%s/%s:%s", NameSpace, req.Target, Tag),
		Cmd: []string{"sh", "-c",
			fmt.Sprintf("malefic-mutant config bind  && cargo build --target %s --release -p malefic",
				req.Target)},
		Env: []string{"TARGET_TRIPLE=" + req.Target + ""},
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

func BuildPrelude(cli *client.Client, req *clientpb.Generate) error {

	SourceCodeVolume := fmt.Sprintf("%s:%s", configs.SourceCodePath, ContainerSourceCodePath)
	CargoRegistryCacheVolume := fmt.Sprintf("%s:%s", filepath.Join(configs.CargoCachePath, "registry"), ContainerCargoRegistryCache)
	CargoGitCacheVolume := fmt.Sprintf("%s:%s", filepath.Join(configs.CargoCachePath, "git"), CargoGitCache)

	timeout := 20 * time.Minute
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	resp, err := cli.ContainerCreate(ctx, &container.Config{
		Image: fmt.Sprintf("%s/%s:%s", NameSpace, req.Target, Tag),
		Cmd: []string{"sh", "-c",
			fmt.Sprintf("malefic-mutant prelude autorun.yaml && cargo build --target %s --release -p malefic-prelude",
				req.Target)},
		Env: []string{"TARGET_TRIPLE=" + req.Target + ""},
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

func BuildLoader(cli *client.Client, req *clientpb.Generate) error {

	SourceCodeVolume := fmt.Sprintf("%s:%s", configs.SourceCodePath, ContainerSourceCodePath)
	CargoRegistryCacheVolume := fmt.Sprintf("%s:%s", filepath.Join(configs.CargoCachePath, "registry"), ContainerCargoRegistryCache)
	CargoGitCacheVolume := fmt.Sprintf("%s:%s", filepath.Join(configs.CargoCachePath, "git"), CargoGitCache)

	timeout := 20 * time.Minute
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	resp, err := cli.ContainerCreate(ctx, &container.Config{
		Image: fmt.Sprintf("%s/%s:%s", NameSpace, req.Target, Tag),
		Cmd: []string{"sh", "-c",
			fmt.Sprintf("malefic-mutant prelude autorun.yaml && cargo build --target %s --release -p malefic-prelude",
				req.Target)},
		Env: []string{"TARGET_TRIPLE=" + req.Target + ""},
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

func BuildModules(cli *client.Client, req *clientpb.Generate) error {

	SourceCodeVolume := fmt.Sprintf("%s:%s", configs.SourceCodePath, ContainerSourceCodePath)
	CargoRegistryCacheVolume := fmt.Sprintf("%s:%s", filepath.Join(configs.CargoCachePath, "registry"), ContainerCargoRegistryCache)
	CargoGitCacheVolume := fmt.Sprintf("%s:%s", filepath.Join(configs.CargoCachePath, "git"), CargoGitCache)

	timeout := 20 * time.Minute
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	resp, err := cli.ContainerCreate(ctx, &container.Config{
		Image: fmt.Sprintf("%s/%s:%s", NameSpace, req.Target, Tag),
		Cmd: []string{"sh", "-c",
			fmt.Sprintf("malefic-mutant config modules  && cargo build --target %s --release -p malefic-modules --features %s",
				req.Target, req.Feature)},
		Env: []string{"TARGET_TRIPLE=" + req.Target + ""},
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

//func buildDLL() {
//	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
//
//}

func MaleficSRDI(req *clientpb.MutantFile, srcPath, dstPath string) ([]byte, error) {
	var cmd *exec.Cmd
	if req.Id != "" {
		dstPath = strings.TrimSuffix(dstPath, filepath.Ext(dstPath))
		cmd = exec.Command(exePath, command, req.Type, srcPath, req.Platform, req.Arch, dstPath)
	} else {
		nameWithoutExt := strings.TrimSuffix(req.Name, filepath.Ext(req.Name))
		srcPath = filepath.Join(configs.BuildOutputPath, req.Name)
		dstPath = filepath.Join(configs.SRDIOutputPath, nameWithoutExt)
		cmd = exec.Command(exePath, command, req.Type, srcPath, req.Platform, req.Arch, dstPath)
	}
	output, err := cmd.CombinedOutput()
	if err != nil {
		return []byte{}, err
	}
	logs.Log.Infof("SRDI output: %s", string(output))
	data, err := os.ReadFile(dstPath)
	if err != nil {
		return []byte{}, err
	}
	return data, nil
}

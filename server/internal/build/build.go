package build

import (
	"context"
	"fmt"
	"github.com/chainreactors/logs"
	"github.com/chainreactors/malice-network/helper/consts"
	"github.com/chainreactors/malice-network/helper/proto/client/clientpb"
	"github.com/chainreactors/malice-network/server/internal/configs"
	"github.com/chainreactors/malice-network/server/internal/db"
	"github.com/chainreactors/malice-network/server/internal/db/models"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"time"
)

var (
	NameSpace                   = "ghcr.io/chainreactors"
	Tag                         = "nightly-2024-08-16-latest"
	ContainerSourceCodePath     = "/root/src"
	ContainerCargoRegistryCache = "/root/cargo/registry"
	CargoGitCache               = "/root/cargo/git"
	exePath                     = filepath.Join(configs.SourceCodePath, "malefic_mutant")
	command                     = "generate"
	funcNameOption              = "--function-name"
	userDataPathOption          = "--user-data-path"
)

var dockerClient *client.Client
var once sync.Once

func GetDockerClient() (*client.Client, error) {
	var err error
	once.Do(func() {
		dockerClient, err = client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
		if err != nil {
			logs.Log.Errorf("Error creating Docker client: %v", err)
		}
	})
	return dockerClient, err
}

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
			fmt.Sprintf("./malefic_mutant generate beacon && cargo build --target %s --release -p malefic",
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
			fmt.Sprintf("./malefic_mutant generate bind  && cargo build --target %s --release -p malefic",
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
			fmt.Sprintf("./malefic_mutant generate prelude autorun.yaml && cargo build --target %s --release -p malefic-prelude",
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
			fmt.Sprintf("./malefic_mutant prelude autorun.yaml && cargo build --target %s --release -p malefic-prelude",
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
			fmt.Sprintf("./malefic_mutant generate modules %s -s && cargo build --target %s --release -p malefic-modules --features %s",
				req.Feature, req.Target, req.Feature)},
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

func SaveArtifact(dst string, bin []byte) error {
	filename := filepath.Join(configs.BuildOutputPath, dst)
	err := os.WriteFile(filename, bin, 0644)
	if err != nil {
		return err
	}
	return nil
}

func NewMaleficSRDIArtifact(name, src, platform, arch, stage, funcName, dataPath string) (*models.Builder, []byte, error) {
	builder, err := db.SaveArtifact(name, "srdi", stage)
	if err != nil {
		return nil, nil, err
	}
	bin, err := MaleficSRDI(src, builder.Path, platform, arch, funcName, dataPath)
	if err != nil {
		return nil, nil, err
	}
	err = os.WriteFile(builder.Path, bin, 0644)
	if err != nil {
		return nil, nil, err
	}
	return builder, bin, nil
}

func MaleficSRDI(src, dst, platform, arch, funcName, dataPath string) ([]byte, error) {
	args := []string{command, consts.SRDIType, src, platform, arch, dst}
	if funcName != "" {
		args = append(args, funcNameOption, funcName)
	}
	if dataPath != "" {
		args = append(args, userDataPathOption, dataPath)
	}

	cmd := exec.Command(exePath, args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return []byte{}, err
	}
	logs.Log.Infof("SRDI output: %s", string(output))
	data, err := os.ReadFile(dst)
	if err != nil {
		return []byte{}, err
	}

	return data, nil
}

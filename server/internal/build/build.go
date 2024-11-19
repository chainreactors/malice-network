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
	"math/rand"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

var (
	NameSpace                   = "ghcr.io/chainreactors"
	Tag                         = "nightly-2023-09-18-latest"
	ContainerSourceCodePath     = "/root/src"
	ContainerCargoRegistryCache = "/root/cargo/registry"
	ContainerCargoGitCache      = "/root/cargo/git"
	ContainerBinPath            = "/root/bin"
	LocalMutantPath             = filepath.Join(configs.BinPath, "malefic-mutant")
	command                     = "generate"
	funcNameOption              = "--function-name"
	userDataPathOption          = "--userdata-path"

	sourcePath, _            = filepath.Abs(configs.SourceCodePath)
	binPath, _               = filepath.Abs(configs.BinPath)
	registryPath, _          = filepath.Abs(filepath.Join(configs.CargoCachePath, "registry"))
	gitPath, _               = filepath.Abs(filepath.Join(configs.CargoCachePath, "git"))
	SourceCodeVolume         = fmt.Sprintf("%s:%s", filepath.ToSlash(sourcePath), ContainerSourceCodePath)
	CargoRegistryCacheVolume = fmt.Sprintf("%s:%s", filepath.ToSlash(registryPath), ContainerCargoRegistryCache)
	CargoGitCacheVolume      = fmt.Sprintf("%s:%s", filepath.ToSlash(gitPath), ContainerCargoGitCache)
	BinPathVolume            = fmt.Sprintf("%s:%s", filepath.ToSlash(binPath), ContainerBinPath)
	Volumes                  = []string{SourceCodeVolume, CargoRegistryCacheVolume, CargoGitCacheVolume, BinPathVolume}
)

var dockerClient *client.Client
var once sync.Once

func generateContainerName(length int) string {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	src := rand.NewSource(time.Now().UnixNano())
	r := rand.New(src)
	randomPart := make([]byte, length)
	for i := range randomPart {
		randomPart[i] = charset[r.Intn(len(charset))]
	}
	return string(randomPart)
}

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
	timeout := 20 * time.Minute
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	buildBeaconCommand := fmt.Sprintf(
		"%s/malefic-mutant generate beacon && cargo build --target %s --release -p malefic",
		ContainerBinPath,
		req.Target,
	)
	containerName := "malefic_" + generateContainerName(8)
	resp, err := cli.ContainerCreate(ctx, &container.Config{
		Image: fmt.Sprintf("%s/%s:%s", NameSpace, req.Target, Tag),
		Cmd:   []string{"sh", "-c", buildBeaconCommand},
		//"cargo run -p malefic-mutant stage0 professional x86_64 source && cargo build --release -p malefic-pulse"},
	}, &container.HostConfig{
		//AutoRemove: true,
		Binds: Volumes,
	}, nil, nil, containerName)
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
	timeout := 20 * time.Minute
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	BuildBindCommand := fmt.Sprintf(
		"%s/malefic-mutant generate bind  && cargo build --target %s --release -p malefic",
		ContainerBinPath,
		req.Target,
	)
	containerName := "malefic_" + generateContainerName(8)

	resp, err := cli.ContainerCreate(ctx, &container.Config{
		Image: fmt.Sprintf("%s/%s:%s", NameSpace, req.Target, Tag),
		Cmd:   []string{"sh", "-c", BuildBindCommand},
	}, &container.HostConfig{
		AutoRemove: true,
		Binds:      Volumes,
	}, nil, nil, containerName)
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

	timeout := 20 * time.Minute
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	BuildPreludeCommand := fmt.Sprintf(
		"%s/malefic-mutant generate prelude autorun.yaml && cargo build --target %s --release -p malefic-prelude",
		ContainerBinPath,
		req.Target,
	)
	containerName := "malefic_" + generateContainerName(8)
	resp, err := cli.ContainerCreate(ctx, &container.Config{
		Image: fmt.Sprintf("%s/%s:%s", NameSpace, req.Target, Tag),
		Cmd:   []string{"sh", "-c", BuildPreludeCommand},
	}, &container.HostConfig{
		AutoRemove: true,
		Binds:      Volumes,
	}, nil, nil, containerName)
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

func BuildPulse(cli *client.Client, req *clientpb.Generate) error {
	timeout := 20 * time.Minute
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	BuildBindCommand := fmt.Sprintf(
		"%s/malefic-mutant generate pulse x64 win && cargo build --target %s --release --profile release-lto -p malefic-pulse",
		ContainerBinPath,
		req.Target,
	)
	containerName := "malefic_" + generateContainerName(8)
	resp, err := cli.ContainerCreate(ctx, &container.Config{
		Image: fmt.Sprintf("%s/%s:%s", NameSpace, req.Target, Tag),
		Cmd:   []string{"sh", "-c", BuildBindCommand},
	}, &container.HostConfig{
		AutoRemove: true,
		Binds:      Volumes,
	}, nil, nil, containerName)
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
	timeout := 20 * time.Minute
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	containerName := "malefic_" + generateContainerName(8)
	buildModules := strings.Join(req.Modules, ",")
	buildModulesCommand := fmt.Sprintf(
		"%s/malefic-mutant generate modules %s -s && cargo build --target %s --release -p malefic-modules --features %s",
		ContainerBinPath,
		buildModules,
		req.Target,
		buildModules,
	)
	resp, err := cli.ContainerCreate(ctx, &container.Config{
		Image: fmt.Sprintf("%s/%s:%s", NameSpace, req.Target, Tag),
		Cmd:   []string{"sh", "-c", buildModulesCommand},
	}, &container.HostConfig{
		AutoRemove: true,
		Binds: []string{
			SourceCodeVolume,
			CargoRegistryCacheVolume,
			CargoGitCacheVolume,
		},
	}, nil, nil, containerName)
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

func SaveArtifact(dst string, bin []byte) error {
	filename := filepath.Join(configs.BuildOutputPath, dst)
	err := os.WriteFile(filename, bin, 0644)
	if err != nil {
		return err
	}
	return nil
}

func NewMaleficSRDIArtifact(name, src, platform, arch, stage, funcName, dataPath string) (*models.Builder, []byte, error) {
	builder, err := db.SaveArtifact(name, "srdi", platform, arch, stage)
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

	cmd := exec.Command(LocalMutantPath, args...)
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

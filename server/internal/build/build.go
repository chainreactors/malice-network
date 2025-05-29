package build

import (
	"bufio"
	"context"
	"fmt"
	"github.com/chainreactors/malice-network/helper/errs"
	"github.com/chainreactors/malice-network/helper/utils/fileutils"
	"github.com/docker/docker/api/types"
	"github.com/wabzsy/gonut"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/chainreactors/logs"
	"github.com/chainreactors/malice-network/helper/consts"
	"github.com/chainreactors/malice-network/helper/cryptography"
	"github.com/chainreactors/malice-network/helper/encoders"
	"github.com/chainreactors/malice-network/helper/proto/client/clientpb"
	"github.com/chainreactors/malice-network/server/internal/configs"
	"github.com/chainreactors/malice-network/server/internal/core"
	"github.com/chainreactors/malice-network/server/internal/db"
	"github.com/chainreactors/malice-network/server/internal/db/models"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
)

var (
	//NameSpace                   = "ghcr.io/chainreactors"
	//Tag                         = "nightly-2023-09-18-latest"
	DefaultImage                = GetDefaultImage()
	ContainerSourceCodePath     = "/root/src"
	ContainerCargoRegistryCache = "/root/cargo/registry"
	ContainerCargoGitCache      = "/root/cargo/git"
	ContainerBinPath            = "/root/bin"
	LocalMutantPath             = filepath.Join(configs.BinPath, "malefic-mutant")
	command                     = "build"
	funcNameOption              = "--function-name"
	userDataPathOption          = "--userdata-path"

	autorunPath              = filepath.Join(configs.SourceCodePath, "autorun.yaml")
	sourcePath, _            = filepath.Abs(configs.SourceCodePath)
	binPath, _               = filepath.Abs(configs.BinPath)
	registryPath, _          = filepath.Abs(filepath.Join(configs.CargoCachePath, "registry"))
	gitPath, _               = filepath.Abs(filepath.Join(configs.CargoCachePath, "git"))
	SourceCodeVolume         = fmt.Sprintf("%s:%s", filepath.ToSlash(sourcePath), ContainerSourceCodePath)
	CargoRegistryCacheVolume = fmt.Sprintf("%s:%s", filepath.ToSlash(registryPath), ContainerCargoRegistryCache)
	CargoGitCacheVolume      = fmt.Sprintf("%s:%s", filepath.ToSlash(gitPath), ContainerCargoGitCache)
	BinPathVolume            = fmt.Sprintf("%s:%s", filepath.ToSlash(binPath), ContainerBinPath)
	//Volumes                  = []string{SourceCodeVolume, CargoRegistryCacheVolume, CargoGitCacheVolume, BinPathVolume}
	Volumes  = []string{SourceCodeVolume, BinPathVolume}
	PATH_ENV = ContainerBinPath + ":/root/cargo/bin:/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin:/opt/osxcross/bin:/usr/bin/mingw-w64"
)

var dockerClient *client.Client
var once sync.Once

func GetDefaultImage() string {
	return "ghcr.io/chainreactors/malefic-builder:" + consts.Ver
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
		"malefic-mutant generate beacon && malefic-mutant build malefic -t %s",
		req.Target,
	)
	containerName := "malefic_" + cryptography.RandomString(8)
	resp, err := cli.ContainerCreate(ctx, &container.Config{
		Env: []string{
			"PATH=" + PATH_ENV,
		},
		Image: DefaultImage,
		Cmd:   []string{"sh", "-c", buildBeaconCommand},
		//"cargo run -p malefic-mutant stage0 professional x86_64 source && cargo build --release -p malefic-pulse"},
	}, &container.HostConfig{
		AutoRemove: true,
		Binds:      Volumes,
	}, nil, nil, containerName)
	if err != nil {
		return err
	}
	if err := cli.ContainerStart(ctx, resp.ID, types.ContainerStartOptions{}); err != nil {
		logs.Log.Errorf("Error starting container: %v", err)
	}
	sendContainerCtrlMsg(false, containerName, req)
	logs.Log.Infof("Container %s started successfully.", resp.ID)
	err = catchLogs(cli, resp.ID, req.Name)
	if err != nil {
		return err
	}
	statusCh, errCh := cli.ContainerWait(ctx, resp.ID, container.WaitConditionNotRunning)
	select {
	case err := <-errCh:
		if err != nil {
			if strings.Contains(err.Error(), "No such container") {
				return nil
			}
			return err
		}
	case <-statusCh:
		logs.Log.Infof("Container %s has stopped and will be automatically removed.", resp.ID)
	}
	sendContainerCtrlMsg(true, containerName, req)
	return nil
}

func BuildBind(cli *client.Client, req *clientpb.Generate) error {
	timeout := 20 * time.Minute
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	BuildBindCommand := fmt.Sprintf(
		"malefic-mutant generate bind && malefic-mutant build malefic -t %s",
		req.Target,
	)
	containerName := "malefic_" + cryptography.RandomString(8)

	resp, err := cli.ContainerCreate(ctx, &container.Config{
		Image: DefaultImage,
		Cmd:   []string{"sh", "-c", BuildBindCommand},
		Env: []string{
			"PATH=" + PATH_ENV,
		},
	}, &container.HostConfig{
		AutoRemove: true,
		Binds:      Volumes,
	}, nil, nil, containerName)
	if err != nil {
		return err
	}

	if err := cli.ContainerStart(ctx, resp.ID, types.ContainerStartOptions{}); err != nil {
		logs.Log.Errorf("Error starting container: %v", err)
	}

	sendContainerCtrlMsg(false, containerName, req)
	logs.Log.Infof("Container %s started successfully.", resp.ID)

	err = catchLogs(cli, resp.ID, req.Name)
	if err != nil {
		return err
	}
	statusCh, errCh := cli.ContainerWait(ctx, resp.ID, container.WaitConditionNotRunning)
	select {
	case err := <-errCh:
		if err != nil {
			if strings.Contains(err.Error(), "No such container") {
				return nil
			}
			return err
		}
	case <-statusCh:
		logs.Log.Infof("Container %s has stopped and will be automatically removed.", resp.ID)
	}
	sendContainerCtrlMsg(true, containerName, req)
	return nil
}

func BuildPrelude(cli *client.Client, req *clientpb.Generate) error {
	err := os.WriteFile(autorunPath, req.Bin, 0644)
	if err != nil {
		return err
	}
	timeout := 20 * time.Minute
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	BuildPreludeCommand := fmt.Sprintf(
		"malefic-mutant generate prelude && malefic-mutant build prelude -t %s",
		req.Target,
	)
	containerName := "malefic_" + cryptography.RandomString(8)
	resp, err := cli.ContainerCreate(ctx, &container.Config{
		Image: DefaultImage,
		Cmd:   []string{"sh", "-c", BuildPreludeCommand},
		Env: []string{
			"PATH=" + PATH_ENV,
		},
	}, &container.HostConfig{
		AutoRemove: true,
		Binds:      Volumes,
	}, nil, nil, containerName)
	if err != nil {
		return err
	}

	if err := cli.ContainerStart(ctx, resp.ID, types.ContainerStartOptions{}); err != nil {
		logs.Log.Errorf("Error starting container: %v", err)
	}

	sendContainerCtrlMsg(false, containerName, req)
	logs.Log.Infof("Container %s started successfully.", resp.ID)

	err = catchLogs(cli, resp.ID, req.Name)
	if err != nil {
		return err
	}

	statusCh, errCh := cli.ContainerWait(ctx, resp.ID, container.WaitConditionNotRunning)
	select {
	case err := <-errCh:
		if err != nil {
			if strings.Contains(err.Error(), "No such container") {
				return nil
			}
			return err
		}
	case <-statusCh:
		logs.Log.Infof("Container %s has stopped and will be automatically removed.", resp.ID)
	}
	sendContainerCtrlMsg(true, containerName, req)
	return nil
}

func BuildPulse(cli *client.Client, req *clientpb.Generate) error {
	timeout := 20 * time.Minute
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	target, ok := consts.GetBuildTarget(req.Target)
	var pulseOs string
	if target.OS == consts.Windows {
		pulseOs = "win"
	} else {
		pulseOs = target.OS
	}
	if !ok {
		return fmt.Errorf("invalid target: %s", req.Target)
	}

	BuildBindCommand := fmt.Sprintf(
		"malefic-mutant generate pulse %s %s &&malefic-mutant build pulse -t %s",
		target.Arch, pulseOs, req.Target,
	)
	containerName := "malefic_" + cryptography.RandomString(8)
	resp, err := cli.ContainerCreate(ctx, &container.Config{
		Image: DefaultImage,
		Cmd:   []string{"sh", "-c", BuildBindCommand},
		Env: []string{
			"PATH=" + PATH_ENV,
		},
	}, &container.HostConfig{
		AutoRemove: true,
		Binds:      Volumes,
	}, nil, nil, containerName)
	if err != nil {
		return err
	}

	if err := cli.ContainerStart(ctx, resp.ID, types.ContainerStartOptions{}); err != nil {
		logs.Log.Errorf("Error starting container: %v", err)
	}

	sendContainerCtrlMsg(false, containerName, req)
	logs.Log.Infof("Container %s started successfully.", resp.ID)

	err = catchLogs(cli, resp.ID, req.Name)
	if err != nil {
		return err
	}

	statusCh, errCh := cli.ContainerWait(ctx, resp.ID, container.WaitConditionNotRunning)
	select {
	case err := <-errCh:
		if err != nil {
			if strings.Contains(err.Error(), "No such container") {
				return nil
			}
			return err
		}
	case <-statusCh:
		logs.Log.Infof("Container %s has stopped and will be automatically removed.", resp.ID)
	}
	sendContainerCtrlMsg(true, containerName, req)
	return nil
}

func BuildModules(cli *client.Client, req *clientpb.Generate) error {
	timeout := 20 * time.Minute
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	containerName := "malefic_" + cryptography.RandomString(8)
	var buildModulesCommand string
	buildModulesCommand = fmt.Sprintf(
		"malefic-mutant generate modules && malefic-mutant build modules -t %s",
		req.Target,
	)
	resp, err := cli.ContainerCreate(ctx, &container.Config{
		Image: DefaultImage,
		Cmd:   []string{"sh", "-c", buildModulesCommand},
		Env: []string{
			"PATH=" + PATH_ENV,
		},
	}, &container.HostConfig{
		AutoRemove: true,
		Binds:      Volumes,
	}, nil, nil, containerName)
	if err != nil {
		return err
	}

	if err := cli.ContainerStart(ctx, resp.ID, types.ContainerStartOptions{}); err != nil {
		logs.Log.Errorf("Error starting container: %v", err)
	}

	sendContainerCtrlMsg(false, containerName, req)
	logs.Log.Infof("Container %s started successfully.", resp.ID)

	err = catchLogs(cli, resp.ID, req.Name)
	if err != nil {
		return err
	}

	statusCh, errCh := cli.ContainerWait(ctx, resp.ID, container.WaitConditionNotRunning)
	select {
	case err := <-errCh:
		if err != nil {
			if strings.Contains(err.Error(), "No such container") {
				return nil
			}
			return err
		}
	case <-statusCh:
		logs.Log.Infof("Container %s has stopped and will be automatically removed.", resp.ID)
	}

	sendContainerCtrlMsg(true, containerName, req)

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

func NewMaleficSRDIArtifact(name, typ, src, platform, arch, stage, funcName, dataPath string) (*models.Builder, []byte, error) {
	builder, err := db.SaveArtifact(name, typ, platform, arch, stage, consts.CommandArtifactUpload)
	if err != nil {
		return nil, nil, err
	}
	bin, err := gonut.DonutShellcodeFromFile(builder.Path, arch, "")
	if err != nil {
		return nil, nil, err
	}
	err = os.WriteFile(builder.ShellcodePath, bin, 0644)
	if err != nil {
		return nil, nil, err
	}
	return builder, bin, nil
}

// for pulse
func OBJCOPYPulse(builder *models.Builder, platform, arch string) ([]byte, error) {
	absBuildOutputPath, err := filepath.Abs(configs.BuildOutputPath)
	if err != nil {
		return nil, err
	}
	dstPath := filepath.Join(absBuildOutputPath, encoders.UUID())
	cmd := exec.Command("objcopy", "-O", "binary", builder.Path, dstPath)
	cmd.Dir = sourcePath
	output, err := cmd.CombinedOutput()
	logs.Log.Debugf("Objcopy output: %s", output)
	if err != nil {
		return nil, err
	}
	bin, err := os.ReadFile(dstPath)
	if err != nil {
		return nil, err
	}
	builder.ShellcodePath = dstPath
	err = db.UpdateBuilderSrdi(builder)
	if err != nil {
		return nil, err
	}
	return bin, nil
}

func SRDIArtifact(builder *models.Builder, platform, arch string) ([]byte, error) {
	if !strings.Contains(builder.Target, consts.Windows) {
		builder.IsSRDI = false
		err := db.UpdateBuilderSrdi(builder)
		if err != nil {
			return nil, err
		}
		return []byte{}, errs.ErrPlartFormNotSupport
	}
	absBuildOutputPath, err := filepath.Abs(configs.BuildOutputPath)
	if err != nil {
		return nil, err
	}
	dstPath := filepath.Join(absBuildOutputPath, encoders.UUID())
	exePath := builder.Path
	if !strings.HasSuffix(exePath, ".exe") {
		exePath = builder.Path + ".exe"
		err = fileutils.CopyFile(builder.Path, exePath)
		if err != nil {
			return nil, fmt.Errorf("copy to .exe failed: %w", err)
		}
	}
	bin, err := gonut.DonutShellcodeFromFile(exePath, arch, "")
	if err != nil {
		return nil, err
	}
	err = os.WriteFile(dstPath, bin, 0644)
	if err != nil {
		return nil, err
	}
	builder.ShellcodePath = dstPath
	err = db.UpdateBuilderSrdi(builder)
	if err != nil {
		return nil, err
	}
	err = fileutils.RemoveFile(exePath)
	if err != nil {
		return nil, err
	}
	return bin, nil
}

func catchLogs(cli *client.Client, containerID, name string) error {
	logOptions := types.ContainerLogsOptions{
		ShowStdout: true,
		ShowStderr: true,
		Follow:     true,
		Timestamps: true,
	}

	logReader, err := cli.ContainerLogs(context.Background(), containerID, logOptions)
	if err != nil {
		logs.Log.Errorf("Error fetching logs for container %s: %v", containerID, err)
		return err
	}
	defer logReader.Close()

	reader := bufio.NewReader(logReader)
	var logBuffer strings.Builder

	for {
		line, err := reader.ReadString('\n')
		if err != nil && err != io.EOF {
			logs.Log.Errorf("Error reading logs for container %s: %v", containerID, err)
			return err
		}

		if err == io.EOF {
			break
		}
		re := regexp.MustCompile(`[\x00-\x1F&&[^\n]]+`)
		line = re.ReplaceAllString(line, "")
		logBuffer.WriteString(line)
		db.UpdateBuilderLog(name, line)
	}

	return nil
}

func sendContainerCtrlMsg(isEnd bool, containerName string, req *clientpb.Generate) {
	if isEnd {
		core.EventBroker.Publish(core.Event{
			EventType: consts.EventBuild,
			IsNotify:  false,
			Message:   fmt.Sprintf("%s type(%s) has completed in container(%s). run `artifact list` to get the artifact.", req.Name, req.Type, containerName),
			Important: true,
		})
	} else {
		core.EventBroker.Publish(core.Event{
			EventType: consts.EventBuild,
			IsNotify:  false,
			Message:   fmt.Sprintf("%s type(%s) has started in container(%s)...", req.Name, req.Type, containerName),
			Important: true,
		})
	}
}

func GetDockerStatus(cli *client.Client, containerName string) (string, error) {
	ctx := context.Background()
	containers, err := cli.ContainerList(ctx, types.ContainerListOptions{All: true})
	if err != nil {
		return "", fmt.Errorf("failed to list containers: %v", err)
	}

	for _, container := range containers {
		if container.Names[0] == "/"+containerName {
			return container.State, nil
		}
	}
	return "", fmt.Errorf("container %s not found", containerName)
}

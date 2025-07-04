package build

import (
	"bufio"
	"context"
	"fmt"
	"github.com/chainreactors/logs"
	"github.com/chainreactors/malice-network/helper/consts"
	"github.com/chainreactors/malice-network/helper/proto/client/clientpb"
	"github.com/chainreactors/malice-network/server/internal/configs"
	"github.com/chainreactors/malice-network/server/internal/core"
	"github.com/chainreactors/malice-network/server/internal/db"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
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

func SaveArtifact(dst string, bin []byte) error {
	filename := filepath.Join(configs.BuildOutputPath, dst)
	err := os.WriteFile(filename, bin, 0644)
	if err != nil {
		return err
	}
	return nil
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

func sendContainerCtrlMsg(isEnd bool, containerName string, req *clientpb.BuildConfig, err error) {
	if core.EventBroker == nil {
		return
	}
	if err != nil {
		core.EventBroker.Publish(core.Event{
			EventType: consts.EventBuild,
			IsNotify:  false,
			Message:   fmt.Sprintf("%s type(%s)  container(%s) has a err %v. ", req.BuildName, req.Type, containerName, err),
			Important: true,
		})
	}
	if isEnd {
		core.EventBroker.Publish(core.Event{
			EventType: consts.EventBuild,
			IsNotify:  false,
			Message:   fmt.Sprintf("%s type(%s) has completed in container(%s). run `artifact list` to get the artifact.", req.BuildName, req.Type, containerName),
			Important: true,
		})
	} else {
		core.EventBroker.Publish(core.Event{
			EventType: consts.EventBuild,
			IsNotify:  false,
			Message:   fmt.Sprintf("%s type(%s) has started in container(%s)...", req.BuildName, req.Type, containerName),
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

package build

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/chainreactors/IoM-go/consts"
	"github.com/chainreactors/IoM-go/proto/client/clientpb"
	"github.com/chainreactors/logs"
	"github.com/chainreactors/malice-network/server/internal/configs"
	"github.com/chainreactors/malice-network/server/internal/core"
	"github.com/chainreactors/malice-network/server/internal/db"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
	"github.com/docker/docker/pkg/stdcopy"
)

var (
	//NameSpace                   = "ghcr.io/chainreactors"
	//Tag                         = "nightly-2023-09-18-latest"
	Ver                          = "latest"
	ContainerSourceCodePath      = "/root/src"
	ContainerCargoRegistryCache  = "/root/cargo/registry"
	ContainerCargoGitCache       = "/root/cargo/git"
	ContainerBinPath             = "/root/bin"
	ContainerBuiltinResourcePath = "/tmp/builtin/resources"
	ContainerCustomResourcePath  = "/tmp/custom/resources"
	ContainerResourcePath        = "/root/src/resources"
	ContainerAutoRunPath         = "/root/src/prelude.yaml"
	ContainerConfigPath          = "/root/src/implant.yaml"
	command                      = "build"
	funcNameOption               = "--function-name"
	userDataPathOption           = "--userdata-path"

	PATH_ENV = ContainerBinPath + ":/root/cargo/bin:/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin:/opt/osxcross/bin:/usr/bin/mingw-w64"
)

// GetVolumes returns Docker volume mounts computed from current configs paths.
// Called at build time (not init time) so that UpdateSourceCodeRoot takes effect.
func GetVolumes() []string {
	sp, _ := filepath.Abs(configs.SourceCodePath)
	bp, _ := filepath.Abs(configs.BinPath)
	return []string{
		fmt.Sprintf("%s:%s", filepath.ToSlash(sp), ContainerSourceCodePath),
		fmt.Sprintf("%s:%s", filepath.ToSlash(bp), ContainerBinPath),
	}
}

var dockerClient *client.Client
var once sync.Once
var dockerStdCopy = stdcopy.StdCopy
var updateBuilderLog = db.UpdateBuilderLog

func GetImage(target string) string {
	if target == consts.TargetX64Windows || target == consts.TargetX86Windows {
		return "ghcr.io/chainreactors/" + consts.TargetX64Windows + ":" + Ver
	}
	return "ghcr.io/chainreactors/malefic-builder:" + Ver
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
	filename := filepath.Join(configs.TempPath, dst)
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

	stdoutPipe, stdoutWriter := io.Pipe()
	stderrPipe, stderrWriter := io.Pipe()

	var wg sync.WaitGroup
	errCh := make(chan error, 3)

	runLogWorker(&wg, errCh, "docker-log-copy:"+name, func() error {
		_, err := dockerStdCopy(stdoutWriter, stderrWriter, logReader)
		if err != nil && !errors.Is(err, io.EOF) {
			return fmt.Errorf("demultiplex logs for container %s: %w", containerID, err)
		}
		return nil
	}, func() {
		_ = stdoutWriter.Close()
		_ = stderrWriter.Close()
	})

	runLogWorker(&wg, errCh, "docker-log-stdout:"+name, func() error {
		return consumeLogPipe(stdoutPipe, name)
	}, func() {
		_ = stdoutPipe.Close()
	})

	runLogWorker(&wg, errCh, "docker-log-stderr:"+name, func() error {
		return consumeLogPipe(stderrPipe, name)
	}, func() {
		_ = stderrPipe.Close()
	})

	wg.Wait()
	close(errCh)

	var allErrs []error
	for err := range errCh {
		allErrs = append(allErrs, err)
	}
	return errors.Join(allErrs...)
}

func runLogWorker(wg *sync.WaitGroup, errCh chan<- error, label string, fn func() error, cleanups ...func()) {
	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := core.RunGuarded(label, fn, core.LogGuardedError(label), cleanups...); err != nil {
			errCh <- err
		}
	}()
}

func consumeLogPipe(r io.Reader, name string) error {
	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		line := sanitizeLogLine(scanner.Text()) + "\n"
		updateBuilderLog(name, line)
	}
	if err := scanner.Err(); err != nil && !errors.Is(err, io.EOF) {
		return fmt.Errorf("scan builder log %s: %w", name, err)
	}
	return nil
}

func sanitizeLogLine(line string) string {
	return strings.Map(func(r rune) rune {
		if r == '\n' || r == '\r' || r == '\t' {
			return r
		}
		if r >= 0 && r < 0x20 {
			return -1
		}
		return r
	}, line)
}

func sendContainerCtrlMsg(isEnd bool, containerName string, req *clientpb.BuildConfig, err error) {
	if core.EventBroker == nil {
		return
	}
	if err != nil {
		core.EventBroker.Publish(core.Event{
			EventType: consts.EventBuild,
			IsNotify:  false,
			Message:   fmt.Sprintf("%s type(%s)  container(%s) has a err %v. ", req.BuildName, req.BuildType, containerName, err),
			Important: true,
		})
	}
	if isEnd {
		core.EventBroker.Publish(core.Event{
			EventType: consts.EventBuild,
			IsNotify:  false,
			Message:   fmt.Sprintf("%s type(%s) has completed in container(%s). run `artifact list` to get the artifact.", req.BuildName, req.BuildType, containerName),
			Important: true,
		})
	} else {
		core.EventBroker.Publish(core.Event{
			EventType: consts.EventBuild,
			IsNotify:  false,
			Message:   fmt.Sprintf("%s type(%s) has started in container(%s)...", req.BuildName, req.BuildType, containerName),
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

package build

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/chainreactors/IoM-go/consts"
	"github.com/chainreactors/IoM-go/proto/client/clientpb"
	errs "github.com/chainreactors/IoM-go/types"
	"github.com/chainreactors/logs"
	"github.com/chainreactors/malice-network/helper/codenames"
	"github.com/chainreactors/malice-network/helper/cryptography"
	"github.com/chainreactors/malice-network/helper/encoders"
	selfType "github.com/chainreactors/malice-network/helper/implanttypes"
	"github.com/chainreactors/malice-network/helper/utils/fileutils"
	"github.com/chainreactors/malice-network/server/internal/configs"
	"github.com/chainreactors/malice-network/server/internal/core"
	"github.com/chainreactors/malice-network/server/internal/db"
	"github.com/chainreactors/malice-network/server/internal/db/models"
	"github.com/chainreactors/malice-network/server/internal/mutant"
	"github.com/docker/docker/api/types/container"
)

type DockerBuilder struct {
	config        *clientpb.BuildConfig
	artifact      *models.Artifact
	containerName string
	containerID   string
	enable3rd     bool
	licenseID     string
	srcPath       string
	volumes       []string
	packResources bool
}

const dockerLogDrainTimeout = 30 * time.Second

func resolveDockerMutantBinary() (string, error) {
	candidates := []struct {
		hostPath      string
		containerPath string
	}{
		{
			hostPath:      filepath.Join(configs.TargetPath, "release", "malefic-mutant"),
			containerPath: filepath.Join(ContainerSourceCodePath, "target", "release", "malefic-mutant"),
		},
		{
			hostPath:      filepath.Join(configs.SourceCodePath, "bin", "malefic-mutant"),
			containerPath: filepath.Join(ContainerSourceCodePath, "bin", "malefic-mutant"),
		},
	}

	for _, candidate := range candidates {
		_, err := os.Stat(candidate.hostPath)
		if err == nil {
			if err := mutant.CheckBinaryExecutable(candidate.hostPath); err != nil {
				return "", err
			}
			return filepath.ToSlash(candidate.containerPath), nil
		}
		if err != nil && !os.IsNotExist(err) {
			return "", fmt.Errorf("failed to stat malefic-mutant candidate %s: %w", candidate.hostPath, err)
		}
	}

	return "malefic-mutant", nil
}

func NewDockerBuilder(req *clientpb.BuildConfig) *DockerBuilder {
	os.MkdirAll(configs.BuildOutputPath, 0700)
	os.MkdirAll(configs.SourceCodePath, 0700)
	os.MkdirAll(configs.ResourcePath, 0700)
	return &DockerBuilder{
		config: req,
	}
}

func (d *DockerBuilder) Generate() (*clientpb.Artifact, error) {
	// init config
	// generate config.yaml
	var artifact *models.Artifact
	var err error
	//var profile *selfType.ProfileConfig
	if d.config.BuildName == "" {
		d.config.BuildName = codenames.GetCodename()
	}
	// get profile
	if needsProfileFiles(d.config) {
		implant, prelude, resources, pErr := db.GetProfileFullConfig(d.config.ProfileName)
		if pErr != nil {
			return nil, fmt.Errorf("failed to get profile config: %s", pErr)
		}
		mergeProfileFiles(d.config, implant, prelude, resources)
	}
	profile, err := selfType.LoadProfile(d.config.MaleficConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to get config: %s", err)
	}
	if d.config.BuildType == consts.CommandBuildProxyDll {
		if profile.Loader == nil {
			profile.Loader = &selfType.LoaderProfile{}
		}
		if profile.Loader.ProxyDll == nil {
			profile.Loader.ProxyDll = &selfType.ProxyDllProfile{}
		}
		if !profile.Loader.ProxyDll.PackResources {
			profile.Loader.ProxyDll.PackResources = true
			normalizedConfig, yErr := profile.ToYAML()
			if yErr != nil {
				return nil, fmt.Errorf("failed to normalize proxydll config: %w", yErr)
			}
			d.config.MaleficConfig = normalizedConfig
		}
	}

	// init artifact status
	artifactId := d.config.ArtifactId
	// save artifact and update status
	if artifactId != 0 && (d.config.BuildType == consts.CommandBuildBeacon || d.config.BuildType == consts.CommandBuildBind) {
		artifact, err = db.SaveArtifactFromID(d.config, artifactId)
	} else {
		artifact, err = db.SaveArtifactFromConfig(d.config)
	}
	if err != nil {
		logs.Log.Errorf("failed to create %s", err)
		return nil, err
	}
	d.artifact = artifact
	db.UpdateBuilderStatus(d.artifact.ID, consts.BuildStatusWaiting)

	//
	d.srcPath = filepath.Join(configs.TempPath, encoders.UUID())
	os.MkdirAll(d.srcPath, 0700)
	// for saas
	//profilePath := filepath.Join(configs.ProfilePath, d.config.ProfileName)
	if d.licenseID != "" {
		//profilePath = ""
		d.srcPath = filepath.Join(configs.TempPath, d.licenseID)
	}
	// writeBuildConfigTo src tmpDir
	err = WriteBuildConfigToPath(d.config, d.srcPath)
	if err != nil {
		return nil, err
	}
	// set volume - 精确挂载特定文件和目录
	d.volumes = GetVolumes()

	// 挂载内置 resources（镜像预留资源）
	rp, _ := filepath.Abs(configs.ResourcePath)
	builtinResourceVolume := fmt.Sprintf("%s:%s", filepath.ToSlash(rp), ContainerBuiltinResourcePath)
	d.volumes = append(d.volumes, builtinResourceVolume)

	// 挂载 implant.yaml（如果存在）
	configPath := filepath.Join(d.srcPath, "implant.yaml")
	if fileutils.Exist(configPath) {
		configVolume := fmt.Sprintf("%s:%s", filepath.ToSlash(configPath), ContainerConfigPath)
		d.volumes = append(d.volumes, configVolume)
	}

	// 挂载 prelude.yaml（必须存在）
	autorunPath := filepath.Join(d.srcPath, "prelude.yaml")
	if fileutils.Exist(autorunPath) {
		autoRunVolume := fmt.Sprintf("%s:%s", filepath.ToSlash(autorunPath), ContainerAutoRunPath)
		d.volumes = append(d.volumes, autoRunVolume)
	}

	// 挂载自定义 resources 目录（如果存在且不为空）
	customResourcesPath := filepath.Join(d.srcPath, "resources")
	if fileutils.Exist(customResourcesPath) {
		entries, err := os.ReadDir(customResourcesPath)
		if err == nil && len(entries) > 0 {
			customResourceVolume := fmt.Sprintf("%s:%s", filepath.ToSlash(customResourcesPath), ContainerCustomResourcePath)
			d.volumes = append(d.volumes, customResourceVolume)
		}
	}
	return artifact.ToProtobuf([]byte{}), nil
}

func (d *DockerBuilder) Execute() error {
	dockerBuildSemaphore <- struct{}{}
	defer func() { <-dockerBuildSemaphore }()
	timeout := 20 * time.Minute
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	cli, err := GetDockerClient()
	if err != nil {
		return err
	}
	var buildCommand string
	profile, err := selfType.LoadProfileFromContent(d.config.MaleficConfig)
	if err != nil {
		return err
	}

	libFlag := ""
	if d.config.OutputType == "lib" {
		libFlag = " --lib"
	}
	debugFlag := ""
	if d.config.Debug {
		debugFlag = " --debug"
	}
	mutantBin, err := resolveDockerMutantBinary()
	if err != nil {
		return err
	}
	// When debug is on, also pass -v (top-level global) so malefic-mutant
	// emits its own verbose diagnostics. The flag must sit between the
	// binary path and the first subcommand (generate / build / tool).
	mutantInvoke := mutantBin
	if d.config.Debug {
		mutantInvoke = mutantBin + " -v"
	}

	// 资源合并前缀命令：先合并 builtin 和 custom resources 到目标目录
	resourceMergePrefix := "mkdir -p /root/src/resources && " +
		"[ -d /tmp/builtin/resources ] && cp -rf /tmp/builtin/resources/. /root/src/resources/ || true && " +
		"[ -d /tmp/custom/resources ] && cp -rf /tmp/custom/resources/. /root/src/resources/ || true && "

	switch d.config.BuildType {
	case consts.CommandBuildBeacon:
		buildCommand = fmt.Sprintf(
			"%s%s generate beacon && %s build%s%s malefic -t %s",
			resourceMergePrefix,
			mutantInvoke,
			mutantInvoke,
			libFlag,
			debugFlag,
			d.config.Target,
		)
	case consts.CommandBuildBind:
		buildCommand = fmt.Sprintf(
			"%s%s generate bind && %s build%s%s malefic -t %s",
			resourceMergePrefix,
			mutantInvoke,
			mutantInvoke,
			libFlag,
			debugFlag,
			d.config.Target,
		)
	case consts.CommandBuildModules:
		buildCommand = fmt.Sprintf(
			"%s%s generate modules -m %s && %s build%s%s modules -m %s -t %s",
			resourceMergePrefix,
			mutantInvoke,
			strings.Join(profile.Implant.Modules, ","),
			mutantInvoke,
			libFlag,
			debugFlag,
			strings.Join(profile.Implant.Modules, ","),
			d.config.Target,
		)
		d.enable3rd = false
	case consts.CommandBuild3rdModules:
		buildCommand = fmt.Sprintf(
			"%s%s generate modules && %s build%s%s 3rd -m %s -t %s",
			resourceMergePrefix,
			mutantInvoke,
			mutantInvoke,
			libFlag,
			debugFlag,
			strings.Join(profile.Implant.ThirdModules, ","),
			d.config.Target,
		)
		d.enable3rd = true
	case consts.CommandBuildPrelude:
		buildCommand = fmt.Sprintf(
			"%s%s generate prelude prelude.yaml && %s build%s prelude -t %s",
			resourceMergePrefix,
			mutantInvoke,
			mutantInvoke,
			debugFlag,
			d.config.Target,
		)
	case consts.CommandBuildPulse:
		target, ok := consts.GetBuildTarget(d.config.Target)
		if !ok {
			return errs.ErrInvalidateTarget
		}
		var pulseOs string
		if target.OS == consts.Windows {
			pulseOs = "win"
		} else {
			pulseOs = target.OS
		}
		shellcodeFlag := ""
		if d.config.OutputType == "shellcode" {
			shellcodeFlag = " --shellcode"
		}
		buildCommand = fmt.Sprintf(
			"%s%s generate pulse -a %s -p %s && %s build%s%s pulse%s -t %s",
			resourceMergePrefix,
			mutantInvoke,
			target.Arch, pulseOs,
			mutantInvoke,
			libFlag, debugFlag, shellcodeFlag, d.config.Target,
		)
	case consts.CommandBuildProxyDll:
		if profile.Loader != nil && profile.Loader.ProxyDll != nil {
			d.packResources = profile.Loader.ProxyDll.PackResources
		}
		buildCommand = fmt.Sprintf(
			"%s%s generate loader proxydll && %s build%s proxy-dll -t %s",
			resourceMergePrefix,
			mutantInvoke,
			mutantInvoke,
			debugFlag,
			d.config.Target,
		)
	}
	d.containerName = "malefic_" + cryptography.RandomString(8)
	// 1. 创建容器
	resp, err := cli.ContainerCreate(ctx, &container.Config{
		Image: GetImage(d.config.Target),
		Cmd:   []string{"bash", "-c", buildCommand},
	}, &container.HostConfig{
		// AutoRemove is intentionally disabled so we can (a) guarantee
		// the log stream has fully drained before the container is gone,
		// and (b) keep the container around for inspection when Debug is
		// set. Removal is handled in the deferred cleanup below.
		AutoRemove: false,
		Binds:      d.volumes,
	}, nil, nil, d.containerName)

	if err != nil {
		db.UpdateBuilderStatus(d.artifact.ID, consts.BuildStatusFailure)
		return fmt.Errorf("failed to create container: %w", err)
	}
	d.containerID = resp.ID

	// 2. Start the container.
	if err := cli.ContainerStart(ctx, resp.ID, container.StartOptions{}); err != nil {
		db.UpdateBuilderStatus(d.artifact.ID, consts.BuildStatusFailure)
		// Container was created but never started; remove it best-effort.
		_ = cli.ContainerRemove(context.Background(), resp.ID, container.RemoveOptions{Force: true})
		return fmt.Errorf("failed to start container: %w", err)
	}

	// Mark it running only after the container starts successfully.
	db.UpdateBuilderStatus(d.artifact.ID, consts.BuildStatusRunning)
	logs.Log.Infof("Container %s started successfully.", resp.ID)

	// 3. Capture logs asynchronously; logsDone lets cleanup wait for the
	// stream to drain after the container exits.
	logsDone := make(chan struct{})
	core.GoGuarded("docker-catch-logs:"+d.config.BuildName, func() error {
		defer close(logsDone)
		if err := catchLogs(cli, resp.ID, d.config.BuildName); err != nil {
			logs.Log.Errorf("Error catching logs: %v", err)
		}
		return nil
	}, core.LogGuardedError("docker-catch-logs:"+d.config.BuildName))

	// Container cleanup waits for the log stream first, so completed builds
	// are not reported before the final Docker output is stored.
	defer func() {
		select {
		case <-logsDone:
		case <-time.After(dockerLogDrainTimeout):
			logs.Log.Warnf("log drain timeout for container %s; continuing cleanup", d.containerName)
		}
		if d.config.Debug {
			logs.Log.Infof("debug mode: keeping container %s (%s) for inspection", d.containerName, resp.ID)
			return
		}
		if rmErr := cli.ContainerRemove(context.Background(), resp.ID, container.RemoveOptions{Force: true}); rmErr != nil {
			logs.Log.Warnf("failed to remove container %s: %v", resp.ID, rmErr)
		}
	}()

	// 4. Wait for the container to exit and check its status.
	statusCh, errCh := cli.ContainerWait(ctx, resp.ID, container.WaitConditionNotRunning)

	select {
	case err := <-errCh:
		if err != nil && !strings.Contains(err.Error(), "No such container") {
			db.UpdateBuilderStatus(d.artifact.ID, consts.BuildStatusFailure)
			return err
		}
	case status := <-statusCh:
		if status.StatusCode != 0 {
			logs.Log.Errorf("Container exited with non-zero status: %d", status.StatusCode)
			db.UpdateBuilderStatus(d.artifact.ID, consts.BuildStatusFailure)
			return fmt.Errorf("container exited with code %d", status.StatusCode)
		}

		// Treat only exit code 0 as success.
		db.UpdateBuilderStatus(d.artifact.ID, consts.BuildStatusCompleted)
		logs.Log.Infof("Container %s finished successfully.", resp.ID)
	}

	return nil
}

func (d *DockerBuilder) Collect() (string, string, error) {
	_, artifactPath, err := MoveBuildOutput(d.config.Target, d.config.BuildType, d.enable3rd, d.config.OutputType, d.packResources, d.config.Debug)
	if err != nil {
		logs.Log.Errorf("failed to move artifact %s output: %s", d.artifact.Name, err)
		return "", consts.BuildStatusFailure, err
	}
	defer fileutils.ForceRemoveAll(d.srcPath)
	absArtifactPath, err := filepath.Abs(artifactPath)
	if err != nil {
		logs.Log.Errorf("failed to find artifactPath: %s", err)
		return "", consts.BuildStatusFailure, err
	}

	d.artifact.Path = absArtifactPath
	err = db.UpdateBuilderPath(d.artifact)
	if err != nil {
		logs.Log.Errorf("failed to update artifactPath: %s", err)
		return "", consts.BuildStatusFailure, err
	}

	_, err = os.ReadFile(absArtifactPath)
	if err != nil {
		logs.Log.Errorf("failed to read artifact file: %s", err)
		return "", consts.BuildStatusFailure, err
	}
	db.UpdateBuilderStatus(d.artifact.ID, consts.BuildStatusCompleted)
	//if d.config.BuildType == consts.CommandBuildBeacon {
	//	if d.config.ArtifactId != 0 {
	//		err = db.UpdatePulseRelink(d.config.ArtifactId, d.artifact.ID)
	//		if err != nil {
	//			logs.Log.Errorf("failed to update pulse relink: %s", err)
	//		}
	//	}
	//}
	return d.artifact.Path, consts.BuildStatusCompleted, nil
}

func GetContainerID(d *DockerBuilder) string {
	return d.containerID
}

func SetLicenseID(d *DockerBuilder, licenseID string) {
	d.licenseID = licenseID
}

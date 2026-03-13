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
	"github.com/docker/docker/api/types"
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
	if d.config.ProfileName != "" && d.config.MaleficConfig == nil {
		implant, prelude, resources, pErr := db.GetProfileFullConfig(d.config.ProfileName)
		if pErr != nil {
			return nil, fmt.Errorf("failed to get profile config: %s", pErr)
		}
		d.config.MaleficConfig = implant
		if d.config.PreludeConfig == nil {
			d.config.PreludeConfig = prelude
		}
		if d.config.Resources == nil {
			d.config.Resources = resources
		}
	}
	_, err = selfType.LoadProfile(d.config.MaleficConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to get config: %s", err)
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
	d.volumes = Volumes

	// 挂载内置 resources（镜像预留资源）
	builtinResourceVolume := fmt.Sprintf("%s:%s", filepath.ToSlash(resourcePath), ContainerBuiltinResourcePath)
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

	// 资源合并前缀命令：先合并 builtin 和 custom resources 到目标目录
	resourceMergePrefix := "mkdir -p /root/src/resources && " +
		"[ -d /tmp/builtin/resources ] && cp -rf /tmp/builtin/resources/. /root/src/resources/ || true && " +
		"[ -d /tmp/custom/resources ] && cp -rf /tmp/custom/resources/. /root/src/resources/ || true && "

	switch d.config.BuildType {
	case consts.CommandBuildBeacon:
		buildCommand = fmt.Sprintf(
			"%smalefic-mutant generate beacon && malefic-mutant build%s malefic -t %s",
			resourceMergePrefix,
			libFlag,
			d.config.Target,
		)
	case consts.CommandBuildBind:
		buildCommand = fmt.Sprintf(
			"%smalefic-mutant generate bind && malefic-mutant build%s malefic -t %s",
			resourceMergePrefix,
			libFlag,
			d.config.Target,
		)
	case consts.CommandBuildModules:
		buildCommand = fmt.Sprintf(
			"%smalefic-mutant generate modules -m %s && malefic-mutant build%s modules -m %s -t %s",
			resourceMergePrefix,
			strings.Join(profile.Implant.Modules, ","),
			libFlag,
			strings.Join(profile.Implant.Modules, ","),
			d.config.Target,
		)
		d.enable3rd = false
	case consts.CommandBuild3rdModules:
		buildCommand = fmt.Sprintf(
			"%smalefic-mutant generate modules && malefic-mutant build%s 3rd -m %s -t %s",
			resourceMergePrefix,
			libFlag,
			strings.Join(profile.Implant.ThirdModules, ","),
			d.config.Target,
		)
		d.enable3rd = true
	case consts.CommandBuildPrelude:
		buildCommand = fmt.Sprintf(
			"%smalefic-mutant generate prelude prelude.yaml && malefic-mutant build prelude -t %s",
			resourceMergePrefix,
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
			"%smalefic-mutant generate pulse -a %s -p %s && malefic-mutant build%s pulse%s -t %s",
			resourceMergePrefix,
			target.Arch, pulseOs, libFlag, shellcodeFlag, d.config.Target,
		)
	}
	d.containerName = "malefic_" + cryptography.RandomString(8)
	// 1. 创建容器
	resp, err := cli.ContainerCreate(ctx, &container.Config{
		Image: GetImage(d.config.Target),
		Cmd:   []string{"bash", "-c", buildCommand},
	}, &container.HostConfig{
		AutoRemove: true,
		Binds:      d.volumes,
	}, nil, nil, d.containerName)

	if err != nil {
		db.UpdateBuilderStatus(d.artifact.ID, consts.BuildStatusFailure)
		return fmt.Errorf("failed to create container: %w", err)
	}
	d.containerID = resp.ID

	// 2. 启动容器
	if err := cli.ContainerStart(ctx, resp.ID, types.ContainerStartOptions{}); err != nil {
		db.UpdateBuilderStatus(d.artifact.ID, consts.BuildStatusFailure)
		return fmt.Errorf("failed to start container: %w", err)
	}

	// 只有启动成功后，才标记为 Running
	db.UpdateBuilderStatus(d.artifact.ID, consts.BuildStatusRunning)
	logs.Log.Infof("Container %s started successfully.", resp.ID)

	// 3. 异步捕获日志
	core.GoGuarded("docker-catch-logs:"+d.config.BuildName, func() error {
		if err := catchLogs(cli, resp.ID, d.config.BuildName); err != nil {
			logs.Log.Errorf("Error catching logs: %v", err)
		}
		return nil
	}, core.LogGuardedError("docker-catch-logs:"+d.config.BuildName))

	// 4. 等待容器结束并检查退出状态
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

		// 只有 ExitCode 为 0 才视为成功
		db.UpdateBuilderStatus(d.artifact.ID, consts.BuildStatusCompleted)
		logs.Log.Infof("Container %s finished successfully.", resp.ID)
	}

	return nil
}

func (d *DockerBuilder) Collect() (string, string, error) {
	_, artifactPath, err := MoveBuildOutput(d.config.Target, d.config.BuildType, d.enable3rd, d.config.OutputType)
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

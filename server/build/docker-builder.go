package build

import (
	"context"
	"fmt"
	"github.com/chainreactors/logs"
	"github.com/chainreactors/malice-network/helper/codenames"
	"github.com/chainreactors/malice-network/helper/consts"
	"github.com/chainreactors/malice-network/helper/cryptography"
	"github.com/chainreactors/malice-network/helper/encoders"
	"github.com/chainreactors/malice-network/helper/errs"
	profile2 "github.com/chainreactors/malice-network/helper/profile"
	"github.com/chainreactors/malice-network/helper/proto/client/clientpb"
	selfType "github.com/chainreactors/malice-network/helper/types"
	"github.com/chainreactors/malice-network/helper/utils/fileutils"
	"github.com/chainreactors/malice-network/server/internal/configs"
	"github.com/chainreactors/malice-network/server/internal/db"
	"github.com/chainreactors/malice-network/server/internal/db/models"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"os"
	"path/filepath"
	"strings"
	"time"
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
	var profileByte []byte
	var profile *selfType.ProfileConfig
	if d.config.BuildName == "" {
		d.config.BuildName = codenames.GetCodename()
	}
	// get profile
	if d.config.ProfileName != "" {
		profileByte, err = db.GetProfileContent(d.config.ProfileName)
		d.config.MaleficConfig = profileByte
	}
	profile, err = selfType.LoadProfile(d.config.MaleficConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to get config: %s", err)
	}
	logs.Log.Debugf("profile: %v", profile)

	// init artifact status
	artifactId := d.config.ArtifactId
	// save artifact and update status
	if artifactId != 0 && d.config.BuildType == consts.CommandBuildBeacon {
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
	err = profile2.WriteBuildConfigToPath(d.config, d.srcPath)
	if err != nil {
		return nil, err
	}
	// set volume - 精确挂载特定文件和目录
	d.volumes = Volumes

	// 挂载 config.yaml（如果存在）
	configPath := filepath.Join(d.srcPath, "config.yaml")
	if fileutils.Exist(configPath) {
		configVolume := fmt.Sprintf("%s:%s", filepath.ToSlash(configPath), ContainerConfigPath)
		d.volumes = append(d.volumes, configVolume)
	}

	// 挂载 autorun.yaml（必须存在）
	autorunPath := filepath.Join(d.srcPath, "autorun.yaml")
	if fileutils.Exist(autorunPath) {
		autoRunVolume := fmt.Sprintf("%s:%s", filepath.ToSlash(autorunPath), ContainerAutoRunPath)
		d.volumes = append(d.volumes, autoRunVolume)
	}

	// 挂载 resources 目录（如果存在）
	resourcesPath := filepath.Join(d.srcPath, "resources")
	if fileutils.Exist(resourcesPath) {
		resourceVolume := fmt.Sprintf("%s:%s", filepath.ToSlash(resourcesPath), ContainerResourcePath)
		d.volumes = append(d.volumes, resourceVolume)
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
	switch d.config.BuildType {
	case consts.CommandBuildBeacon:
		buildCommand = fmt.Sprintf(
			"malefic-mutant generate beacon && malefic-mutant build malefic -t %s",
			d.config.Target,
		)
	case consts.CommandBuildBind:
		buildCommand = fmt.Sprintf(
			"malefic-mutant generate bind && malefic-mutant build malefic -t %s",
			d.config.Target,
		)
	case consts.CommandBuildModules:
		buildCommand = fmt.Sprintf(
			"malefic-mutant generate modules -m %s && malefic-mutant build modules -m %s -t %s",
			strings.Join(profile.Implant.Modules, ","),
			strings.Join(profile.Implant.Modules, ","),
			d.config.Target,
		)
		d.enable3rd = false
	case consts.CommandBuild3rdModules:
		buildCommand = fmt.Sprintf(
			"malefic-mutant generate modules && malefic-mutant build 3rd -m %s -t %s",
			strings.Join(profile.Implant.ThirdModules, ","),
			d.config.Target,
		)
		d.enable3rd = true
	case consts.CommandBuildPrelude:
		buildCommand = fmt.Sprintf(
			"malefic-mutant generate prelude autorun.yaml && malefic-mutant build prelude -t %s",
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
		buildCommand = fmt.Sprintf(
			"malefic-mutant generate pulse -a %s -p %s && malefic-mutant build pulse -t %s",
			target.Arch, pulseOs, d.config.Target,
		)
	}
	d.containerName = "malefic_" + cryptography.RandomString(8)
	resp, err := cli.ContainerCreate(ctx, &container.Config{
		Env: []string{
			//"PATH=" + PATH_ENV,
		},
		Image: GetImage(d.config.Target),
		Cmd:   []string{"bash", "-c", buildCommand},
	}, &container.HostConfig{
		AutoRemove: true,
		Binds:      d.volumes,
	}, nil, nil, d.containerName)
	if err != nil {
		return err
	}
	d.containerID = resp.ID
	if err := cli.ContainerStart(ctx, resp.ID, types.ContainerStartOptions{}); err != nil {
		db.UpdateBuilderStatus(d.artifact.ID, consts.BuildStatusFailure)
	}
	db.UpdateBuilderStatus(d.artifact.ID, consts.BuildStatusRunning)
	logs.Log.Infof("Container %s started successfully.", resp.ID)

	err = catchLogs(cli, resp.ID, d.config.BuildName)
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
			db.UpdateBuilderStatus(d.artifact.ID, consts.BuildStatusFailure)
			return err
		}
	case <-statusCh:
		logs.Log.Infof("Container %s has stopped and will be automatically removed.", resp.ID)
	}
	return nil
}

func (d *DockerBuilder) Collect() (string, string, error) {
	_, artifactPath, err := MoveBuildOutput(d.config.Target, d.config.BuildType, d.enable3rd)
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

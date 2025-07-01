package build

import (
	"context"
	"fmt"
	"github.com/chainreactors/logs"
	"github.com/chainreactors/malice-network/helper/codenames"
	"github.com/chainreactors/malice-network/helper/consts"
	"github.com/chainreactors/malice-network/helper/cryptography"
	"github.com/chainreactors/malice-network/helper/errs"
	"github.com/chainreactors/malice-network/helper/proto/client/clientpb"
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
	builder       *models.Builder
	containerName string
	containerID   string
}

func NewDockerBuilder(req *clientpb.BuildConfig) *DockerBuilder {
	return &DockerBuilder{
		config: req,
	}
}

func (d *DockerBuilder) GenerateConfig() (*clientpb.Builder, error) {
	var builder *models.Builder
	var err error
	if d.config.ArtifactId != 0 && d.config.Type == consts.CommandBuildBeacon {
		builder, err = db.SaveArtifactFromID(d.config, d.config.ArtifactId, d.config.Source)
	} else {
		if d.config.BuildName == "" {
			d.config.BuildName = codenames.GetCodename()
		}
		builder, err = db.SaveArtifactFromConfig(d.config)
	}
	if err != nil {
		logs.Log.Errorf("failed to save build %s: %s", builder.Name, err)
		return nil, err
	}
	d.builder = builder
	db.UpdateBuilderStatus(d.builder.ID, consts.BuildStatusWaiting)
	return builder.ToProtobuf(), nil
}

func (d *DockerBuilder) ExecuteBuild() error {
	dockerBuildSemaphore <- struct{}{}
	defer func() { <-dockerBuildSemaphore }()
	timeout := 20 * time.Minute
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	if d.config.Inputs == nil {
		_, err := GenerateProfile(d.config)
		if err != nil {
			return fmt.Errorf("failed to create config: %s", err)
		}
	}
	cli, err := GetDockerClient()
	if err != nil {
		return err
	}
	var buildCommand string
	switch d.config.Type {
	case consts.CommandBuildBeacon:
		buildCommand = fmt.Sprintf(
			"malefic-mutant generate beacon;malefic-mutant build malefic -t %s",
			d.config.Target,
		)
	case consts.CommandBuildBind:
		buildCommand = fmt.Sprintf(
			"malefic-mutant generate bind && malefic-mutant build malefic -t %s",
			d.config.Target,
		)
	case consts.CommandBuildModules:
		buildCommand = fmt.Sprintf(
			"malefic-mutant generate modules && malefic-mutant build modules -t %s",
			d.config.Target,
		)
	case consts.CommandBuildPrelude:
		buildCommand = fmt.Sprintf(
			"malefic-mutant generate prelude && malefic-mutant build prelude -t %s",
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
			"malefic-mutant generate pulse %s %s &&malefic-mutant build pulse -t %s",
			target.Arch, pulseOs, d.config.Target,
		)
	}
	d.containerName = "malefic_" + cryptography.RandomString(8)
	resp, err := cli.ContainerCreate(ctx, &container.Config{
		Env: []string{
			//"PATH=" + PATH_ENV,
		},
		Image: DefaultImage,
		Cmd:   []string{"bash", "-c", buildCommand},
	}, &container.HostConfig{
		AutoRemove: true,
		Binds:      Volumes,
	}, nil, nil, d.containerName)
	if err != nil {
		return err
	}
	d.containerID = resp.ID
	if err := cli.ContainerStart(ctx, resp.ID, types.ContainerStartOptions{}); err != nil {
		db.UpdateBuilderStatus(d.builder.ID, consts.BuildStatusFailure)
		SendBuildMsg(d.builder, consts.BuildStatusFailure, "")
	}
	db.UpdateBuilderStatus(d.builder.ID, consts.BuildStatusRunning)
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
			db.UpdateBuilderStatus(d.builder.ID, consts.BuildStatusFailure)
			SendBuildMsg(d.builder, consts.BuildStatusFailure, "")
			return err
		}
	case <-statusCh:
		logs.Log.Infof("Container %s has stopped and will be automatically removed.", resp.ID)
	}
	return nil
}

func (d *DockerBuilder) CollectArtifact() (string, string) {
	_, artifactPath, err := MoveBuildOutput(d.config.Target, d.config.Type)
	if err != nil {
		logs.Log.Errorf("failed to move builder %s output: %s", d.builder.Name, err)
		db.UpdateBuilderStatus(d.builder.ID, consts.BuildStatusFailure)
		return "", consts.BuildStatusFailure
	}
	err = db.UpdateBuilderPath(d.builder)
	if err != nil {
		logs.Log.Errorf("failed to update Builder %s path: %s", d.builder.Name, err)
	}
	absArtifactPath, err := filepath.Abs(artifactPath)
	if err != nil {
		logs.Log.Errorf("failed to find artifactPath: %s", err)
		SendBuildMsg(d.builder, consts.BuildStatusFailure, "")
		return "", consts.BuildStatusFailure
	}

	d.builder.Path = absArtifactPath
	err = db.UpdateBuilderPath(d.builder)
	if err != nil {
		SendBuildMsg(d.builder, consts.BuildStatusFailure, "")
		logs.Log.Errorf("failed to update %s path: %s", d.builder.Name, err)
		return "", consts.BuildStatusFailure
	}

	_, err = os.ReadFile(absArtifactPath)
	if err != nil {
		SendBuildMsg(d.builder, consts.BuildStatusFailure, "")
		logs.Log.Errorf("failed to read artifact file: %s", err)
		return "", consts.BuildStatusFailure
	}
	if d.builder.IsSRDI {
		target, ok := consts.GetBuildTarget(d.config.Target)
		if !ok {
			logs.Log.Errorf("builder %s(%s %s): %s", d.builder.Name, d.builder.Type, d.builder.Target, errs.ErrInvalidateTarget)
			return "", consts.BuildStatusFailure
		}
		if d.builder.Type == consts.CommandBuildPulse {
			logs.Log.Infof("objcopy start ...")
			_, err = OBJCOPYPulse(d.builder, target.OS, target.Arch)
			if err != nil {
				logs.Log.Errorf("failed to objcopy: %s", err)
				db.UpdateBuilderStatus(d.builder.ID, consts.BuildStatusCompleted)
			}
			logs.Log.Infof("objcopy end ...")
		} else {
			_, err = SRDIArtifact(d.builder, target.OS, target.Arch)
			if err != nil {
				logs.Log.Errorf("failed to srid: %s", err)
				db.UpdateBuilderStatus(d.builder.ID, consts.BuildStatusCompleted)
				return "", consts.BuildStatusCompleted
			}
		}
	}
	db.UpdateBuilderStatus(d.builder.ID, consts.BuildStatusCompleted)
	SendBuildMsg(d.builder, consts.BuildStatusCompleted, "")
	return d.builder.Path, consts.BuildStatusCompleted
}

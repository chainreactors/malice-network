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
		builder, err = db.SaveArtifactFromID(d.config, d.config.ArtifactId, d.config.Resource)
	} else {
		if d.config.BuildName == "" {
			d.config.BuildName = codenames.GetCodename()
		}
		builder, err = db.SaveArtifactFromConfig(d.config)
	}
	if err != nil {
		logs.Log.Errorf("save build db error: %v", err)
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
			logs.Log.Errorf("failed to create config: %v", err)
			return fmt.Errorf("failed to create config: %v", err)
		}
	}
	cli, err := GetDockerClient()
	if err != nil {
		logs.Log.Errorf("docker client failed %s", err)
		return err
	}
	var buildCommand string
	switch d.config.Type {
	case consts.CommandBuildBeacon:
		buildCommand = fmt.Sprintf(
			"malefic-mutant generate -s beacon;malefic-mutant build malefic -t %s",
			d.config.Target,
		)
	case consts.CommandBuildBind:
		buildCommand = fmt.Sprintf(
			"malefic-mutant generate -s bind && malefic-mutant build malefic -t %s",
			d.config.Target,
		)
	case consts.CommandBuildModules:
		buildCommand = fmt.Sprintf(
			"malefic-mutant generate -s modules && malefic-mutant build modules -t %s",
			d.config.Target,
		)
	case consts.CommandBuildPrelude:
		buildCommand = fmt.Sprintf(
			"malefic-mutant generate -s prelude && malefic-mutant build prelude -t %s",
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
			"malefic-mutant generate -s pulse %s %s &&malefic-mutant build pulse -t %s",
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
		logs.Log.Errorf("docker start failed %s", err)
		return err
	}
	d.containerID = resp.ID
	if err := cli.ContainerStart(ctx, resp.ID, types.ContainerStartOptions{}); err != nil {
		logs.Log.Errorf("Error starting container: %v", err)
		db.UpdateBuilderStatus(d.builder.ID, consts.BuildStatusFailure)
	}
	db.UpdateBuilderStatus(d.builder.ID, consts.BuildStatusRunning)
	sendContainerCtrlMsg(false, d.containerName, d.config, nil)
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
			return err
		}
	case <-statusCh:
		logs.Log.Infof("Container %s has stopped and will be automatically removed.", resp.ID)
	}
	return nil
}

func (d *DockerBuilder) CollectArtifact() {
	_, artifactPath, err := MoveBuildOutput(d.config.Target, d.config.Type)
	if err != nil {
		logs.Log.Errorf("move build output error: %v", err)
		sendContainerCtrlMsg(true, d.containerName, d.config, fmt.Errorf("db err %v", err))
		db.UpdateBuilderStatus(d.builder.ID, consts.BuildStatusFailure)
		return
	}
	err = db.UpdateBuilderPath(d.builder)
	if err != nil {
		sendContainerCtrlMsg(true, d.containerName, d.config, fmt.Errorf("db err %v", err))
		logs.Log.Errorf("update builder path and status error: %v", err)
	}
	absArtifactPath, err := filepath.Abs(artifactPath)
	if err != nil {
		sendContainerCtrlMsg(true, d.containerName, d.config, fmt.Errorf("artifactPath err %v", err))
		return
	}

	d.builder.Path = absArtifactPath
	err = db.UpdateBuilderPath(d.builder)
	if err != nil {
		sendContainerCtrlMsg(true, d.containerName, d.config, fmt.Errorf("db err %v", err))
		return
	}

	_, err = os.ReadFile(absArtifactPath)
	if err != nil {
		sendContainerCtrlMsg(true, d.containerName, d.config, fmt.Errorf("read artifactFile err %v", err))
		return
	}
	if d.builder.IsSRDI {
		target, ok := consts.GetBuildTarget(d.config.Target)
		if !ok {
			sendContainerCtrlMsg(true, d.containerName, d.config, errs.ErrInvalidateTarget)
			return
		}
		if d.builder.Type == consts.CommandBuildPulse {
			logs.Log.Infof("objcopy start ...")
			_, err = OBJCOPYPulse(d.builder, target.OS, target.Arch)
			if err != nil {
				sendContainerCtrlMsg(true, d.containerName, d.config, fmt.Errorf("objcopy error: %v", err))
			}
			logs.Log.Infof("objcopy end ...")
		} else {
			_, err = SRDIArtifact(d.builder, target.OS, target.Arch)
			if err != nil {
				sendContainerCtrlMsg(true, d.containerName, d.config, fmt.Errorf("SRDI error %v", err))
				return
			}
		}
	}
	db.UpdateBuilderStatus(d.builder.ID, consts.BuildStatusCompleted)
	sendContainerCtrlMsg(true, d.containerName, d.config, nil)
	return
}

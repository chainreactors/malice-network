package build

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/chainreactors/logs"
	"github.com/chainreactors/malice-network/helper/codenames"
	"github.com/chainreactors/malice-network/helper/consts"
	"github.com/chainreactors/malice-network/helper/cryptography"
	"github.com/chainreactors/malice-network/helper/errs"
	"github.com/chainreactors/malice-network/helper/proto/client/clientpb"
	selfType "github.com/chainreactors/malice-network/helper/types"
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
}

func NewDockerBuilder(req *clientpb.BuildConfig) *DockerBuilder {
	return &DockerBuilder{
		config: req,
	}
}

func (d *DockerBuilder) Generate() (*clientpb.Artifact, error) {
	var builder *models.Artifact
	var err error
	var profileByte []byte
	if d.config.Inputs == nil {
		profileByte, err = GenerateProfile(d.config)
		if err != nil {
			return nil, fmt.Errorf("failed to create config: %s", err)
		}
	}
	if d.config.ArtifactId != 0 && d.config.Type == consts.CommandBuildBeacon {
		builder, err = db.SaveArtifactFromID(d.config, d.config.ArtifactId, d.config.Source, profileByte)
	} else {
		if d.config.BuildName == "" {
			d.config.BuildName = codenames.GetCodename()
		}
		builder, err = db.SaveArtifactFromConfig(d.config, profileByte)
	}
	if err != nil {
		logs.Log.Errorf("failed to save build %s: %s", builder.Name, err)
		return nil, err
	}
	d.artifact = builder
	db.UpdateBuilderStatus(d.artifact.ID, consts.BuildStatusWaiting)
	return builder.ToProtobuf([]byte{}), nil
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
	switch d.config.Type {
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
		var profileParams selfType.ProfileParams
		err = json.Unmarshal(d.config.ParamsBytes, &profileParams)
		if err != nil {
			return err
		}
		if profileParams.Enable3RD {
			buildCommand = fmt.Sprintf(
				"malefic-mutant generate modules && malefic-mutant build 3rd -m %s -t %s",
				profileParams.Modules,
				d.config.Target,
			)
			d.enable3rd = true
		} else {
			buildCommand = fmt.Sprintf(
				"malefic-mutant generate modules -m %s && malefic-mutant build modules -m %s -t %s",
				profileParams.Modules,
				profileParams.Modules,
				d.config.Target,
			)
			d.enable3rd = false
		}
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
		db.UpdateBuilderStatus(d.artifact.ID, consts.BuildStatusFailure)
		SendBuildMsg(d.artifact, consts.BuildStatusFailure, "")
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
			SendBuildMsg(d.artifact, consts.BuildStatusFailure, "")
			return err
		}
	case <-statusCh:
		logs.Log.Infof("Container %s has stopped and will be automatically removed.", resp.ID)
	}
	return nil
}

func (d *DockerBuilder) Collect() (string, string) {
	_, artifactPath, err := MoveBuildOutput(d.config.Target, d.config.Type, d.enable3rd)
	if err != nil {
		logs.Log.Errorf("failed to move artifact %s output: %s", d.artifact.Name, err)
		db.UpdateBuilderStatus(d.artifact.ID, consts.BuildStatusFailure)
		return "", consts.BuildStatusFailure
	}

	absArtifactPath, err := filepath.Abs(artifactPath)
	if err != nil {
		logs.Log.Errorf("failed to find artifactPath: %s", err)
		SendBuildMsg(d.artifact, consts.BuildStatusFailure, "")
		return "", consts.BuildStatusFailure
	}

	d.artifact.Path = absArtifactPath
	err = db.UpdateBuilderPath(d.artifact)
	if err != nil {
		SendBuildMsg(d.artifact, consts.BuildStatusFailure, "")
		logs.Log.Errorf("failed to update %s path: %s", d.artifact.Name, err)
		return "", consts.BuildStatusFailure
	}

	_, err = os.ReadFile(absArtifactPath)
	if err != nil {
		SendBuildMsg(d.artifact, consts.BuildStatusFailure, "")
		logs.Log.Errorf("failed to read artifact file: %s", err)
		return "", consts.BuildStatusFailure
	}
	db.UpdateBuilderStatus(d.artifact.ID, consts.BuildStatusCompleted)
	SendBuildMsg(d.artifact, consts.BuildStatusCompleted, "")
	if d.config.Type == consts.CommandBuildBeacon {
		if d.config.ArtifactId != 0 {
			err = db.UpdatePulseRelink(d.config.ArtifactId, d.artifact.ID)
			if err != nil {
				logs.Log.Errorf("failed to update pulse relink: %s", err)
			}
		}
	}
	err = SendAddContent(d.artifact.Name)
	if err != nil {
		logs.Log.Errorf("failed to add artifact path to website: %s", err)
	}
	return d.artifact.Path, consts.BuildStatusCompleted
}

//func (d *DockerBuilder) GetBeaconID() uint32 {
//	return d.config.ArtifactId
//}
//
//func (d *DockerBuilder) SetBeaconID(id uint32) error {
//	d.config.ArtifactId = id
//	if d.config.Params == "" {
//		params := &configType.ProfileParams{
//			OriginBeaconID: id,
//		}
//		d.config.Params = params.String()
//	} else {
//		var newParams *configType.ProfileParams
//		err := json.Unmarshal([]byte(d.config.Params), &newParams)
//		if err != nil {
//			return err
//		}
//		newParams.OriginBeaconID = d.config.ArtifactId
//		d.config.Params = newParams.String()
//	}
//	return nil
//}

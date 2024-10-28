package rpc

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"github.com/chainreactors/logs"
	"github.com/chainreactors/malice-network/helper/consts"
	"github.com/chainreactors/malice-network/helper/proto/client/clientpb"
	"github.com/chainreactors/malice-network/server/internal/build"
	"github.com/chainreactors/malice-network/server/internal/configs"
	"github.com/chainreactors/malice-network/server/internal/db"
	"github.com/docker/docker/client"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
)

var dockerClient *client.Client
var once sync.Once
var maleficConfig = "malefic_config"
var community = "community"
var prebuild = "prebuild"

func randomString(length int) string {
	b := make([]byte, length)
	_, _ = rand.Read(b)
	return base64.URLEncoding.EncodeToString(b)
}

func setEnv() []string {
	environs := make([]string, 0)
	// rustflag
	rustflag := "RUSTFLAGS=-A warnings "
	rsFiles, _ := findRSFiles(configs.BuildPath)
	for _, rsFile := range rsFiles {
		rustflag = rustflag + fmt.Sprintf("--remap-path-prefix=%s=%s.rs ", rsFile, randomString(12))
	}
	// add to environs
	environs = append(environs, rustflag)
	return environs
}

func findRSFiles(root string) ([]string, error) {
	var rsFiles []string
	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() && strings.HasSuffix(info.Name(), ".rs") {
			rsFiles = append(rsFiles, path)
		}
		return nil
	})
	return rsFiles, err
}

func getDockerClient() (*client.Client, error) {
	var err error
	once.Do(func() {
		dockerClient, err = client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
		if err != nil {
			logs.Log.Errorf("Error creating Docker client: %v", err)
		}
	})
	return dockerClient, err
}

func (rpc *Server) Generate(ctx context.Context, req *clientpb.Generate) (*clientpb.Empty, error) {
	cli, err := getDockerClient()
	if err != nil {
		return &clientpb.Empty{}, err
	}
	err = build.DbToConfig(req)
	if err != nil {
		return nil, err
	}
	logs.Log.Infof("start to build ...", req.Target)
	// malefic-config
	buildArgs := []string{req.Stager, community, prebuild}
	switch req.Stager {
	case "prelude":
		buildArgs = []string{req.Stager, "autorun.yaml", community, prebuild}
	}
	cmd := exec.Command(filepath.Join(configs.BuildPath, maleficConfig), buildArgs[0:]...)
	// 打印即将执行的build命令
	logs.Log.Infof("building: %s %s", cmd.Path, strings.Join(cmd.Args, " "))
	_, err = cmd.CombinedOutput()
	if err != nil {
		logs.Log.Errorf("exec failed %s", err)
	}

	// set environs for rust compiler
	environs := setEnv()
	switch req.Type {
	case consts.PE:
		err = build.BuildPE(cli, req, environs)
		if err != nil {
			return nil, err
		}
	}

	fileName, err := db.SaveBuilder(req)
	if fileName != "" {
		err = build.MoveBuildOutput(req.Target, req.Type, fileName)
	} else {
		logs.Log.Errorf("save builder error: %v, you can find build output in ./malice/build/target/%s/", err,
			req.Target)
	}
	return &clientpb.Empty{}, err
}

func (rpc *Server) GetBuilders(ctx context.Context, req *clientpb.Empty) (*clientpb.Builders, error) {
	builders, err := db.GetBuilders()
	if err != nil {
		return nil, err
	}
	return builders, nil
}

func (rpc *Server) DownloadOutput(ctx context.Context, req *clientpb.Sync) (*clientpb.SyncResp, error) {
	filePath, err := build.GetOutPutPath(req.FileId)
	if err != nil {
		return nil, err
	}
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, err
	}
	return &clientpb.SyncResp{
		Name:    filepath.Base(filePath),
		Content: data,
	}, nil
}

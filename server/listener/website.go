package listener

import (
	"bytes"
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	cryptostream "github.com/chainreactors/malice-network/server/internal/stream"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/chainreactors/logs"
	"github.com/chainreactors/malice-network/helper/consts"
	"github.com/chainreactors/malice-network/helper/proto/client/clientpb"
	"github.com/chainreactors/malice-network/helper/proto/services/listenerrpc"
	"github.com/chainreactors/malice-network/helper/utils/fileutils"
	"github.com/chainreactors/malice-network/server/internal/certutils"
	"github.com/chainreactors/malice-network/server/internal/configs"
	"github.com/chainreactors/malice-network/server/internal/core"
)

type Website struct {
	port     int
	server   *http.Server
	rpc      listenerrpc.ListenerRPCClient
	rootPath string
	Name     string
	Enable   bool
	CertName string
	*core.PipelineConfig
	Content  map[string]*clientpb.WebContent
	Artifact map[string]*clientpb.WebContent
}

func StartWebsite(rpc listenerrpc.ListenerRPCClient, pipeline *clientpb.Pipeline, content map[string]*clientpb.WebContent) (*Website, error) {
	websitePp := pipeline.GetWeb()
	web := &Website{
		Name:           pipeline.Name,
		port:           int(websitePp.Port),
		rootPath:       websitePp.Root,
		rpc:            rpc,
		CertName:       pipeline.CertName,
		PipelineConfig: core.FromPipeline(pipeline),
		Content:        make(map[string]*clientpb.WebContent),
	}
	for _, c := range content {
		err := web.AddContent(c)
		if err != nil {
			return nil, err
		}
	}
	err := web.Start()
	if err != nil {
		return nil, err
	}
	return web, nil
}

func (w *Website) ID() string {
	return w.Name
}

func (w *Website) Start() error {
	mux := http.NewServeMux()
	mux.HandleFunc(w.rootPath, w.websiteContentHandler)
	if w.Cert != nil && w.Cert.Enable {
		tlsConfig, err := certutils.WrapToTlsConfig(w.Cert)
		if err != nil {
			return err
		}
		w.server = &http.Server{Addr: fmt.Sprintf("0.0.0.0:%d", w.port),
			TLSConfig: tlsConfig,
			Handler:   mux,
		}
		go func() {
			if err = w.server.ListenAndServeTLS("", ""); err != nil && err != http.ErrServerClosed {
				logs.Log.Errorf("HTTP Server failed to start: %v", err)
			}
		}()
	} else {
		w.server = &http.Server{Addr: fmt.Sprintf("0.0.0.0:%d", w.port),
			Handler: mux,
		}
		go func() {
			if err := w.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
				logs.Log.Errorf("HTTP Server failed to start: %v", err)
			}
		}()
	}

	w.Enable = true
	return nil
}

func (w *Website) Close() error {
	if w.server != nil {
		logs.Log.Importantf("Stopping server")
		err := w.server.Shutdown(nil)
		if err != nil {
			return err
		}
		w.server = nil
		return nil
	} else {
		return errors.New("server is not running")
	}
}

func (w *Website) ToProtobuf() *clientpb.Pipeline {
	p := &clientpb.Pipeline{
		Name:       w.Name,
		Enable:     w.Enable,
		ListenerId: w.ListenerID,
		Type:       consts.WebsitePipeline,
		CertName:   w.CertName,
		Body: &clientpb.Pipeline_Web{
			Web: &clientpb.Website{
				Name:       w.Name,
				ListenerId: w.ListenerID,
				Port:       uint32(w.port),
				Root:       w.rootPath,
			},
		},
		Tls: w.Cert.ToProtobuf(),
	}
	return p
}

func (w *Website) websiteContentHandler(resp http.ResponseWriter, req *http.Request) {
	contentPath := strings.TrimPrefix(req.URL.Path, w.rootPath)
	artifactContent, ok := w.Content[strings.Trim(contentPath, "/")]
	if !ok {
		logs.Log.Debugf("%s Failed to get content in artifactContent ", req.URL)
	} else {
		key, iv := configs.GenerateKeyAndIVFromString()
		encryptor, _ := cryptostream.NewAesCtrEncryptor(key, iv)
		encrypted, err := hex.DecodeString(artifactContent.Path)
		logs.Log.Errorf("")
		decReader := bytes.NewReader(encrypted)
		decWriter := &bytes.Buffer{}

		err = encryptor.Decrypt(decReader, decWriter)
		artifactName := decWriter.Bytes()
		artifact, err := w.rpc.FindArtifact(context.Background(), &clientpb.Artifact{
			Name: string(artifactName),
		})
		if err != nil {
			return
		}
		if len(artifact.Bin) > 0 {
			resp.Write(artifact.Bin)
		}
		return
	}

	content, ok := w.Content[strings.Trim(contentPath, "/")]
	if !ok {
		logs.Log.Debugf("%s Failed to get content ", req.URL)
		return
	}

	resp.Header().Add("Content-Type", content.ContentType)
	resp.Header().Add("Cache-Control", "no-store, no-cache, must-revalidate")
	data, err := os.ReadFile(content.File)
	if err != nil {
		return
	}

	resp.Write(data)
}

func (w *Website) AddContent(content *clientpb.WebContent) error {
	contentPath := filepath.Join(configs.WebsitePath, content.WebsiteId, content.Id)
	if !fileutils.Exist(contentPath) {
		err := os.WriteFile(contentPath, content.Content, 0644)
		if err != nil {
			return err
		}
	}
	w.Content[strings.Trim(content.Path, "/")] = &clientpb.WebContent{
		Path:        content.Path,
		File:        contentPath,
		ContentType: content.ContentType,
	}
	return nil
}

func (w *Website) AddArtifactContent(content *clientpb.WebContent) error {
	contentPath := filepath.Join(configs.WebsitePath, content.WebsiteId, content.Id)
	w.Content[strings.Trim(content.Path, "/")] = &clientpb.WebContent{
		Path: content.Path,
		File: contentPath,
	}
	return nil
}

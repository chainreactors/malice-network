package listener

import (
	"context"
	"errors"
	"fmt"
	"github.com/chainreactors/logs"
	"github.com/chainreactors/malice-network/helper/consts"
	"github.com/chainreactors/malice-network/helper/cryptography"
	"github.com/chainreactors/malice-network/helper/encoders"
	"github.com/chainreactors/malice-network/helper/proto/client/clientpb"
	"github.com/chainreactors/malice-network/helper/proto/services/listenerrpc"
	"github.com/chainreactors/malice-network/helper/utils/fileutils"
	"github.com/chainreactors/malice-network/helper/utils/formatutils"
	"github.com/chainreactors/malice-network/server/internal/certutils"
	"github.com/chainreactors/malice-network/server/internal/configs"
	"github.com/chainreactors/malice-network/server/internal/core"
	"net/http"
	"os"
	"path/filepath"
	"strings"
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
		Artifact:       make(map[string]*clientpb.WebContent),
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
	mux.HandleFunc(certutils.ACMERootPath, w.acmeChallengeHandler)
	mux.HandleFunc(w.rootPath, w.websiteContentHandler)
	if w.TLSConfig != nil && w.TLSConfig.Enable && w.TLSConfig.Cert != nil {
		tlsConfig, err := certutils.GetTlsConfig(w.TLSConfig.Cert)
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
		Tls: w.TLSConfig.ToProtobuf(),
	}
	return p
}

func (w *Website) websiteContentHandler(resp http.ResponseWriter, req *http.Request) {
	contentPath := strings.TrimPrefix(req.URL.Path, w.rootPath)
	parts := strings.SplitN(contentPath, "/", 2)
	var artifactName string
	var formatted string
	artifactName = parts[0]
	if len(parts) == 2 {
		formatted = parts[1]
	}
	artifactContent, ok := w.Artifact[artifactName]
	if !ok {
		logs.Log.Debugf("%s failed to get content in artifactContent ", req.URL)
	} else {
		encrypted, err := cryptography.HexToBytes(artifactContent.Path)
		if err != nil {
			logs.Log.Errorf("failed to hex: %s", err)
			return
		}
		decrypt, err := cryptography.DecryptWithGlobalKey(encrypted)
		if err != nil {
			logs.Log.Errorf("failed to decrypt: %s", err)
			return
		}
		artifact, err := w.rpc.FindArtifact(context.Background(), &clientpb.Artifact{
			Name: string(decrypt),
		})

		if err != nil {
			logs.Log.Errorf("failed to find artifact: %s", err)
			return
		}
		if formatted == "" {
			resp.Write(artifact.Bin)
			return
		}
		encryptedFormatted, err := cryptography.HexToBytes(formatted)
		if err != nil {
			logs.Log.Errorf("failed to hex: %s", err)
			return
		}
		decryptFormatted, err := cryptography.DecryptWithGlobalKey(encryptedFormatted)
		if err != nil {
			logs.Log.Errorf("failed to decrypt: %s", err)
			return
		}
		// create a tempfile for shellcode
		filename := filepath.Join(configs.BuildOutputPath, encoders.UUID())
		if err := os.WriteFile(filename, artifact.Bin, 0644); err != nil {
			logs.Log.Errorf("failed to write file: %s", err)
			return
		}
		var usePulse bool = false
		if artifact.Type == consts.CommandBuildPulse {
			usePulse = true
		}
		shellcode, err := formatutils.SRDIArtifact(filename, artifact.Platform, artifact.Arch, usePulse)
		if err != nil {
			logs.Log.Errorf("failed to convert to shellcode: %s", err)
			return
		}
		// delete tempfile
		if err := os.Remove(filename); err != nil {
			logs.Log.Errorf("failed to remove file: %s", err)
			return
		}
		// convert artifact to format
		formatter := formatutils.NewFormatter()
		result, err := formatter.Convert(shellcode, string(decryptFormatted))
		if err != nil {
			logs.Log.Errorf("failed to convert: %s", err)
			return
		}
		if len(result.Data) > 0 {
			resp.Write(result.Data)
		}
		return
	}

	content, ok := w.Content[strings.Trim(contentPath, "/")]
	if !ok {
		logs.Log.Debugf("%s failed to get content ", req.URL)
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

// acmeChallengeHandler
func (w *Website) acmeChallengeHandler(resp http.ResponseWriter, req *http.Request) {
	if certutils.GetACMEManager() == nil {
		http.Error(resp, "ACME not enabled", http.StatusNotFound)
		return
	}
	certutils.GetACMEManager().GetManager().HTTPHandler(nil)
}

func (w *Website) AddArtifactContent(content *clientpb.WebContent) error {
	contentPath := filepath.Join(configs.WebsitePath, content.WebsiteId, content.Id)
	w.Artifact[strings.Trim(content.Path, "/")] = &clientpb.WebContent{
		Path: content.Path,
		File: contentPath,
	}
	return nil
}

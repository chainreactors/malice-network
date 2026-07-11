package rpc

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"github.com/chainreactors/IoM-go/consts"
	"github.com/chainreactors/IoM-go/proto/client/clientpb"
	"github.com/chainreactors/IoM-go/types"
	"github.com/chainreactors/logs"
	"github.com/chainreactors/malice-network/helper/utils/fileutils"
	"github.com/chainreactors/malice-network/helper/utils/output"
	"github.com/chainreactors/malice-network/server/internal/certutils"
	"github.com/chainreactors/malice-network/server/internal/configs"
	"github.com/chainreactors/malice-network/server/internal/core"
	"github.com/chainreactors/malice-network/server/internal/db"
	"github.com/chainreactors/malice-network/server/internal/db/models"
	"google.golang.org/protobuf/proto"
	"os"
	"path"
	"strings"
)

func resolveWebsite(name, listenerID string) (*models.Pipeline, error) {
	if name == "" {
		return nil, fmt.Errorf("website name required")
	}

	query := db.NewPipelineQuery().WhereName(name).WhereType(consts.WebsitePipeline).WithCert()
	if listenerID != "" {
		query = query.WhereListenerID(listenerID)
	}

	websites, err := query.Find()
	if err != nil {
		return nil, err
	}

	switch len(websites) {
	case 0:
		return nil, types.ErrNotFoundPipeline
	case 1:
		return websites[0], nil
	default:
		return nil, fmt.Errorf("multiple websites named '%s' found across listeners, please specify listener_id", name)
	}
}

func getWebsiteRuntime(name, listenerID string) (*models.Pipeline, *core.Job, error) {
	website, err := resolveWebsite(name, listenerID)
	if err != nil {
		return nil, nil, err
	}

	job, err := core.Jobs.GetByListener(website.Name, website.ListenerId)
	if err != nil {
		if errors.Is(err, types.ErrNotFoundPipeline) {
			return website, nil, nil
		}
		return nil, nil, err
	}

	return website, job, nil
}

func cloneWebsiteJob(job *core.Job, contents map[string]*clientpb.WebContent) *clientpb.Job {
	if job == nil {
		return nil
	}
	if job.Pipeline == nil || job.Pipeline.GetWeb() == nil {
		return job.ToProtobuf()
	}

	pipelineCopy := proto.Clone(job.Pipeline).(*clientpb.Pipeline)
	pipelineCopy.GetWeb().Contents = contents

	return &clientpb.Job{
		Id:       job.ID,
		Name:     job.Name,
		Pipeline: pipelineCopy,
	}
}

func MapContents(webpipe *clientpb.Pipeline) error {
	if webpipe == nil || webpipe.GetWeb() == nil {
		return errors.New("website pipeline required")
	}
	web := webpipe.GetWeb()
	if web.Contents == nil {
		web.Contents = make(map[string]*clientpb.WebContent)
	}
	contents, err := db.FindWebContentsByWebsiteAndListener(webpipe.Name, webpipe.ListenerId)
	if err != nil {
		return err
	}

	for _, content := range contents {
		web.Contents[content.Path] = content.ToProtobuf(true)
	}
	return nil
}

// ListWebContent - 列出网站的所有内容
func (rpc *Server) ListWebContent(ctx context.Context, req *clientpb.Website) (*clientpb.WebContents, error) {
	if req == nil {
		return nil, types.ErrMissingRequestField
	}
	website, err := resolveWebsite(req.Name, req.ListenerId)
	if err != nil {
		return nil, err
	}
	contents, err := db.FindWebContentsByWebsiteAndListener(website.Name, website.ListenerId)
	if err != nil {
		return nil, err
	}
	res := &clientpb.WebContents{}
	for _, content := range contents {
		res.Contents = append(res.Contents, content.ToProtobuf(false))
	}

	return res, nil
}

// WebsiteAddContent - Add content to a website, the website is created if `name` does not exist
func (rpc *Server) AddWebsiteContent(ctx context.Context, req *clientpb.Website) (*clientpb.WebContent, error) {
	if req == nil {
		return nil, types.ErrMissingRequestField
	}
	website, job, err := getWebsiteRuntime(req.Name, req.ListenerId)
	if err != nil {
		return nil, err
	}
	req.ListenerId = website.ListenerId

	var lns *core.Listener
	if job != nil {
		lns, err = core.Listeners.Get(website.ListenerId)
		if err != nil {
			return nil, err
		}
	}
	var contentModel *models.WebsiteContent
	if len(req.Contents) != 0 {
		for _, content := range req.Contents {
			content.WebsiteId = website.Name
			content.ListenerId = website.ListenerId
			content.Size = uint64(len(content.Content))
			if rpcLog != nil {
				rpcLog.Infof("Add website content (%s) %s -> %s", content.File, content.Path, content.Type)
			}
			contentModel, err = db.AddContent(content)
			if err != nil {
				return nil, err
			}
			if job != nil {
				lns.PushCtrl(&clientpb.JobCtrl{
					Ctrl:    consts.CtrlWebContentAdd,
					Job:     job.ToProtobuf(),
					Content: content,
				})
			}
		}
	}
	if contentModel == nil {
		return nil, errors.New("no content provided")
	}

	return contentModel.ToProtobuf(false), nil
}

// WebsiteUpdateContent - Update specific content from a website
func (rpc *Server) UpdateWebsiteContent(ctx context.Context, req *clientpb.WebContent) (*clientpb.WebContent, error) {
	if req == nil {
		return nil, types.ErrMissingRequestField
	}
	existingContent, err := db.FindWebContent(req.Id)
	if err != nil {
		return nil, err
	}
	if req.WebsiteId == "" {
		req.WebsiteId = existingContent.PipelineID
	}
	if req.Path == "" {
		req.Path = existingContent.Path
	}
	if req.Type == "" {
		req.Type = existingContent.Type
	}
	if req.ContentType == "" {
		req.ContentType = existingContent.ContentType
	}
	if req.ListenerId == "" {
		req.ListenerId = existingContent.ListenerID
	}
	if req.ListenerId == "" && existingContent.Pipeline != nil {
		req.ListenerId = existingContent.Pipeline.ListenerId
	}

	content, err := db.AddContent(req)
	if err != nil {
		return nil, err
	}

	website, job, err := getWebsiteRuntime(req.WebsiteId, req.ListenerId)
	if err != nil {
		return nil, err
	}
	if job != nil {
		lns, err := core.Listeners.Get(website.ListenerId)
		if err != nil {
			return nil, err
		}
		lns.PushCtrl(&clientpb.JobCtrl{
			Ctrl:    consts.CtrlWebContentAdd,
			Job:     job.ToProtobuf(),
			Content: content.ToProtobuf(true),
		})
	}

	return content.ToProtobuf(false), nil
}

func (rpc *Server) UpdateWebsiteContentMetadata(ctx context.Context, req *clientpb.WebContent) (*clientpb.WebContent, error) {
	if req == nil {
		return nil, types.ErrMissingRequestField
	}
	content, err := db.UpdateWebContentMetadata(req)
	if err != nil {
		return nil, err
	}
	return content.ToProtobuf(false), nil
}

// WebsiteRemoveContent - Remove specific content from a website
func (rpc *Server) RemoveWebsiteContent(ctx context.Context, req *clientpb.WebContent) (*clientpb.Empty, error) {
	if req == nil {
		return nil, types.ErrMissingRequestField
	}
	existingContent, err := db.FindWebContent(req.Id)
	if err != nil {
		return nil, err
	}
	if req.WebsiteId == "" {
		req.WebsiteId = existingContent.PipelineID
	}
	if req.Path == "" {
		req.Path = existingContent.Path
	}
	if req.ListenerId == "" {
		req.ListenerId = existingContent.ListenerID
	}
	if req.ListenerId == "" && existingContent.Pipeline != nil {
		req.ListenerId = existingContent.Pipeline.ListenerId
	}

	website, job, err := getWebsiteRuntime(req.WebsiteId, req.ListenerId)
	if err != nil {
		return nil, err
	}
	if job != nil {
		lns, err := core.Listeners.Get(website.ListenerId)
		if err != nil {
			return nil, err
		}
		lns.PushCtrl(&clientpb.JobCtrl{
			Ctrl: consts.CtrlWebContentRemove,
			Job: cloneWebsiteJob(job, map[string]*clientpb.WebContent{
				req.Path: {
					Path: req.Path,
				},
			}),
		})
	}

	err = db.RemoveContent(req.Id)
	if err != nil {
		return nil, err
	}
	return &clientpb.Empty{}, nil
}

func (rpc *Server) RegisterWebsite(ctx context.Context, req *clientpb.Pipeline) (*clientpb.Empty, error) {
	if req == nil || req.GetWeb() == nil {
		return nil, types.ErrMissingRequestField
	}
	if err := validatePipelineIdentity(req); err != nil {
		return nil, err
	}
	lns, err := core.Listeners.Get(req.ListenerId)
	if err != nil {
		return nil, err
	}
	req.Ip = lns.IP
	pipelineModel := models.FromPipelinePb(req)
	_, err = db.SavePipeline(pipelineModel)
	if err != nil {
		return nil, err
	}
	websiteDir, err := fileutils.SafeJoin(configs.WebsitePath, path.Join(req.ListenerId, req.Name))
	if err != nil {
		return nil, err
	}
	if err := os.MkdirAll(websiteDir, 0o700); err != nil {
		return nil, err
	}
	for _, content := range req.GetWeb().Contents {
		content.WebsiteId = req.Name
		content.ListenerId = req.ListenerId
		_, err = db.AddContent(content)
		if err != nil {
			return nil, err
		}
	}

	return &clientpb.Empty{}, nil
}

func (rpc *Server) StartWebsite(ctx context.Context, req *clientpb.CtrlPipeline) (*clientpb.Empty, error) {
	if req == nil {
		return nil, types.ErrMissingRequestField
	}
	listenerID, err := resolveListenerID(req)
	if err != nil {
		return nil, err
	}

	webpipe, err := resolveWebsite(req.Name, listenerID)
	if err != nil {
		return nil, err
	}
	certName := req.GetCertName()
	if certName == "" && req.GetPipeline() != nil {
		certName = req.GetPipeline().GetCertName()
	}
	if certName != "" {
		_, err := db.FindCertificate(certName)
		if err != nil {
			return nil, err
		}
		webpipe, err = db.UpdatePipelineCert(certName, webpipe)
		if err != nil {
			return nil, err
		}
	} else if req.Pipeline != nil && req.Pipeline.Tls != nil {
		tlsConfig, err := prepareWebsiteTLS(webpipe.Name, req.Pipeline.Tls)
		if err != nil {
			return nil, err
		}
		webpipe, err = db.SetPipelineTLS(webpipe, tlsConfig, req.Pipeline.CertName)
		if err != nil {
			return nil, err
		}
	}
	listener, err := core.Listeners.Get(webpipe.ListenerId)
	if err != nil {
		return nil, err
	}
	webpb := webpipe.ToProtobuf()
	err = MapContents(webpb)
	if err != nil {
		return nil, err
	}
	webpb.Enable = true
	job := &core.Job{
		ID:       core.NextJobID(),
		Pipeline: webpb,
		Name:     webpipe.Name,
	}
	ctrlID := listener.PushCtrl(&clientpb.JobCtrl{
		Ctrl: consts.CtrlWebsiteStart,
		Job:  job.ToProtobuf(),
	})

	status := listener.WaitCtrl(ctrlID)
	if err := waitForCtrlStatus("start website", req.Name, status); err != nil {
		_ = db.DisablePipelineByListener(webpipe.Name, webpipe.ListenerId)
		return nil, err
	}

	err = db.EnablePipelineByListener(webpipe.Name, webpipe.ListenerId)
	if err != nil {
		return nil, err
	}
	core.Jobs.AddPipeline(webpb)

	artifacts, err := db.GetValidArtifacts()
	if err != nil {
		logs.Log.Errorf("failed to find artifact: %s", err)
		return &clientpb.Empty{}, nil
	}
	for _, artifact := range artifacts {
		content, err := db.AddAmountWebContent(artifact.Name, webpb.Name, webpb.ListenerId)
		if err != nil {
			return nil, err
		}
		listener.PushCtrl(&clientpb.JobCtrl{
			Ctrl: consts.CtrlWebContentAddArtifact,
			Job: &clientpb.Job{
				Pipeline: webpb,
			},
			Content: content,
		})
		logs.Log.Infof("artifact %s amounts at %s", artifact.Name, path.Join(webpb.URL(), output.Encode(artifact.Name)))
	}
	return &clientpb.Empty{}, nil
}

func (rpc *Server) UpdateWebsiteTLS(ctx context.Context, req *clientpb.PipelineTLSUpdate) (*clientpb.Pipeline, error) {
	if req == nil {
		return nil, types.ErrMissingRequestField
	}
	listenerID, err := resolveWebsiteTLSListenerID(req)
	if err != nil {
		return nil, err
	}

	website, job, err := getWebsiteRuntime(req.Name, listenerID)
	if err != nil {
		return nil, err
	}

	updated, err := applyWebsiteTLSUpdate(website, req)
	if err != nil {
		return nil, err
	}

	if job != nil {
		if _, err := rpc.StopWebsite(ctx, &clientpb.CtrlPipeline{Name: updated.Name, ListenerId: updated.ListenerId}); err != nil {
			return nil, err
		}
		if _, err := rpc.StartWebsite(ctx, &clientpb.CtrlPipeline{Name: updated.Name, ListenerId: updated.ListenerId}); err != nil {
			return nil, err
		}
	}
	updated, err = resolveWebsite(updated.Name, updated.ListenerId)
	if err != nil {
		return nil, err
	}

	pb := updated.ToProtobuf()
	if err := MapContents(pb); err != nil {
		return nil, err
	}
	return pb, nil
}

func resolveWebsiteTLSListenerID(req *clientpb.PipelineTLSUpdate) (string, error) {
	if req.GetListenerId() != "" {
		return req.GetListenerId(), nil
	}
	if req.GetName() == "" {
		return "", fmt.Errorf("website name required")
	}
	websites, err := db.NewPipelineQuery().WhereName(req.GetName()).WhereType(consts.WebsitePipeline).Find()
	if err != nil {
		return "", err
	}
	switch len(websites) {
	case 0:
		return "", types.ErrNotFoundPipeline
	case 1:
		return websites[0].ListenerId, nil
	default:
		return "", fmt.Errorf("multiple websites named '%s' found across listeners, please specify listener_id", req.GetName())
	}
}

func applyWebsiteTLSUpdate(website *models.Pipeline, req *clientpb.PipelineTLSUpdate) (*models.Pipeline, error) {
	if website == nil {
		return nil, types.ErrNotFoundPipeline
	}
	switch req.GetMode() {
	case clientpb.TLSUpdateMode_TLS_UPDATE_MODE_DISABLE:
		return db.SetPipelineTLS(website, &clientpb.TLS{Enable: false}, "")
	case clientpb.TLSUpdateMode_TLS_UPDATE_MODE_EXISTING_CERT:
		certName := strings.TrimSpace(req.GetCertName())
		if certName == "" {
			return nil, fmt.Errorf("cert_name is required")
		}
		cert, err := db.FindCertificate(certName)
		if err != nil {
			return nil, err
		}
		tlsConfig := cert.ToProtobuf()
		if tlsConfig == nil || tlsConfig.Cert == nil || tlsConfig.Cert.Cert == "" || tlsConfig.Cert.Key == "" {
			return nil, fmt.Errorf("certificate %s is invalid", certName)
		}
		return db.UpdatePipelineCert(certName, website)
	case clientpb.TLSUpdateMode_TLS_UPDATE_MODE_INLINE_CERT:
		return applyInlineWebsiteTLS(website, req)
	default:
		return nil, fmt.Errorf("tls update mode is required")
	}
}

func applyInlineWebsiteTLS(website *models.Pipeline, req *clientpb.PipelineTLSUpdate) (*models.Pipeline, error) {
	tlsConfig := req.GetTls()
	if tlsConfig != nil {
		tlsConfig.Enable = true
	}
	tlsConfig, err := prepareWebsiteTLS(website.Name, tlsConfig)
	if err != nil {
		return nil, err
	}
	tlsConfig.Enable = true
	if req.GetSaveCert() {
		saveName := strings.TrimSpace(req.GetSaveCertName())
		if saveName == "" {
			return nil, fmt.Errorf("save_cert_name is required when save_cert is enabled")
		}
		certModel, err := db.SaveCertFromTLSWithOptions(tlsConfig, "", "", db.SaveCertOptions{
			Name:    saveName,
			Comment: req.GetCertComment(),
		})
		if err != nil {
			return nil, err
		}
		certTLS := certModel.ToProtobuf()
		if certTLS == nil {
			return nil, fmt.Errorf("saved certificate %s is invalid", certModel.Name)
		}
		return db.UpdatePipelineCert(certModel.Name, website)
	}
	tlsConfig.Cert.Name = ""
	tlsConfig.Cert.Comment = req.GetCertComment()
	return db.SetPipelineTLS(website, tlsConfig, "")
}

func prepareWebsiteTLS(name string, tlsConfig *clientpb.TLS) (*clientpb.TLS, error) {
	if tlsConfig == nil {
		return nil, fmt.Errorf("tls config is required")
	}
	if !tlsConfig.GetEnable() {
		return tlsConfig, nil
	}
	if tlsConfig.GetCert().GetCert() == "" && tlsConfig.GetCert().GetKey() == "" {
		generated, err := certutils.GenerateSelfTLS(name, tlsConfig.GetCertSubject())
		if err != nil {
			return nil, err
		}
		generated.CertSubject = tlsConfig.GetCertSubject()
		return generated, nil
	}
	if tlsConfig.GetCert().GetCert() == "" || tlsConfig.GetCert().GetKey() == "" {
		return nil, fmt.Errorf("cert and key are required")
	}
	if _, err := tls.X509KeyPair([]byte(tlsConfig.GetCert().GetCert()), []byte(tlsConfig.GetCert().GetKey())); err != nil {
		return nil, fmt.Errorf("invalid certificate key pair: %w", err)
	}
	return tlsConfig, nil
}

func (rpc *Server) StopWebsite(ctx context.Context, req *clientpb.CtrlPipeline) (*clientpb.Empty, error) {
	listenerID, err := resolveListenerID(req)
	if err != nil {
		return nil, err
	}

	website, job, err := getWebsiteRuntime(req.Name, listenerID)
	if err != nil {
		return nil, err
	}

	if job != nil {
		listener, err := core.Listeners.Get(website.ListenerId)
		if err != nil {
			core.Jobs.Remove(website.ListenerId, website.Name)
		} else {
			ctrlID := listener.PushCtrl(&clientpb.JobCtrl{
				Ctrl: consts.CtrlWebsiteStop,
				Job:  job.ToProtobuf(),
			})
			status := listener.WaitCtrl(ctrlID)
			if err := waitForCtrlStatus("stop website", req.Name, status); err != nil {
				return nil, err
			}
		}
	}

	err = db.DisablePipelineByListener(website.Name, website.ListenerId)
	if err != nil {
		return nil, err
	}
	if job != nil {
		if listener, err := core.Listeners.Get(website.ListenerId); err == nil {
			listener.RemovePipeline(job.Pipeline)
		} else {
			core.Jobs.Remove(website.ListenerId, website.Name)
		}
	}
	return &clientpb.Empty{}, nil
}

func (rpc *Server) ListWebsites(ctx context.Context, req *clientpb.Listener) (*clientpb.Pipelines, error) {
	listenerID := ""
	if req != nil {
		listenerID = req.Id
	}
	modelPipelines, err := db.ListWebsitesByListener(listenerID)
	if err != nil {
		return nil, err
	}
	result := modelPipelines.ToProtobuf()
	for _, pipeline := range result.Pipelines {
		_, runtimeErr := core.Jobs.GetByListener(pipeline.Name, pipeline.ListenerId)
		runtimePipeline, runtimeOK := core.Listeners.FindByListener(pipeline.ListenerId, pipeline.Name)
		pipeline.Enable = pipeline.Enable && runtimeErr == nil && runtimeOK && runtimePipeline.Enable
	}
	return result, nil
}

func (rpc *Server) DeleteWebsite(ctx context.Context, req *clientpb.CtrlPipeline) (*clientpb.Empty, error) {
	listenerID, err := resolveListenerID(req)
	if err != nil {
		return nil, err
	}

	website, job, err := getWebsiteRuntime(req.Name, listenerID)
	if err != nil {
		return nil, err
	}
	if job != nil {
		if listener, err := core.Listeners.Get(website.ListenerId); err == nil {
			ctrlID := listener.PushCtrl(&clientpb.JobCtrl{
				Ctrl: consts.CtrlWebsiteStop,
				Job:  job.ToProtobuf(),
			})
			status := listener.WaitCtrl(ctrlID)
			if err := waitForCtrlStatus("delete website", req.Name, status); err != nil {
				return nil, err
			}
			listener.RemovePipeline(job.Pipeline)
		} else {
			core.Jobs.Remove(website.ListenerId, website.Name)
		}
	}

	err = db.DeleteWebsiteByListener(website.Name, website.ListenerId)
	if err != nil && !errors.Is(err, db.ErrRecordNotFound) {
		return nil, err
	}

	return &clientpb.Empty{}, nil
}

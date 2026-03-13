package rpc

import (
	"context"
	"errors"
	"fmt"
	"github.com/chainreactors/IoM-go/consts"
	"github.com/chainreactors/IoM-go/proto/client/clientpb"
	"github.com/chainreactors/IoM-go/types"
	"github.com/chainreactors/logs"
	"github.com/chainreactors/malice-network/helper/utils/fileutils"
	"github.com/chainreactors/malice-network/helper/utils/output"
	"github.com/chainreactors/malice-network/server/internal/configs"
	"github.com/chainreactors/malice-network/server/internal/core"
	"github.com/chainreactors/malice-network/server/internal/db"
	"github.com/chainreactors/malice-network/server/internal/db/models"
	"google.golang.org/protobuf/proto"
	"os"
	"path"
)

func resolveWebsite(name, listenerID string) (*models.Pipeline, error) {
	if name == "" {
		return nil, fmt.Errorf("website name required")
	}

	query := db.NewPipelineQuery().WhereName(name).WhereType(consts.WebsitePipeline)
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
	web := webpipe.GetWeb()
	contents, err := db.FindWebContentsByWebsite(webpipe.Name)
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
	contents, err := db.FindWebContentsByWebsite(req.Name)
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
			content.Size = uint64(len(content.Content))
			rpcLog.Infof("Add website content (%s) %s -> %s", content.File, content.Path, content.Type)
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

// WebsiteRemoveContent - Remove specific content from a website
func (rpc *Server) RemoveWebsiteContent(ctx context.Context, req *clientpb.WebContent) (*clientpb.Empty, error) {
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
	websiteDir, err := fileutils.SafeJoin(configs.WebsitePath, req.Name)
	if err != nil {
		return nil, err
	}
	if err := os.MkdirAll(websiteDir, 0o700); err != nil {
		return nil, err
	}
	for _, content := range req.GetWeb().Contents {
		content.WebsiteId = req.Name
		_, err = db.AddContent(content)
		if err != nil {
			return nil, err
		}
	}

	return &clientpb.Empty{}, nil
}

func (rpc *Server) StartWebsite(ctx context.Context, req *clientpb.CtrlPipeline) (*clientpb.Empty, error) {
	listenerID, err := resolveListenerID(req)
	if err != nil {
		return nil, err
	}

	webpipe, err := resolveWebsite(req.Name, listenerID)
	if err != nil {
		return nil, err
	}
	if req.CertName != "" {
		_, err := db.FindCertificate(req.CertName)
		if err != nil {
			return nil, err
		}
		webpipe, err = db.UpdatePipelineCert(req.CertName, webpipe)
		if err != nil {
			return nil, err
		}
	} else if req.Pipeline != nil && req.Pipeline.Tls != nil {
		webpipe.Tls.Enable = req.Pipeline.Tls.Enable
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
		content, err := db.AddAmountWebContent(artifact.Name, webpb.Name)
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

func (rpc *Server) StopWebsite(ctx context.Context, req *clientpb.CtrlPipeline) (*clientpb.Empty, error) {
	listenerID, err := resolveListenerID(req)
	if err != nil {
		return nil, err
	}

	website, job, err := getWebsiteRuntime(req.Name, listenerID)
	if err != nil {
		return nil, err
	}

	listener, err := core.Listeners.Get(website.ListenerId)
	if err != nil {
		return nil, err
	}

	if job != nil {
		ctrlID := listener.PushCtrl(&clientpb.JobCtrl{
			Ctrl: consts.CtrlWebsiteStop,
			Job:  job.ToProtobuf(),
		})
		status := listener.WaitCtrl(ctrlID)
		if err := waitForCtrlStatus("stop website", req.Name, status); err != nil {
			return nil, err
		}
	}

	err = db.DisablePipelineByListener(website.Name, website.ListenerId)
	if err != nil {
		return nil, err
	}
	if job != nil {
		listener.RemovePipeline(job.Pipeline)
	}
	return &clientpb.Empty{}, nil
}

func (rpc *Server) ListWebsites(ctx context.Context, req *clientpb.Listener) (*clientpb.Pipelines, error) {
	modelPipelines, err := db.ListWebsitesByListener(req.Id)
	if err != nil {
		return nil, err
	}
	return modelPipelines.ToProtobuf(), nil
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
	listener, err := core.Listeners.Get(website.ListenerId)
	if err != nil {
		return nil, err
	}

	if job != nil {
		ctrlID := listener.PushCtrl(&clientpb.JobCtrl{
			Ctrl: consts.CtrlWebsiteStop,
			Job:  job.ToProtobuf(),
		})
		status := listener.WaitCtrl(ctrlID)
		if err := waitForCtrlStatus("delete website", req.Name, status); err != nil {
			return nil, err
		}
		listener.RemovePipeline(job.Pipeline)
	}

	err = db.DeleteWebsite(website.Name)
	if err != nil && !errors.Is(err, db.ErrRecordNotFound) {
		return nil, err
	}

	return &clientpb.Empty{}, nil
}

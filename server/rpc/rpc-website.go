package rpc

import (
	"context"
	"fmt"
	"github.com/chainreactors/malice-network/helper/consts"
	"github.com/chainreactors/malice-network/helper/proto/client/clientpb"
	"github.com/chainreactors/malice-network/server/internal/certutils"
	"github.com/chainreactors/malice-network/server/internal/core"
	"github.com/chainreactors/malice-network/server/internal/db"
	"github.com/chainreactors/malice-network/server/internal/db/models"
	"github.com/chainreactors/malice-network/server/internal/website"
	"mime"
	"path/filepath"
	"strings"
)

func (rpc *Server) Websites(ctx context.Context, _ *clientpb.Empty) (*clientpb.Websites, error) {
	websiteNames, err := website.Names()
	if err != nil {
		return nil, err
	}
	websites := &clientpb.Websites{Websites: []*clientpb.Website{}}
	for _, name := range websiteNames {
		siteContent, err := website.MapContent(name, false)
		if err != nil {
			continue
		}
		websites.Websites = append(websites.Websites, siteContent)
	}
	return websites, nil
}

func (rpc *Server) WebsiteRemove(ctx context.Context, req *clientpb.Website) (*clientpb.Empty, error) {
	dbWebsite, err := website.WebsiteByName(req.ID)
	if err != nil {
		return nil, err
	}

	err = db.RemoveWebsite(dbWebsite.ID)
	if err != nil {
		return nil, err
	}
	core.EventBroker.Publish(core.Event{
		EventType: consts.EventWebsite,
		Op:        consts.CtrlWebsiteRemove,
		Message:   req.ID,
		Important: true,
	})

	return &clientpb.Empty{}, nil
}

// Website - Get one website
func (rpc *Server) Website(ctx context.Context, req *clientpb.Website) (*clientpb.Website, error) {
	return website.MapContent(req.ID, true)
}

// WebsiteAddContent - Add content to a website, the website is created if `name` does not exist
func (rpc *Server) WebsiteAddContent(ctx context.Context, req *clientpb.WebsiteAddContent) (*clientpb.Website, error) {
	if 0 < len(req.Contents) {
		for _, content := range req.Contents {
			// If no content-type was specified by the client we try to detect the mime based on path ext
			if content.ContentType == "" {
				content.ContentType = mime.TypeByExtension(filepath.Ext(content.Path))
				if content.ContentType == "" {
					content.ContentType = "text/html; charset=utf-8" // Default mime
				}
			}

			content.Size = uint64(len(content.Content))
			rpcLog.Infof("Add website content (%s) %s -> %s", req.Name, content.Path, content.ContentType)
			_, err := website.AddContent(req.Name, content)
			if err != nil {
				return nil, err
			}
		}
	} else {
		_, err := website.AddWebsite(req.Name)
		if err != nil {
			return nil, err
		}
	}

	core.EventBroker.Publish(core.Event{
		EventType: consts.EventWebsite,
		Op:        consts.CtrlWebsiteAdd,
		Message:   req.Name,
		Important: true,
	})

	return website.MapContent(req.Name, true)
}

// WebsiteUpdateContent - Update specific content from a website, currently you can only the update Content-type field
func (rpc *Server) WebsiteUpdateContent(ctx context.Context, req *clientpb.WebsiteAddContent) (*clientpb.Website, error) {
	dbWebsite, err := website.WebsiteByName(req.Name)
	if err != nil {
		return nil, err
	}
	for _, content := range req.Contents {
		_, _ = website.AddContent(dbWebsite.ID, content)
	}

	core.EventBroker.Publish(core.Event{
		EventType: consts.EventWebsite,
		Op:        consts.CtrlWebsiteUpdate,
		Message:   req.Name,
	})

	return website.MapContent(req.Name, false)
}

// WebsiteRemoveContent - Remove specific content from a website
func (rpc *Server) WebsiteRemoveContent(ctx context.Context, req *clientpb.WebsiteRemoveContent) (*clientpb.Website, error) {
	for _, path := range req.Paths {
		err := website.RemoveContent(req.Name, path)
		if err != nil {
			return nil, err
		}
	}

	core.EventBroker.Publish(core.Event{
		EventType: consts.EventWebsite,
		Message:   req.Name,
	})

	return website.MapContent(req.Name, false)
}

func (rpc *Server) RegisterWebsite(ctx context.Context, req *clientpb.Pipeline) (*clientpb.WebsiteResponse, error) {
	ip := getRemoteAddr(ctx)
	ip = strings.Split(ip, ":")[0]
	pipelineModel := models.FromPipelinePb(req, ip)
	var err error
	if pipelineModel.Tls.Enable && pipelineModel.Tls.Cert == "" && pipelineModel.Tls.Key == "" {
		pipelineModel.Tls.Cert, pipelineModel.Tls.Key, err = certutils.GenerateTlsCert(req.Name, req.ListenerId)
		if err != nil {
			return &clientpb.WebsiteResponse{}, err
		}
	}
	err = db.CreatePipeline(pipelineModel)
	var id = ""
	if err != nil {
		return &clientpb.WebsiteResponse{}, err
	}
	getWeb := req.GetWeb()
	if 0 < len(getWeb.Contents) {
		for _, content := range getWeb.Contents {
			if content.ContentType == "" {
				content.ContentType = mime.TypeByExtension(filepath.Ext(content.Path))
				if content.ContentType == "" {
					content.ContentType = "text/html; charset=utf-8" // Default mime
				}
			}
			content.Size = uint64(len(content.Content))
			id, err = website.AddContent(getWeb.ID, content)
			if err != nil {
				return nil, err
			}
		}
	}
	return &clientpb.WebsiteResponse{ID: id}, nil
}

func (rpc *Server) StartWebsite(ctx context.Context, req *clientpb.CtrlPipeline) (*clientpb.Empty, error) {
	pipelineDB, err := db.FindPipeline(req.Name)
	if err != nil {
		return &clientpb.Empty{}, err
	}
	pipeline := models.ToPipelinePB(pipelineDB)
	listener := core.Listeners.Get(pipeline.ListenerId)
	if listener == nil {
		return nil, fmt.Errorf("listener %s not found", req.ListenerId)
	}
	listener.AddPipeline(pipeline)
	w, err := website.MapContents(req.Name)
	if err != nil {
		return &clientpb.Empty{}, err
	}
	pipeline.GetWeb().Contents = w.Contents
	pipeline.Enable = true
	pipeline.GetWeb().ID = pipelineDB.WebPath
	core.Jobs.Add(&core.Job{
		ID:      core.CurrentJobID(),
		Message: pipeline,
		Name:    pipeline.Name,
	})

	core.Jobs.Ctrl <- &clientpb.JobCtrl{
		Id:   core.NextCtrlID(),
		Ctrl: consts.CtrlWebsiteStart,
		Job: &clientpb.Job{
			Id:       core.NextJobID(),
			Pipeline: pipeline,
		},
	}
	err = db.EnablePipeline(pipelineDB)
	if err != nil {
		return nil, err
	}

	return &clientpb.Empty{}, nil
}

func (rpc *Server) StopWebsite(ctx context.Context, req *clientpb.CtrlPipeline) (*clientpb.Empty, error) {
	pipelineDB, err := db.FindPipeline(req.Name)
	if err != nil {
		return nil, err
	}
	pipeline := models.ToPipelinePB(pipelineDB)
	ctrl := clientpb.JobCtrl{
		Id:   core.NextCtrlID(),
		Ctrl: consts.CtrlWebsiteStop,
		Job: &clientpb.Job{
			Id:       core.NextJobID(),
			Pipeline: pipeline,
		},
	}
	core.Jobs.Ctrl <- &ctrl
	err = db.DisablePipeline(pipelineDB)
	listener := core.Listeners.Get(pipeline.ListenerId)
	if listener == nil {
		return nil, fmt.Errorf("listener %s not found", req.ListenerId)
	}
	listener.RemovePipeline(pipeline)
	if err != nil {
		return nil, err
	}
	return &clientpb.Empty{}, nil

}

func (rpc *Server) UploadWebsite(ctx context.Context, req *clientpb.Website) (*clientpb.Empty, error) {
	ctrl := clientpb.JobCtrl{
		Id:   core.NextCtrlID(),
		Ctrl: consts.CtrlWebsiteRegister,
		Job: &clientpb.Job{
			Id: core.NextJobID(),
			Pipeline: &clientpb.Pipeline{
				Body: &clientpb.Pipeline_Web{
					Web: req,
				},
			},
		},
	}

	core.Jobs.Ctrl <- &ctrl
	return &clientpb.Empty{}, nil
}

func (rpc *Server) ListWebsites(ctx context.Context, req *clientpb.Listener) (*clientpb.Pipelines, error) {
	var websites []*clientpb.Pipeline
	pipelines, err := db.ListWebsite(req.Id)
	if err != nil {
		return nil, err
	}
	for _, pipeline := range pipelines {
		websites = append(websites, models.ToPipelinePB(pipeline))
	}
	return &clientpb.Pipelines{Pipelines: websites}, nil
}

func (rpc *Server) DeleteWebsite(ctx context.Context, req *clientpb.CtrlPipeline) (*clientpb.Empty, error) {
	pipelineDB, err := db.FindPipeline(req.Name)
	if err != nil {
		return nil, err
	}
	pipeline := models.ToPipelinePB(pipelineDB)
	listener := core.Listeners.Get(pipeline.ListenerId)
	if listener == nil {
		return nil, fmt.Errorf("listener %s not found", req.ListenerId)
	}
	listener.RemovePipeline(pipeline)
	core.Jobs.Ctrl <- &clientpb.JobCtrl{
		Id:   core.NextCtrlID(),
		Ctrl: consts.CtrlWebsiteStop,
		Job: &clientpb.Job{
			Id:       core.NextJobID(),
			Pipeline: pipeline,
		},
	}
	err = db.DeletePipeline(req.Name)
	if err != nil {
		return nil, err
	}
	err = db.DeleteWebsite(req.Name)
	if err != nil {
		return nil, err
	}
	return &clientpb.Empty{}, nil
}

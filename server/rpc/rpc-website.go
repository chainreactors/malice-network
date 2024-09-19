package rpc

import (
	"context"
	"github.com/chainreactors/malice-network/helper/consts"
	"github.com/chainreactors/malice-network/proto/client/clientpb"
	"github.com/chainreactors/malice-network/proto/listener/lispb"
	"github.com/chainreactors/malice-network/server/internal/certutils"
	"github.com/chainreactors/malice-network/server/internal/core"
	"github.com/chainreactors/malice-network/server/internal/db"
	"github.com/chainreactors/malice-network/server/internal/db/models"
	"github.com/chainreactors/malice-network/server/internal/website"
	"mime"
	"path/filepath"
)

func (rpc *Server) Websites(ctx context.Context, _ *clientpb.Empty) (*lispb.Websites, error) {
	websiteNames, err := website.Names()
	if err != nil {
		return nil, err
	}
	websites := &lispb.Websites{Websites: []*lispb.Website{}}
	for _, name := range websiteNames {
		siteContent, err := website.MapContent(name, false)
		if err != nil {
			continue
		}
		websites.Websites = append(websites.Websites, siteContent)
	}
	return websites, nil
}

func (rpc *Server) WebsiteRemove(ctx context.Context, req *lispb.Website) (*clientpb.Empty, error) {
	dbWebsite, err := website.WebsiteByName(req.Name)
	if err != nil {
		return nil, err
	}

	err = db.RemoveWebsite(dbWebsite.ID)
	if err != nil {
		return nil, err
	}
	core.EventBroker.Publish(core.Event{
		EventType: consts.EventWebsite,
		Message:   req.Name,
	})

	return &clientpb.Empty{}, nil
}

// Website - Get one website
func (rpc *Server) Website(ctx context.Context, req *lispb.Website) (*lispb.Website, error) {
	return website.MapContent(req.Name, true)
}

// WebsiteAddContent - Add content to a website, the website is created if `name` does not exist
func (rpc *Server) WebsiteAddContent(ctx context.Context, req *lispb.WebsiteAddContent) (*lispb.Website, error) {

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
		Message:   req.Name,
	})

	return website.MapContent(req.Name, true)
}

// WebsiteUpdateContent - Update specific content from a website, currently you can only the update Content-type field
func (rpc *Server) WebsiteUpdateContent(ctx context.Context, req *lispb.WebsiteAddContent) (*lispb.Website, error) {
	dbWebsite, err := website.WebsiteByName(req.Name)
	if err != nil {
		return nil, err
	}
	for _, content := range req.Contents {
		_, _ = website.AddContent(dbWebsite.Name, content)
	}

	core.EventBroker.Publish(core.Event{
		EventType: consts.EventWebsite,
		Message:   req.Name,
	})

	return website.MapContent(req.Name, false)
}

// WebsiteRemoveContent - Remove specific content from a website
func (rpc *Server) WebsiteRemoveContent(ctx context.Context, req *lispb.WebsiteRemoveContent) (*lispb.Website, error) {
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

func (rpc *Server) RegisterWebsite(ctx context.Context, req *lispb.Pipeline) (*lispb.WebsiteResponse, error) {
	if req.GetTls().Enable && req.GetTls().Cert == "" && req.GetTls().Key == "" {
		cert, key, err := certutils.GenerateTlsCert(req.GetWeb().Name)
		if err != nil {
			return &lispb.WebsiteResponse{}, err
		}
		req.GetTls().Cert = cert
		req.GetTls().Key = key
	}
	err := db.CreatePipeline(req)
	var id = ""
	if err != nil {
		return &lispb.WebsiteResponse{}, err
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
			id, err = website.AddContent(getWeb.Name, content)
			if err != nil {
				return nil, err
			}
		}
	}
	return &lispb.WebsiteResponse{ID: id}, nil
}

func (rpc *Server) StartWebsite(ctx context.Context, req *lispb.CtrlPipeline) (*clientpb.Empty, error) {
	pipelineDB, err := db.FindPipeline(req.Name, req.ListenerId)
	if err != nil {
		return &clientpb.Empty{}, err
	}
	pipeline := models.ToProtobuf(&pipelineDB)
	contents, err := website.MapContent(req.Name, true)
	if err != nil {
		return &clientpb.Empty{}, err
	}
	pipeline.GetWeb().Contents = contents.Contents
	pipeline.GetWeb().Enable = true
	job := &core.Job{
		ID:      core.CurrentJobID(),
		Message: pipeline,
		Name:    pipeline.GetWeb().Name,
	}
	core.Jobs.Add(job)
	ctrl := clientpb.JobCtrl{
		Id:   core.NextCtrlID(),
		Ctrl: consts.CtrlWebsiteStart,
		Job: &clientpb.Job{
			Id:       core.NextJobID(),
			Pipeline: pipeline,
		},
	}
	core.Jobs.Ctrl <- &ctrl
	return &clientpb.Empty{}, nil
}

func (rpc *Server) StopWebsite(ctx context.Context, req *lispb.CtrlPipeline) (*clientpb.Empty, error) {
	pipelineDB, err := db.FindPipeline(req.Name, req.ListenerId)
	if err != nil {
		return &clientpb.Empty{}, err
	}
	pipeline := models.ToProtobuf(&pipelineDB)
	ctrl := clientpb.JobCtrl{
		Id:   core.NextCtrlID(),
		Ctrl: consts.CtrlWebsiteStop,
		Job: &clientpb.Job{
			Id:       core.NextJobID(),
			Pipeline: pipeline,
		},
	}
	core.Jobs.Ctrl <- &ctrl
	return &clientpb.Empty{}, nil

}

func (rpc *Server) UploadWebsite(ctx context.Context, req *lispb.WebsiteAssets) (*clientpb.Empty, error) {
	db.WebsiteByName(req.Assets[0].WebName, "")
	ctrl := clientpb.JobCtrl{
		Id:   core.NextCtrlID(),
		Ctrl: consts.CtrlWebsiteRegister,
		Job: &clientpb.Job{
			Id:            core.NextJobID(),
			WebsiteAssets: req,
		},
	}
	core.Jobs.Ctrl <- &ctrl
	return &clientpb.Empty{}, nil
}

func (rpc *Server) ListWebsites(ctx context.Context, req *lispb.ListenerName) (*lispb.Websites, error) {
	var websites []*lispb.Website
	pipelines, err := db.ListPipelines(req.Name, "web")
	if err != nil {
		return nil, err
	}
	for _, pipeline := range pipelines {
		webProtoBuf := &lispb.Website{
			Name:     pipeline.Name,
			RootPath: pipeline.WebPath,
			Port:     uint32(pipeline.Port),
			Enable:   pipeline.Enable,
		}

		websites = append(websites, webProtoBuf)
	}
	return &lispb.Websites{Websites: websites}, nil
}

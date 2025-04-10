package rpc

import (
	"context"
	"errors"
	"github.com/chainreactors/malice-network/helper/consts"
	"github.com/chainreactors/malice-network/helper/proto/client/clientpb"
	"github.com/chainreactors/malice-network/server/internal/certutils"
	"github.com/chainreactors/malice-network/server/internal/configs"
	"github.com/chainreactors/malice-network/server/internal/core"
	"github.com/chainreactors/malice-network/server/internal/db"
	"github.com/chainreactors/malice-network/server/internal/db/models"
	"os"
	"path/filepath"
)

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
	job, err := core.Jobs.Get(req.Name)
	if err != nil {
		return nil, err
	}
	lns, err := core.Listeners.Get(job.Pipeline.ListenerId)
	if err != nil {
		return nil, err
	}
	var contentModel *models.WebsiteContent
	if len(req.Contents) != 0 {
		for _, content := range req.Contents {
			content.Size = uint64(len(content.Content))
			rpcLog.Infof("Add website content (%s) %s -> %s", content.File, content.Path, content.Type)
			contentModel, err = db.AddContent(content)
			if err != nil {
				return nil, err
			}

			job.Pipeline.GetWeb().Contents[content.Path] = content
			lns.PushCtrl(&clientpb.JobCtrl{
				Ctrl: consts.CtrlWebContentAdd,
				Job:  job.ToProtobuf(),
			})
		}
	}

	return contentModel.ToProtobuf(false), nil
}

// WebsiteUpdateContent - Update specific content from a website
func (rpc *Server) UpdateWebsiteContent(ctx context.Context, req *clientpb.WebContent) (*clientpb.WebContent, error) {
	content, err := db.AddContent(req)
	if err != nil {
		return nil, err
	}

	job, err := core.Jobs.Get(req.WebsiteId)
	if err != nil {
		return nil, err
	}
	lns, err := core.Listeners.Get(job.Pipeline.ListenerId)
	if err != nil {
		return nil, err
	}
	lns.PushCtrl(&clientpb.JobCtrl{
		Ctrl: consts.CtrlWebContentAdd,
		Job:  job.ToProtobuf(),
	})

	return content.ToProtobuf(false), nil
}

// WebsiteRemoveContent - Remove specific content from a website
func (rpc *Server) RemoveWebsiteContent(ctx context.Context, req *clientpb.WebContent) (*clientpb.Empty, error) {
	job, err := core.Jobs.Get(req.WebsiteId)
	if err != nil {
		return nil, err
	}
	lns, err := core.Listeners.Get(job.Pipeline.ListenerId)
	if err != nil {
		return nil, err
	}
	lns.PushCtrl(&clientpb.JobCtrl{
		Ctrl: consts.CtrlWebContentRemove,
		Job:  job.ToProtobuf(),
	})

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
	if pipelineModel.Tls.Enable && pipelineModel.Tls.Cert == "" && pipelineModel.Tls.Key == "" {
		pipelineModel.Tls.Cert, pipelineModel.Tls.Key, err = certutils.GenerateTlsCert(req.Name, req.ListenerId)
		if err != nil {
			return nil, err
		}
	}
	_, err = db.SavePipeline(pipelineModel)
	if err != nil {
		return nil, err
	}
	_ = os.Mkdir(filepath.Join(configs.WebsitePath, req.Name), os.ModePerm)
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
	webpipe, err := db.FindWebsiteByName(req.Name)
	if err != nil {
		return nil, err
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
	core.Jobs.Add(job)
	listener.PushCtrl(&clientpb.JobCtrl{
		Ctrl: consts.CtrlWebsiteStart,
		Job:  job.ToProtobuf(),
	})
	err = db.EnablePipeline(webpipe.Name)
	if err != nil {
		return nil, err
	}

	return &clientpb.Empty{}, nil
}

func (rpc *Server) StopWebsite(ctx context.Context, req *clientpb.CtrlPipeline) (*clientpb.Empty, error) {
	job, err := core.Jobs.Get(req.Name)
	if err != nil {
		return nil, err
	}

	err = db.DisablePipeline(job.Pipeline.Name)
	listener, err := core.Listeners.Get(job.Pipeline.ListenerId)
	if err != nil {
		return nil, err
	}
	listener.PushCtrl(&clientpb.JobCtrl{
		Ctrl: consts.CtrlWebsiteStop,
		Job:  job.ToProtobuf(),
	})
	listener.RemovePipeline(job.Pipeline)
	if err != nil {
		return nil, err
	}
	return &clientpb.Empty{}, nil
}

func (rpc *Server) ListWebsites(ctx context.Context, req *clientpb.Listener) (*clientpb.Pipelines, error) {
	var websites []*clientpb.Pipeline
	pipelines, err := db.ListWebsite(req.Id)
	if err != nil {
		return nil, err
	}
	for _, pipeline := range pipelines {
		websites = append(websites, pipeline.ToProtobuf())
	}
	return &clientpb.Pipelines{Pipelines: websites}, nil
}

func (rpc *Server) DeleteWebsite(ctx context.Context, req *clientpb.CtrlPipeline) (*clientpb.Empty, error) {
	pipelineDB, err := db.FindPipeline(req.Name)
	if err != nil {
		return nil, err
	}
	pipeline := pipelineDB.ToProtobuf()
	listener, err := core.Listeners.Get(pipeline.ListenerId)
	if err != nil {
		return nil, err
	}
	listener.RemovePipeline(pipeline)
	err = db.DeleteWebsite(req.Name)
	if err != nil && !errors.Is(err, db.ErrRecordNotFound) {
		return nil, err
	}

	job, err := core.Jobs.Get(req.Name)
	if err != nil {
		return nil, err
	}

	listener.PushCtrl(&clientpb.JobCtrl{
		Ctrl: consts.CtrlWebsiteStop,
		Job:  job.ToProtobuf(),
	})

	return &clientpb.Empty{}, nil
}

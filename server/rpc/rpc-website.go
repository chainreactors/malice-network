package rpc

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/chainreactors/malice-network/helper/consts"
	"github.com/chainreactors/malice-network/helper/proto/client/clientpb"
	"github.com/chainreactors/malice-network/server/internal/certutils"
	"github.com/chainreactors/malice-network/server/internal/configs"
	"github.com/chainreactors/malice-network/server/internal/core"
	"github.com/chainreactors/malice-network/server/internal/db"
	"github.com/chainreactors/malice-network/server/internal/db/models"
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
func (rpc *Server) WebsiteAddContent(ctx context.Context, req *clientpb.Website) (*clientpb.Empty, error) {
	if len(req.Contents) != 0 {
		for _, content := range req.Contents {
			content.Size = uint64(len(content.Content))
			rpcLog.Infof("Add website content (%s) %s -> %s", content.File, content.Path, content.Type)
			_, err := db.AddContent(content)
			if err != nil {
				return nil, err
			}

			job := core.Jobs.Get(content.WebsiteId)
			if job == nil {
				return nil, fmt.Errorf("website %s not found", req.Name)
			}
			job.Message.(*clientpb.Pipeline).GetWeb().Contents[content.Path] = content
			core.Jobs.Ctrl <- &clientpb.JobCtrl{
				Id:   core.NextCtrlID(),
				Ctrl: consts.CtrlWebContentAdd,
				Job:  job.ToProtobuf(),
			}
		}
	}

	return &clientpb.Empty{}, nil
}

// WebsiteUpdateContent - Update specific content from a website
func (rpc *Server) WebsiteUpdateContent(ctx context.Context, req *clientpb.WebContent) (*clientpb.Empty, error) {
	_, err := db.AddContent(req)
	if err != nil {
		return nil, err
	}

	job := core.Jobs.Get(req.WebsiteId)
	if job == nil {
		return nil, fmt.Errorf("website %s not found", req.WebsiteId)
	}

	core.Jobs.Ctrl <- &clientpb.JobCtrl{
		Id:   core.NextCtrlID(),
		Ctrl: consts.CtrlWebContentUpdate,
		Job:  job.ToProtobuf(),
	}

	return &clientpb.Empty{}, nil
}

// WebsiteRemoveContent - Remove specific content from a website
func (rpc *Server) WebsiteRemoveContent(ctx context.Context, req *clientpb.WebContent) (*clientpb.Empty, error) {
	web, err := db.FindWebContent(req.Id)
	if err != nil {
		return nil, err
	}
	err = db.RemoveContent(req.Id)
	if err != nil {
		return nil, err
	}

	job := core.Jobs.Get(web.PipelineID)
	if job == nil {
		return nil, fmt.Errorf("website %s not found", req.WebsiteId)
	}

	core.Jobs.Ctrl <- &clientpb.JobCtrl{
		Id:   core.NextCtrlID(),
		Ctrl: consts.CtrlWebContentRemove,
		Job:  job.ToProtobuf(),
	}

	return &clientpb.Empty{}, nil
}

func (rpc *Server) RegisterWebsite(ctx context.Context, req *clientpb.Pipeline) (*clientpb.Empty, error) {
	ip := getRemoteAddr(ctx)
	ip = strings.Split(ip, ":")[0]
	pipelineModel := models.FromPipelinePb(req, ip)
	var err error
	if pipelineModel.Tls.Enable && pipelineModel.Tls.Cert == "" && pipelineModel.Tls.Key == "" {
		pipelineModel.Tls.Cert, pipelineModel.Tls.Key, err = certutils.GenerateTlsCert(req.Name, req.ListenerId)
		if err != nil {
			return nil, err
		}
	}
	err = db.CreatePipeline(pipelineModel)
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

	listener := core.Listeners.Get(webpipe.ListenerID)
	if listener == nil {
		return nil, fmt.Errorf("listener %s not found", req.ListenerId)
	}

	webpb := webpipe.ToProtobuf()
	listener.AddPipeline(webpb)
	err = MapContents(webpb)
	if err != nil {
		return nil, err
	}
	webpb.Enable = true
	job := &core.Job{
		ID:      core.NextJobID(),
		Message: webpb,
		Name:    webpipe.Name,
	}
	core.Jobs.Add(job)

	core.Jobs.Ctrl <- &clientpb.JobCtrl{
		Id:   core.NextCtrlID(),
		Ctrl: consts.CtrlWebsiteStart,
		Job:  job.ToProtobuf(),
	}
	err = db.EnablePipeline(webpipe)
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
	pipeline := pipelineDB.ToProtobuf()

	job := core.Jobs.Get(req.Name)
	if job == nil {
		return nil, fmt.Errorf("website %s not found", req.Name)
	}

	ctrl := clientpb.JobCtrl{
		Id:   core.NextCtrlID(),
		Ctrl: consts.CtrlWebsiteStop,
		Job:  job.ToProtobuf(),
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
	listener := core.Listeners.Get(pipeline.ListenerId)
	if listener == nil {
		return nil, fmt.Errorf("listener %s not found", req.ListenerId)
	}
	listener.RemovePipeline(pipeline)

	job := core.Jobs.Get(req.Name)
	if job == nil {
		return nil, fmt.Errorf("website %s not found", req.Name)
	}

	core.Jobs.Ctrl <- &clientpb.JobCtrl{
		Id:   core.NextCtrlID(),
		Ctrl: consts.CtrlWebsiteStop,
		Job:  job.ToProtobuf(),
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

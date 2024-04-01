package rpc

import (
	"context"
	"github.com/chainreactors/malice-network/helper/consts"
	"github.com/chainreactors/malice-network/proto/client/clientpb"
	"github.com/chainreactors/malice-network/server/internal/core"
	"github.com/chainreactors/malice-network/server/internal/db"
	"github.com/chainreactors/malice-network/server/internal/website"
	"mime"
	"path/filepath"
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
	web, err := website.MapContent(req.Name, false)
	if err != nil {
		return nil, err
	}
	err = website.RemoveWebAllContent(web.ID)
	if err != nil {
		return nil, err
	}

	dbWebsite, err := website.WebsiteByName(req.Name)
	if err != nil {
		return nil, err
	}

	err = db.RemoveWebSite(dbWebsite.ID)
	if err != nil {
		return nil, err
	}
	core.EventBroker.Publish(core.Event{
		EventType: consts.EventWebsite,
		Data:      []byte(req.Name),
	})

	return &clientpb.Empty{}, nil
}

// Website - Get one website
func (rpc *Server) Website(ctx context.Context, req *clientpb.Website) (*clientpb.Website, error) {
	return website.MapContent(req.Name, true)
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
			err := website.AddContent(req.Name, content)
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
		Data:      []byte(req.Name),
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
		website.AddContent(dbWebsite.Name, content)
	}

	core.EventBroker.Publish(core.Event{
		EventType: consts.EventWebsite,
		Data:      []byte(req.Name),
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
		Data:      []byte(req.Name),
	})

	return website.MapContent(req.Name, false)
}

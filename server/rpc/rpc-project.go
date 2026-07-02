package rpc

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/chainreactors/IoM-go/proto/client/clientpb"
	"github.com/chainreactors/malice-network/server/internal/db"
	"github.com/chainreactors/malice-network/server/internal/db/models"
	"gorm.io/gorm"
)

func (rpc *Server) CreateProject(ctx context.Context, req *clientpb.CreateProjectRequest) (*clientpb.Project, error) {
	if req.Name == "" {
		return nil, fmt.Errorf("project name cannot be empty")
	}

	existing, err := db.NewProjectQuery().Unscoped().WhereName(req.Name).First()
	if err == nil && existing != nil {
		return nil, fmt.Errorf("project '%s' already exists", req.Name)
	} else if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, fmt.Errorf("failed to check project: %w", err)
	}

	project := &models.Project{
		Name:        req.Name,
		Description: req.Description,
		Note:        req.Note,
	}
	if err := db.Save(project); err != nil {
		return nil, fmt.Errorf("failed to create project: %w", err)
	}
	return project.ToProtobuf(), nil
}

func (rpc *Server) GetProject(ctx context.Context, req *clientpb.Project) (*clientpb.Project, error) {
	var project *models.Project
	var err error

	if req.Id != "" {
		project, err = db.NewProjectQuery().WhereID(req.Id).First()
	} else if req.Name != "" {
		project, err = db.NewProjectQuery().WhereName(req.Name).First()
	} else {
		return nil, fmt.Errorf("project id or name is required")
	}

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("project not found")
		}
		return nil, fmt.Errorf("failed to get project: %w", err)
	}
	return project.ToProtobuf(), nil
}

func (rpc *Server) ListProjects(ctx context.Context, req *clientpb.Empty) (*clientpb.Projects, error) {
	projects, err := db.NewProjectQuery().OrderByCreated().Find()
	if err != nil {
		return nil, fmt.Errorf("failed to list projects: %w", err)
	}
	return projects.ToProtobuf(), nil
}

func (rpc *Server) UpdateProject(ctx context.Context, req *clientpb.UpdateProjectRequest) (*clientpb.Project, error) {
	if req.Id == "" {
		return nil, fmt.Errorf("project id is required")
	}

	_, err := db.NewProjectQuery().WhereID(req.Id).First()
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("project not found")
		}
		return nil, fmt.Errorf("failed to get project: %w", err)
	}

	updates := map[string]interface{}{
		"updated_at": time.Now(),
	}
	if req.Name != "" {
		updates["name"] = req.Name
	}
	if req.Description != "" {
		updates["description"] = req.Description
	}
	if req.Note != "" {
		updates["note"] = req.Note
	}

	if err := db.NewProjectQuery().WhereID(req.Id).Updates(updates); err != nil {
		return nil, fmt.Errorf("failed to update project: %w", err)
	}

	project, err := db.NewProjectQuery().WhereID(req.Id).First()
	if err != nil {
		return nil, fmt.Errorf("failed to get updated project: %w", err)
	}
	return project.ToProtobuf(), nil
}

func (rpc *Server) DeleteProject(ctx context.Context, req *clientpb.DeleteProjectRequest) (*clientpb.Empty, error) {
	if req.Id == "" {
		return nil, fmt.Errorf("project id is required")
	}

	project, err := db.NewProjectQuery().Unscoped().WhereID(req.Id).First()
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("project not found")
		}
		return nil, fmt.Errorf("failed to get project: %w", err)
	}

	if project.Name == "default" {
		return nil, fmt.Errorf("cannot delete the default project")
	}

	if req.Hard {
		if err := db.NewProjectQuery().Unscoped().WhereID(req.Id).Delete(); err != nil {
			return nil, fmt.Errorf("failed to hard-delete project: %w", err)
		}
	} else {
		if err := db.NewProjectQuery().Unscoped().WhereID(req.Id).Updates(map[string]interface{}{
			"is_deleted": true,
			"updated_at": time.Now(),
		}); err != nil {
			return nil, fmt.Errorf("failed to soft-delete project: %w", err)
		}
	}

	return &clientpb.Empty{}, nil
}

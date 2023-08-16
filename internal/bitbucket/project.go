package bitbucket

import (
	"context"
	"fmt"
)

// ProjectService provides operations around bitbucket projects
type ProjectService interface {
	Get(context.Context, *GetProjectRequest) (*Project, error)
	Create(context.Context, *CreateProjectRequest) (*Project, error)
	Update(context.Context, *UpdateProjectRequest) (*Project, error)
	Delete(context.Context, *DeleteProjectRequest) error
}

type projectService struct {
	client *Client
}

// Project represents a Bitbucket Project
type Project struct {
	Name        string `json:"name"`
	Key         string `json:"key"`
	ID          int    `json:"id"`
	Description string `json:"description"`
	Scope       string `json:"scope,omitempty"`
	Type        string `json:"type"`
	Public      bool   `json:"public"`
}

// GetProjectRequest contains the fields required to fetch a project
type GetProjectRequest struct {
	Key string `json:"key"`
}

func (ps *projectService) Get(ctx context.Context, getReq *GetProjectRequest) (*Project, error) {
	req, err := ps.client.newRequest("GET", fmt.Sprintf("projects/%s", getReq.Key), nil)
	if err != nil {
		return nil, fmt.Errorf("error creating request for getting projects: %w", err)
	}

	p := Project{}
	err = ps.client.do(ctx, req, &p)
	if err != nil {
		return nil, fmt.Errorf("error fetching projects: %w", err)
	}
	return &p, nil
}

// CreateProjectRequest contains the fields required to create a project
type CreateProjectRequest struct {
	Name        string `json:"name"`
	Key         string `json:"key"`
	Description string `json:"description,omitempty"`
	Public      bool   `json:"public,omitempty"`
}

func (ps *projectService) Create(ctx context.Context, createReq *CreateProjectRequest) (*Project, error) {
	req, err := ps.client.newRequest("POST", "projects", createReq)
	if err != nil {
		return nil, fmt.Errorf("error creating request for creating project: %w", err)
	}

	p := Project{}
	err = ps.client.do(ctx, req, &p)
	if err != nil {
		return nil, fmt.Errorf("error creating project: %w", err)
	}

	return &p, nil
}

// DeleteProjectRequest contains the fields required to delete a project
type DeleteProjectRequest struct {
	Key string `json:"key"`
}

func (ps *projectService) Delete(ctx context.Context, deleteReq *DeleteProjectRequest) error {
	req, err := ps.client.newRequest("DELETE", fmt.Sprintf("projects/%s", deleteReq.Key), nil)
	if err != nil {
		return fmt.Errorf("error creating request for deleting project: %w", err)
	}

	err = ps.client.do(ctx, req, nil)
	if err != nil {
		return fmt.Errorf("error deleting project: %w", err)
	}

	return nil
}

// UpdateProjectRequest contains the fields required to update a project
type UpdateProjectRequest struct {
	Key         string `json:"key"`
	Description string `json:"description,omitempty"`
	Public      bool   `json:"public,omitempty"`
}

func (ps *projectService) Update(ctx context.Context, updateReq *UpdateProjectRequest) (*Project, error) {
	req, err := ps.client.newRequest("PUT", fmt.Sprintf("projects/%s", updateReq.Key), updateReq)
	if err != nil {
		return nil, fmt.Errorf("error creating request for updating project: %w", err)
	}

	p := Project{}
	err = ps.client.do(ctx, req, &p)
	if err != nil {
		return nil, fmt.Errorf("error updating project: %w", err)
	}

	return &p, nil
}

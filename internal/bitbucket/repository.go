package bitbucket

import (
	"context"
	"fmt"
	"net/http"
)

type RepositoryService interface {
	Get(context.Context, *Repository) (*Repository, error)
	Create(context.Context, *Repository) (*Repository, error)
	Update(context.Context, *Repository) (*Repository, error)
	Delete(context.Context, *Repository) error
	// Groups permissions
	GetGroups(context.Context, *Repository) ([]Group, error)
	AddGroup(context.Context, *Repository, *Group) error
	RevokeGroup(context.Context, *Repository, *Group) error
}

type repositoryService struct {
	client *Client
}

type Repository struct {
	ID          int    `json:"-"`
	Name        string `json:"name"`
	Public      bool   `json:"public"`
	Project     string `json:"-"`
	Description string `json:"description"`
}

type Group struct {
	Name       string
	Permission string
}

type repositoryJson struct {
	ID      int    `json:"id"`
	Name    string `json:"name"`
	Project struct {
		Key string `json:"key"`
	} `json:"project"`
	Description string `json:"description"`
}

func (service *repositoryService) Get(ctx context.Context, repository *Repository) (*Repository, error) {
	req, err := service.client.newRequest(http.MethodGet, fmt.Sprintf("projects/%s/repos/%s", repository.Project, repository.Name), nil)
	if err != nil {
		return nil, fmt.Errorf("error creating request for getting repository: %w", err)
	}

	var repo repositoryJson
	err = service.client.do(ctx, req, &repo)
	if err != nil {
		return nil, fmt.Errorf("error fetching repository: %w", err)
	}
	return repo.toRepository(), nil
}

func (service *repositoryService) Create(ctx context.Context, repository *Repository) (*Repository, error) {
	req, err := service.client.newRequest(http.MethodPost, fmt.Sprintf("projects/%s/repos", repository.Project), repository)
	if err != nil {
		return nil, fmt.Errorf("error creating request for creating repository: %w", err)
	}

	var repo repositoryJson
	err = service.client.do(ctx, req, &repo)
	if err != nil {
		return nil, fmt.Errorf("error creating repository: %w", err)
	}

	return repo.toRepository(), nil
}

func (service *repositoryService) Update(ctx context.Context, repository *Repository) (*Repository, error) {
	req, err := service.client.newRequest(http.MethodPut, fmt.Sprintf("projects/%s/repos/%s", repository.Project, repository.Name), repository)
	if err != nil {
		return nil, fmt.Errorf("error updating request for creating repository: %w", err)
	}

	var repo repositoryJson
	err = service.client.do(ctx, req, &repo)
	if err != nil {
		return nil, fmt.Errorf("error updating repository: %w", err)
	}

	return repo.toRepository(), nil
}

func (service *repositoryService) Delete(ctx context.Context, repository *Repository) error {
	req, err := service.client.newRequest(http.MethodDelete, fmt.Sprintf("projects/%s/repos/%s", repository.Project, repository.Name), nil)
	if err != nil {
		return fmt.Errorf("error creating request for deleting repository: %w", err)
	}

	err = service.client.do(ctx, req, nil)
	if err != nil {
		return fmt.Errorf("error deleting repository: %w", err)
	}

	return nil
}

func (service *repositoryService) GetGroups(ctx context.Context, repository *Repository) ([]Group, error) {
	url := fmt.Sprintf("projects/%s/repos/%s/permissions/groups", repository.Project, repository.Name)
	req, err := service.client.newRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("error creating request for getting repository groups: %w", err)
	}

	var response struct {
		Values []struct {
			Group struct {
				Name string `json:"name"`
			} `json:"group"`
			Permission string `json:"permission"`
		} `json:"values"`
	}
	err = service.client.do(ctx, req, &response)
	if err != nil {
		return nil, fmt.Errorf("error getting repository group: %w", err)
	}

	groups := []Group{}
	for _, entry := range response.Values {
		groups = append(groups, Group{Name: entry.Group.Name, Permission: entry.Permission})
	}
	return groups, nil
}

func (service *repositoryService) AddGroup(ctx context.Context, repository *Repository, group *Group) error {
	url := fmt.Sprintf("projects/%s/repos/%s/permissions/groups?name=%s&permission=%s", repository.Project, repository.Name, group.Name, group.Permission)
	req, err := service.client.newRequest(http.MethodPut, url, nil)
	if err != nil {
		return fmt.Errorf("error creating request for adding repository group: %w", err)
	}

	err = service.client.do(ctx, req, nil)
	if err != nil {
		return fmt.Errorf("error adding repository group: %w", err)
	}
	return nil
}

func (service *repositoryService) RevokeGroup(ctx context.Context, repository *Repository, group *Group) error {
	url := fmt.Sprintf("projects/%s/repos/%s/permissions/groups?name=%s", repository.Project, repository.Name, group.Name)
	req, err := service.client.newRequest(http.MethodDelete, url, nil)
	if err != nil {
		return fmt.Errorf("error creating request for revoking repository group: %w", err)
	}

	err = service.client.do(ctx, req, nil)
	if err != nil {
		return fmt.Errorf("error revoking repository group: %w", err)
	}
	return nil
}

func (r *repositoryJson) toRepository() *Repository {
	return &Repository{ID: r.ID, Name: r.Name, Project: r.Project.Key, Description: r.Description}
}

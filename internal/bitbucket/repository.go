package bitbucket

import (
	"context"
	"fmt"
)

type RepositoryService interface {
	Get(context.Context, *Repository) (*Repository, error)
	Create(context.Context, *Repository) (*Repository, error)
	Update(context.Context, *Repository) (*Repository, error)
	Delete(context.Context, *Repository) error
}

type repositoryService struct {
	client *Client
}

type Repository struct {
	ID          int    `json:"-"`
	Name        string `json:"name"`
	Project     string `json:"-"`
	Description string `json:"description"`
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
	req, err := service.client.newRequest("GET", fmt.Sprintf("projects/%s/repos/%s", repository.Project, repository.Name), nil)
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
	req, err := service.client.newRequest("POST", fmt.Sprintf("projects/%s/repos", repository.Project), repository)
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
	req, err := service.client.newRequest("PUT", fmt.Sprintf("projects/%s/repos/%s", repository.Project, repository.Name), repository)
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
	req, err := service.client.newRequest("PUT", fmt.Sprintf("projects/%s/repos/%s", repository.Project, repository.Name), nil)
	if err != nil {
		return fmt.Errorf("error creating request for deleting repository: %w", err)
	}

	err = service.client.do(ctx, req, nil)
	if err != nil {
		return fmt.Errorf("error deleting repository: %w", err)
	}

	return nil
}

func (r *repositoryJson) toRepository() *Repository {
	return &Repository{ID: r.ID, Name: r.Name, Project: r.Project.Key, Description: r.Description}
}

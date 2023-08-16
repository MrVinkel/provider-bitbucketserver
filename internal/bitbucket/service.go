package bitbucket

type BitBucketService struct {
	Projects     ProjectService
	Repositories RepositoryService
}

func NewService(client *Client) (*BitBucketService, error) {
	service := BitBucketService{
		Projects:     &projectService{client: client},
		Repositories: &repositoryService{client: client},
	}
	return &service, nil
}

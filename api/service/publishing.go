package service

type PublishingService interface {
	GetPublishingRepositories() (string, error)
}

func NewPublishingService() *publishingService {
	return &publishingService{}
}

type publishingService struct {
}

func (s *publishingService) GetPublishingRepositories() (string, error) {
	return "PublishingRepositories", nil
}

package service

import (
	"fmt"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/pennsieve/publishing-service/api/models"
	"github.com/pennsieve/publishing-service/api/store"
)

type PublishingService interface {
	GetPublishingRepositories() ([]models.Repository, error)
}

func NewPublishingService(store store.PublishingStore) *publishingService {
	return &publishingService{
		store: store,
	}
}

type publishingService struct {
	store store.PublishingStore
}

func (s *publishingService) GetPublishingRepositories() ([]models.Repository, error) {
	output, err := s.store.GetRepositories()

	if err != nil {
		return nil, err
	}

	var items []models.Repository
	for _, item := range output.Items {
		repository := models.Repository{}
		err = attributevalue.UnmarshalMap(item, &repository)
		if err != nil {
			return nil, fmt.Errorf("UnmarshalMap: %v\n", err)
		}
		items = append(items, repository)
	}

	return items, nil
}

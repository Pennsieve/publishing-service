package service

import (
	"fmt"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/pennsieve/publishing-service/api/models"
	"github.com/pennsieve/publishing-service/api/store"
	log "github.com/sirupsen/logrus"
)

type PublishingService interface {
	GetPublishingRepositories() ([]models.Repository, []models.Question, error)
}

func NewPublishingService(store store.PublishingStore) *publishingService {
	return &publishingService{
		store: store,
	}
}

type publishingService struct {
	store store.PublishingStore
}

func (s *publishingService) GetPublishingRepositories() ([]models.Repository, []models.Question, error) {
	log.Println("GetPublishingRepositories()")
	var err error

	output, err := s.store.GetRepositories()
	if err != nil {
		log.Fatalln("GetPublishingRepositories() store.GetRepositories() err: ", err)
		return nil, nil, err
	}

	var items []models.Repository
	for _, item := range output.Items {
		repository := models.Repository{}
		err = attributevalue.UnmarshalMap(item, &repository)
		if err != nil {
			return nil, nil, fmt.Errorf("UnmarshalMap: %v\n", err)
		}
		items = append(items, repository)
	}

	output2, err := s.store.GetQuestions()
	if err != nil {
		log.Fatalln("GetPublishingRepositories() store.GetQuestions() err: ", err)
		return nil, nil, err
	}

	var items2 []models.Question
	for _, item := range output2.Items {
		question := models.Question{}
		err = attributevalue.UnmarshalMap(item, &question)
		if err != nil {
			return nil, nil, fmt.Errorf("UnmarshalMap: %v\n", err)
		}
		items2 = append(items2, question)
	}

	return items, items2, nil
}

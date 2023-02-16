package service

import (
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

	repositories, err := s.store.GetRepositories()
	if err != nil {
		log.Fatalln("GetPublishingRepositories() store.GetRepositories() err: ", err)
		return nil, nil, err
	}

	questions, err := s.store.GetQuestions()
	if err != nil {
		log.Fatalln("GetPublishingRepositories() store.GetQuestions() err: ", err)
		return nil, nil, err
	}

	// TODO: create RepositoryDTO from repositories and questions

	return repositories, questions, nil
}

package service

import (
	"github.com/pennsieve/publishing-service/api/dtos"
	"github.com/pennsieve/publishing-service/api/store"
	log "github.com/sirupsen/logrus"
)

type PublishingService interface {
	GetPublishingRepositories() ([]dtos.RepositoryDTO, error)
	GetProposalQuestions() ([]dtos.QuestionDTO, error)
}

func NewPublishingService(store store.PublishingStore) *publishingService {
	return &publishingService{
		store: store,
	}
}

type publishingService struct {
	store store.PublishingStore
}

func (s *publishingService) GetPublishingRepositories() ([]dtos.RepositoryDTO, error) {
	log.Println("GetPublishingRepositories()")
	var err error

	repositories, err := s.store.GetRepositories()
	if err != nil {
		log.Fatalln("GetPublishingRepositories() store.GetRepositories() err: ", err)
		return nil, err
	}

	questions, err := s.store.GetQuestions()
	if err != nil {
		log.Fatalln("GetPublishingRepositories() store.GetQuestions() err: ", err)
		return nil, err
	}

	// create a Questions lookup map indexed by Id number
	var questionMap = make(map[int]dtos.QuestionDTO)
	for i := 0; i < len(questions); i++ {
		questionMap[questions[i].Id] = dtos.BuildQuestionDTO(questions[i])
	}

	// TODO: create RepositoryDTO from repositories and questions
	var repositoryDTOs []dtos.RepositoryDTO
	for i := 0; i < len(repositories); i++ {
		repositoryDTOs = append(repositoryDTOs, dtos.BuildRepositoryDTO(repositories[i], questionMap))
	}
	return repositoryDTOs, nil
}

func (s *publishingService) GetProposalQuestions() ([]dtos.QuestionDTO, error) {
	log.Println("GetProposalQuestions()")
	var err error

	questions, err := s.store.GetQuestions()
	if err != nil {
		log.Fatalln("GetProposalQuestions() store.GetQuestions() err: ", err)
		return nil, err
	}

	var questionDTOs []dtos.QuestionDTO
	for i := 0; i < len(questions); i++ {
		questionDTOs = append(questionDTOs, dtos.QuestionDTO{
			Id:       questions[i].Id,
			Question: questions[i].Question,
		})
	}

	return questionDTOs, nil
}

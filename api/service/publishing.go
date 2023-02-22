package service

import (
	"fmt"
	"github.com/google/uuid"
	"github.com/pennsieve/publishing-service/api/dtos"
	"github.com/pennsieve/publishing-service/api/models"
	"github.com/pennsieve/publishing-service/api/store"
	log "github.com/sirupsen/logrus"
)

type PublishingService interface {
	GetPublishingRepositories() ([]dtos.RepositoryDTO, error)
	GetProposalQuestions() ([]dtos.QuestionDTO, error)
	GetDatasetProposalsForUser(id int64) ([]dtos.DatasetProposalDTO, error)
	GetDatasetProposalsForWorkspace(id int64) ([]dtos.DatasetProposalDTO, error)
	CreateDatasetProposal(userId int, dto dtos.DatasetProposalDTO) (*dtos.DatasetProposalDTO, error)
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

func proposalDTOsList(proposals []models.DatasetProposal) []dtos.DatasetProposalDTO {
	var proposalDTOs []dtos.DatasetProposalDTO
	for i := 0; i < len(proposals); i++ {
		proposalDTOs = append(proposalDTOs, dtos.BuildDatasetProposalDTO(proposals[i]))
	}
	return proposalDTOs
}

func (s *publishingService) GetDatasetProposalsForUser(id int64) ([]dtos.DatasetProposalDTO, error) {
	log.Println("GetProposalQuestions()")

	proposals, err := s.store.GetDatasetProposalsForUser(id)
	if err != nil {
		return nil, err
	}

	return proposalDTOsList(proposals), nil
}

func (s *publishingService) GetDatasetProposalsForWorkspace(id int64) ([]dtos.DatasetProposalDTO, error) {
	log.Println("GetProposalQuestions()")

	proposals, err := s.store.GetDatasetProposalsForWorkspace(id)
	if err != nil {
		return nil, err
	}

	return proposalDTOsList(proposals), nil
}

func (s *publishingService) CreateDatasetProposal(userId int, dto dtos.DatasetProposalDTO) (*dtos.DatasetProposalDTO, error) {
	log.Println("service.CreateDatasetProposal()")

	var survey []models.Survey
	for i := 0; i < len(dto.Survey); i++ {
		survey = append(survey, dtos.BuildSurvey(dto.Survey[i]))
	}

	proposal := &models.DatasetProposal{
		UserId:         userId,
		ProposalNodeId: fmt.Sprintf("%s:%s:%s", "N", "proposal", uuid.NewString()),
		Name:           dto.Name,
		Description:    dto.Description,
		RepositoryId:   dto.RepositoryId,
		Status:         "DRAFT",
		Survey:         survey,
	}
	log.Println("service.CreateDatasetProposal() proposal: ", proposal)

	result, err := s.store.CreateDatasetProposal(proposal)
	if err != nil {
		log.Fatalln("service.CreateDatasetProposal() - store.CreateDatasetProposal() failed: ", err)
		return nil, err
	}

	dtoResult := dtos.BuildDatasetProposalDTO(*result)
	return &dtoResult, nil
}

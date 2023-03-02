package service

import (
	"fmt"
	"github.com/google/uuid"
	"github.com/pennsieve/publishing-service/api/dtos"
	"github.com/pennsieve/publishing-service/api/models"
	"github.com/pennsieve/publishing-service/api/store"
	log "github.com/sirupsen/logrus"
	"time"
)

type PublishingService interface {
	GetPublishingRepositories() ([]dtos.RepositoryDTO, error)
	GetProposalQuestions() ([]dtos.QuestionDTO, error)
	GetDatasetProposal(userId int, nodeId string) (dtos.DatasetProposalDTO, error)
	GetDatasetProposalsForUser(id int64) ([]dtos.DatasetProposalDTO, error)
	GetDatasetProposalsForWorkspace(id int64) ([]dtos.DatasetProposalDTO, error)
	CreateDatasetProposal(userId int, dto dtos.DatasetProposalDTO) (*dtos.DatasetProposalDTO, error)
	UpdateDatasetProposal(userId int, dto dtos.DatasetProposalDTO) (*dtos.DatasetProposalDTO, error)
	DeleteDatasetProposal(proposal dtos.DatasetProposalDTO) (bool, error)
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

func (s *publishingService) GetDatasetProposal(userId int, nodeId string) (dtos.DatasetProposalDTO, error) {
	log.WithFields(log.Fields{"userId": userId, "nodeId": nodeId}).Info("service.GetDatasetProposal()")

	proposal, err := s.store.GetDatasetProposal(userId, nodeId)
	if err != nil {
		// TODO: fix this, we should not return anything for the value
		return dtos.DatasetProposalDTO{}, err
	}

	proposalDTO := dtos.BuildDatasetProposalDTO(proposal)

	return proposalDTO, nil
}

func (s *publishingService) GetDatasetProposalsForUser(userId int64) ([]dtos.DatasetProposalDTO, error) {
	log.WithFields(log.Fields{"userId": userId}).Info("service.GetDatasetProposalsForUser()")

	proposals, err := s.store.GetDatasetProposalsForUser(userId)
	if err != nil {
		log.Error("store.GetDatasetProposalsForUser() failed: ", err)
		return nil, err
	}

	return proposalDTOsList(proposals), nil
}

func (s *publishingService) GetDatasetProposalsForWorkspace(id int64) ([]dtos.DatasetProposalDTO, error) {
	log.WithFields(log.Fields{"id": id}).Info("service.GetDatasetProposalsForWorkspace()")

	proposals, err := s.store.GetDatasetProposalsForWorkspace(id)
	if err != nil {
		return nil, err
	}

	return proposalDTOsList(proposals), nil
}

// TODO: validate RepositoryId, ensure it is in Repositories table
// TODO: move generating ProposalNodeId string elsewhere (pennsieve-core?)
// TODO: refactor Create..() and Update..() to use common code
func (s *publishingService) CreateDatasetProposal(userId int, dto dtos.DatasetProposalDTO) (*dtos.DatasetProposalDTO, error) {
	log.Println("service.CreateDatasetProposal()")

	var survey []models.Survey
	for i := 0; i < len(dto.Survey); i++ {
		survey = append(survey, dtos.BuildSurvey(dto.Survey[i]))
	}

	var contributors []models.Contributor
	for i := 0; i < len(dto.Contributors); i++ {
		contributors = append(contributors, dtos.BuildContributor(dto.Contributors[i]))
	}

	currentTime := time.Now().Unix()

	proposal := &models.DatasetProposal{
		UserId:             userId,
		ProposalNodeId:     fmt.Sprintf("%s:%s:%s", "N", "proposal", uuid.NewString()),
		Name:               dto.Name,
		Description:        dto.Description,
		RepositoryId:       dto.RepositoryId,
		OrganizationNodeId: dto.OrganizationNodeId,
		Status:             "DRAFT",
		Survey:             survey,
		Contributors:       contributors,
		CreatedAt:          currentTime,
		UpdatedAt:          currentTime,
	}
	log.WithFields(log.Fields{"proposal": fmt.Sprintf("%+v", proposal)}).Debug("service.CreateDatasetProposal()")

	_, err := s.store.CreateDatasetProposal(proposal)
	if err != nil {
		log.Fatalln("service.CreateDatasetProposal() - store.CreateDatasetProposal() failed: ", err)
		return nil, err
	}

	dtoResult := dtos.BuildDatasetProposalDTO(*proposal)
	return &dtoResult, nil
}

func (s *publishingService) UpdateDatasetProposal(userId int, dto dtos.DatasetProposalDTO) (*dtos.DatasetProposalDTO, error) {
	log.Println("service.UpdateDatasetProposal()")

	var survey []models.Survey
	for i := 0; i < len(dto.Survey); i++ {
		survey = append(survey, dtos.BuildSurvey(dto.Survey[i]))
	}

	currentTime := time.Now().Unix()

	proposal := &models.DatasetProposal{
		UserId:             userId,
		ProposalNodeId:     dto.ProposalNodeId,
		Name:               dto.Name,
		Description:        dto.Description,
		RepositoryId:       dto.RepositoryId,
		OrganizationNodeId: dto.OrganizationNodeId,
		Status:             dto.Status,
		Survey:             survey,
		CreatedAt:          dto.CreatedAt,
		UpdatedAt:          currentTime,
	}
	log.WithFields(log.Fields{"proposal": fmt.Sprintf("%+v", proposal)}).Debug("service.UpdateDatasetProposal()")

	_, err := s.store.UpdateDatasetProposal(proposal)
	if err != nil {
		log.Fatalln("store.UpdateDatasetProposal() failed: ", err)
		return nil, err
	}

	dtoResult := dtos.BuildDatasetProposalDTO(*proposal)
	return &dtoResult, nil
}

func (s *publishingService) DeleteDatasetProposal(proposalDTO dtos.DatasetProposalDTO) (bool, error) {
	log.WithFields(log.Fields{"proposalDTO": fmt.Sprintf("%+v", proposalDTO)}).Info("service.DeleteDatasetProposal()")

	proposal := dtos.BuildDatasetProposal(proposalDTO)

	err := s.store.DeleteDatasetProposal(proposal)
	if err != nil {
		log.Fatalln("store.DeleteDatasetProposal() failed: ", err)
		return false, err
	}

	return true, nil
}

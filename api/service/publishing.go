package service

import (
	"context"
	"fmt"
	"github.com/google/uuid"
	pgdbModels "github.com/pennsieve/pennsieve-go-core/pkg/models/pgdb"
	"github.com/pennsieve/publishing-service/api/aws/ses"
	sesTypes "github.com/pennsieve/publishing-service/api/aws/ses/types"
	"github.com/pennsieve/publishing-service/api/dtos"
	"github.com/pennsieve/publishing-service/api/models"
	"github.com/pennsieve/publishing-service/api/notification"
	"github.com/pennsieve/publishing-service/api/store"
	log "github.com/sirupsen/logrus"
	"os"
	"time"
)

type PublishingService interface {
	GetPublishingInfo() ([]dtos.InfoDTO, error)
	GetPublishingRepositories() ([]dtos.RepositoryDTO, error)
	GetProposalQuestions() ([]dtos.QuestionDTO, error)
	GetDatasetProposal(userId int, nodeId string) (dtos.DatasetProposalDTO, error)
	GetDatasetProposalsForUser(id int64) ([]dtos.DatasetProposalDTO, error)
	GetDatasetProposalsForWorkspace(orgNodeId string, status string) ([]dtos.DatasetProposalDTO, error)
	CreateDatasetProposal(userId int64, dto dtos.DatasetProposalDTO) (*dtos.DatasetProposalDTO, error)
	UpdateDatasetProposal(userId int64, existing dtos.DatasetProposalDTO, dto dtos.DatasetProposalDTO) (*dtos.DatasetProposalDTO, error)
	DeleteDatasetProposal(proposal dtos.DatasetProposalDTO) (bool, error)
	SubmitDatasetProposal(userId int, nodeId string) (*dtos.DatasetProposalDTO, error)
	WithdrawDatasetProposal(userId int, nodeId string) (*dtos.DatasetProposalDTO, error)
	AcceptDatasetProposal(repositoryId int, nodeId string) (*dtos.DatasetProposalDTO, error)
	RejectDatasetProposal(repositoryId int, nodeId string) (*dtos.DatasetProposalDTO, error)
}

func NewPublishingService(pubStore store.PublishingStore, pennsieve store.PennsievePublishingStore, notifier notification.Notifier) *publishingService {
	return &publishingService{
		store:     pubStore,
		pennsieve: pennsieve,
		notifier:  notifier,
	}
}

type publishingService struct {
	store     store.PublishingStore
	pennsieve store.PennsievePublishingStore
	notifier  notification.Notifier
}

func usersName(user *pgdbModels.User) string {
	return fmt.Sprintf("%s %s", user.FirstName, user.LastName)
}

func sendEmail(ctx context.Context, sender string, recipients []string, subject string, body string) error {
	// send email message
	emailAgent := ses.MakeEmailer()
	err := emailAgent.SendMessage(ctx, sender, recipients, subject, body, sesTypes.Text)
	if err != nil {
		log.WithFields(log.Fields{"error": fmt.Sprintf("%+v", err)}).Error("service.sendEmail()")
	}
	return err
}

func (s *publishingService) notifyPublishingTeam(proposal *models.DatasetProposal, action notification.Notification, repository *models.Repository) error {
	log.WithFields(log.Fields{"proposal": fmt.Sprintf("%+v", proposal), "action": action, "repository": fmt.Sprintf("%+v", repository)}).Info("service.notifyPublishingTeam()")

	ctx := context.TODO()

	// get Publishing team for the Repository
	publishers, err := s.pennsieve.GetPublishingTeam(ctx, repository)
	if err != nil {
		log.WithFields(log.Fields{"failed": "GetPublishingTeam()", "error": fmt.Sprintf("%+v", err)}).Error("service.notifyPublishingTeam()")
		return err
	}
	log.WithFields(log.Fields{"publishers": fmt.Sprintf("%+v", publishers)}).Info("service.notifyPublishingTeam()")

	// build list of Publisher's email addresses
	var recipients []string
	for _, publisher := range publishers {
		// TODO: make sure email address is not null and not the empty string
		recipients = append(recipients, publisher.UserEmailAddress)
	}

	messageAttributes := notification.MessageAttributes{
		"AppURL":          fmt.Sprintf("app.%s", os.Getenv("PENNSIEVE_DOMAIN")),
		"AuthorName":      proposal.OwnerName,
		"AuthorEmail":     proposal.EmailAddress,
		"ProposalTitle":   proposal.Name,
		"WorkspaceName":   repository.DisplayName,
		"WorkspaceNodeId": repository.OrganizationNodeId,
	}

	switch action {
	case notification.Submitted:
		err = s.notifier.ProposalSubmitted(messageAttributes, recipients)
	case notification.Withdrawn:
		err = s.notifier.ProposalWithdrawn(messageAttributes, recipients)
	}

	return err
}

func (s *publishingService) notifyProposalOwner(proposal *models.DatasetProposal, action notification.Notification, repository *models.Repository) error {
	log.WithFields(log.Fields{"proposal": fmt.Sprintf("%+v", proposal), "action": action, "repository": fmt.Sprintf("%+v", repository)}).Info("service.notifyProposalOwner()")

	ctx := context.TODO()

	// lookup the Welcome Workspace
	welcomeWorkspace, err := s.pennsieve.GetWelcomeWorkspace(ctx)
	if err != nil {
		log.WithFields(log.Fields{"error": fmt.Sprintf("%+v", err)}).Error("service.notifyProposalOwner()")
		return err
	}

	// the recipients are just the proposal owner/author
	var recipients []string
	recipients = append(recipients, proposal.EmailAddress)

	messageAttributes := notification.MessageAttributes{
		"AppURL":                 fmt.Sprintf("app.%s", os.Getenv("PENNSIEVE_DOMAIN")),
		"AuthorName":             proposal.OwnerName,
		"AuthorEmail":            proposal.EmailAddress,
		"ProposalTitle":          proposal.Name,
		"WorkspaceName":          repository.DisplayName,
		"WorkspaceNodeId":        repository.OrganizationNodeId,
		"WelcomeWorkspaceNodeId": welcomeWorkspace.NodeId,
	}

	switch action {
	case notification.Accepted:
		err = s.notifier.ProposalAccepted(messageAttributes, recipients)
	case notification.Rejected:
		err = s.notifier.ProposalRejected(messageAttributes, recipients)
	}

	return err
}

func (s *publishingService) GetPublishingInfo() ([]dtos.InfoDTO, error) {
	log.Println("GetPublishingInfo()")
	var err error

	info, err := s.store.GetInfo()
	if err != nil {
		log.Fatalln("GetPublishingInfo() store.GetInfo() err: ", err)
		return nil, err
	}

	var infoDTOs []dtos.InfoDTO
	for i := 0; i < len(info); i++ {
		infoDTOs = append(infoDTOs, dtos.BuildInfoDTO(info[i]))
	}

	return infoDTOs, nil
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
		proposalDTOs = append(proposalDTOs, dtos.BuildDatasetProposalDTO(&proposals[i]))
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

func (s *publishingService) GetDatasetProposalsForWorkspace(orgNodeId string, status string) ([]dtos.DatasetProposalDTO, error) {
	log.WithFields(log.Fields{"orgNodeId": orgNodeId, "status": status}).Info("service.GetDatasetProposalsForWorkspace()")

	// TODO: verify that status is one of: SUBMITTED, ACCEPTED, REJECTED

	proposals, err := s.store.GetDatasetProposalsForWorkspace(orgNodeId, status)
	if err != nil {
		return nil, err
	}

	return proposalDTOsList(proposals), nil
}

// TODO: validate RepositoryId, ensure it is in Repositories table
// TODO: move generating ProposalNodeId string elsewhere (pennsieve-core?)
// TODO: refactor Create..() and Update..() to use common code
func (s *publishingService) CreateDatasetProposal(userId int64, dto dtos.DatasetProposalDTO) (*dtos.DatasetProposalDTO, error) {
	log.Println("service.CreateDatasetProposal()")

	user, err := s.pennsieve.GetProposalUser(context.TODO(), userId)
	if err != nil {
		log.WithFields(log.Fields{"failure": "pennsieve.GetProposalUser()", "error": fmt.Sprintf("%+v", err)}).Error("service.CreateDatasetProposal()")
		return nil, err
	}

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
		UserId:             int(user.Id),
		NodeId:             fmt.Sprintf("%s:%s:%s", "N", "proposal", uuid.NewString()),
		OwnerName:          usersName(user),
		EmailAddress:       user.Email,
		Name:               dto.Name,
		Description:        dto.Description,
		OrganizationNodeId: dto.OrganizationNodeId,
		ProposalStatus:     "DRAFT",
		Survey:             survey,
		Contributors:       contributors,
		CreatedAt:          currentTime,
		UpdatedAt:          currentTime,
	}
	log.WithFields(log.Fields{"proposal": fmt.Sprintf("%+v", proposal)}).Debug("service.CreateDatasetProposal()")

	_, err = s.store.CreateDatasetProposal(proposal)
	if err != nil {
		log.Fatalln("service.CreateDatasetProposal() - store.CreateDatasetProposal() failed: ", err)
		return nil, err
	}

	dtoResult := dtos.BuildDatasetProposalDTO(proposal)
	return &dtoResult, nil
}

func (s *publishingService) UpdateDatasetProposal(userId int64, existing dtos.DatasetProposalDTO, update dtos.DatasetProposalDTO) (*dtos.DatasetProposalDTO, error) {
	log.WithFields(log.Fields{"userId": userId, "existing": fmt.Sprintf("%+v", existing), "update": fmt.Sprintf("%+v", update)}).Info("service.UpdateDatasetProposal()")

	user, err := s.pennsieve.GetProposalUser(context.TODO(), userId)
	if err != nil {
		log.WithFields(log.Fields{"failure": "pennsieve.GetProposalUser()", "error": fmt.Sprintf("%+v", err)}).Error("service.UpdateDatasetProposal()")
		return nil, err
	}

	var survey []models.Survey
	for i := 0; i < len(update.Survey); i++ {
		survey = append(survey, dtos.BuildSurvey(update.Survey[i]))
	}

	var contributors []models.Contributor
	for i := 0; i < len(update.Contributors); i++ {
		contributors = append(contributors, dtos.BuildContributor(update.Contributors[i]))
	}

	currentTime := time.Now().Unix()

	updated := &models.DatasetProposal{
		UserId:             int(user.Id),
		NodeId:             existing.NodeId,
		OwnerName:          usersName(user),
		EmailAddress:       user.Email,
		Name:               update.Name,
		Description:        update.Description,
		OrganizationNodeId: existing.OrganizationNodeId,
		ProposalStatus:     existing.ProposalStatus,
		Survey:             survey,
		Contributors:       contributors,
		CreatedAt:          existing.CreatedAt,
		UpdatedAt:          currentTime,
	}
	log.WithFields(log.Fields{"updated": fmt.Sprintf("%+v", updated)}).Debug("service.UpdateDatasetProposal()")

	_, err = s.store.UpdateDatasetProposal(updated)
	if err != nil {
		log.Fatalln("store.UpdateDatasetProposal() failed: ", err)
		return nil, err
	}

	dtoResult := dtos.BuildDatasetProposalDTO(updated)
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

func (s *publishingService) SubmitDatasetProposal(userId int, nodeId string) (*dtos.DatasetProposalDTO, error) {
	log.WithFields(log.Fields{"userId": userId, "nodeId": nodeId}).Info("service.SubmitDatasetProposal()")

	// get Dataset Proposal by User Id and Node Id
	proposal, err := s.store.GetDatasetProposal(userId, nodeId)
	if err != nil {
		return nil, err
	}
	log.WithFields(log.Fields{"proposal": fmt.Sprintf("%+v", proposal)}).Debug("service.SubmitDatasetProposal()")

	// verify that the Dataset Proposal Status is “DRAFT”
	if proposal.ProposalStatus != "DRAFT" {
		return nil, fmt.Errorf("invalid action: proposal.status must be DRAFT in order to submit")
	}

	// get the Repository using the Organization Node Id on the Dataset Proposal
	repository, err := s.store.GetRepository(proposal.OrganizationNodeId)

	// verify that Organization NodeId is the same on the Repository and the Dataset Proposal (extra check)
	if proposal.OrganizationNodeId != repository.OrganizationNodeId {
		return nil, fmt.Errorf("invalid state: OrganizationNodeId on proposal does not match the Repository")
	}

	// ensure that all Repository Questions are answered in the Dataset Proposal Survey
	// TODO: refactor this
	ok := true
	for _, repositoryQuestionId := range repository.Questions {
		answered := false
		for _, surveyQuestion := range proposal.Survey {
			if surveyQuestion.QuestionId == repositoryQuestionId {
				answered = true
			}
		}
		if !answered {
			ok = false
		}
	}
	if !ok {
		return nil, fmt.Errorf("invalid request: all Repository questions have not been answered")
	}

	// update Dataset Proposal
	currentTime := time.Now().Unix()
	submitted := proposal
	submitted.ProposalStatus = "SUBMITTED"
	submitted.UpdatedAt = currentTime
	submitted.SubmittedAt = currentTime

	updated, err := s.store.UpdateDatasetProposal(submitted)
	if err != nil {
		return nil, err
	}

	// send email to Repository Publishers Team
	log.WithFields(log.Fields{"notify": "publishers"}).Info("service.SubmitDatasetProposal()")
	err = s.notifyPublishingTeam(submitted, notification.Submitted, repository)
	if err != nil {
		log.WithFields(log.Fields{"notifyStatus": "error", "error": fmt.Sprintf("%+v", err)}).Error("service.SubmitDatasetProposal()")
	}

	dtoResult := dtos.BuildDatasetProposalDTO(updated)
	return &dtoResult, nil
}

func (s *publishingService) WithdrawDatasetProposal(userId int, nodeId string) (*dtos.DatasetProposalDTO, error) {
	log.WithFields(log.Fields{"userId": userId, "nodeId": nodeId}).Info("service.WithdrawDatasetProposal()")

	// get Dataset Proposal by User Id and Node Id
	proposal, err := s.store.GetDatasetProposal(userId, nodeId)
	if err != nil {
		return nil, err
	}
	log.WithFields(log.Fields{"proposal": fmt.Sprintf("%+v", proposal)}).Debug("service.WithdrawDatasetProposal()")

	// verify that the Dataset Proposal Status is “SUBMITTED”
	if proposal.ProposalStatus != "SUBMITTED" {
		return nil, fmt.Errorf("invalid action: proposal.status must be SUBMITTED in order to withdraw")
	}

	// get the Repository using the Organization Node Id on the Dataset Proposal
	repository, err := s.store.GetRepository(proposal.OrganizationNodeId)

	// update Dataset Proposal
	currentTime := time.Now().Unix()
	withdrawn := proposal
	withdrawn.ProposalStatus = "WITHDRAWN"
	withdrawn.UpdatedAt = currentTime
	withdrawn.WithdrawnAt = currentTime

	updated, err := s.store.UpdateDatasetProposal(withdrawn)
	if err != nil {
		return nil, err
	}

	// send email to Repository Publishers Team
	log.WithFields(log.Fields{"notify": "publishers"}).Info("service.WithdrawDatasetProposal()")
	err = s.notifyPublishingTeam(withdrawn, notification.Withdrawn, repository)
	if err != nil {
		log.WithFields(log.Fields{"notifyStatus": "error", "error": fmt.Sprintf("%+v", err)}).Error("service.WithdrawDatasetProposal()")
	}

	dtoResult := dtos.BuildDatasetProposalDTO(updated)
	return &dtoResult, nil
}

func (s *publishingService) AcceptDatasetProposal(repositoryId int, nodeId string) (*dtos.DatasetProposalDTO, error) {
	log.WithFields(log.Fields{"repositoryId": repositoryId, "nodeId": nodeId}).Info("service.AcceptDatasetProposal()")

	// get Dataset Proposal by Repository Id and Node Id
	proposal, err := s.store.GetDatasetProposalForRepository(repositoryId, "SUBMITTED", nodeId)
	if err != nil {
		return nil, err
	}
	log.WithFields(log.Fields{"proposal": fmt.Sprintf("%+v", proposal)}).Debug("service.AcceptDatasetProposal()")

	// verify that the Dataset Proposal Status is “SUBMITTED”
	if proposal.ProposalStatus != "SUBMITTED" {
		return nil, fmt.Errorf("invalid action: proposal.status must be SUBMITTED in order to accept")
	}

	// get the Repository using the Organization Node Id on the Dataset Proposal
	repository, err := s.store.GetRepository(proposal.OrganizationNodeId)

	// create dataset
	result, err := s.pennsieve.CreateDatasetForAcceptedProposal(context.TODO(), proposal)
	if err != nil {
		log.WithFields(log.Fields{"failure": "CreateDatasetForAcceptedProposal", "err": fmt.Sprintf("%+v", err)}).Error("service.AcceptDatasetProposal()")
		return nil, fmt.Errorf(fmt.Sprintf("failed to CreateDatasetForAcceptedProposal (error: %+v)", err))
	}
	log.WithFields(log.Fields{"result": fmt.Sprintf("%+v", result)}).Debug("service.AcceptDatasetProposal()")

	// update Dataset Proposal
	// - set Status = “ACCEPTED”
	// - set AcceptedAt = current time
	currentTime := time.Now().Unix()
	accepted := proposal
	accepted.ProposalStatus = "ACCEPTED"
	accepted.DatasetNodeId = result.Dataset.NodeId.String
	accepted.OrganizationNodeId = result.Organization.NodeId
	accepted.UpdatedAt = currentTime
	accepted.AcceptedAt = currentTime

	updated, err := s.store.UpdateDatasetProposal(accepted)
	if err != nil {
		return nil, err
	}

	// send email to Dataset Proposal author/originator
	log.WithFields(log.Fields{"notify": "owner"}).Info("service.AcceptDatasetProposal()")
	err = s.notifyProposalOwner(accepted, notification.Accepted, repository)
	if err != nil {
		log.WithFields(log.Fields{"notifyStatus": "error", "error": fmt.Sprintf("%+v", err)}).Error("service.AcceptDatasetProposal()")
	}

	dtoResult := dtos.BuildDatasetProposalDTO(updated)
	return &dtoResult, nil
}

func (s *publishingService) RejectDatasetProposal(repositoryId int, nodeId string) (*dtos.DatasetProposalDTO, error) {
	log.WithFields(log.Fields{"repositoryId": repositoryId, "nodeId": nodeId}).Info("service.RejectDatasetProposal()")

	// get Dataset Proposal by Repository Id and Node Id
	proposal, err := s.store.GetDatasetProposalForRepository(repositoryId, "SUBMITTED", nodeId)
	if err != nil {
		return nil, err
	}
	log.WithFields(log.Fields{"proposal": fmt.Sprintf("%+v", proposal)}).Debug("service.RejectDatasetProposal()")

	// verify that the Dataset Proposal Status is “SUBMITTED”
	if proposal.ProposalStatus != "SUBMITTED" {
		return nil, fmt.Errorf("invalid action: proposal.status must be SUBMITTED in order to reject")
	}

	// get the Repository using the Organization Node Id on the Dataset Proposal
	repository, err := s.store.GetRepository(proposal.OrganizationNodeId)

	// update Dataset Proposal
	// - set Status = “REJECTED”
	// - set AcceptedAt = current time
	currentTime := time.Now().Unix()
	rejected := proposal
	rejected.ProposalStatus = "REJECTED"
	rejected.UpdatedAt = currentTime
	rejected.RejectedAt = currentTime

	updated, err := s.store.UpdateDatasetProposal(rejected)
	if err != nil {
		return nil, err
	}

	// send email to Dataset Proposal author/originator
	log.WithFields(log.Fields{"notify": "owner"}).Info("service.RejectDatasetProposal()")
	err = s.notifyProposalOwner(rejected, notification.Rejected, repository)
	if err != nil {
		log.WithFields(log.Fields{"notifyStatus": "error", "error": fmt.Sprintf("%+v", err)}).Error("service.RejectDatasetProposal()")
	}

	dtoResult := dtos.BuildDatasetProposalDTO(updated)
	return &dtoResult, nil
}

package service

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/google/uuid"
	pgdbModels "github.com/pennsieve/pennsieve-go-core/pkg/models/pgdb"
	"github.com/pennsieve/publishing-service/api/aws/ses"
	sesTypes "github.com/pennsieve/publishing-service/api/aws/ses/types"
	"github.com/pennsieve/publishing-service/api/dtos"
	"github.com/pennsieve/publishing-service/api/logging"
	"github.com/pennsieve/publishing-service/api/models"
	"github.com/pennsieve/publishing-service/api/notification"
	"github.com/pennsieve/publishing-service/api/store"
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
	AcceptDatasetProposal(orgNodeId string, nodeId string) (*dtos.DatasetProposalDTO, error)
	RejectDatasetProposal(orgNodeId string, nodeId string) (*dtos.DatasetProposalDTO, error)
}

// NewPublishingService builds the service for one request. logger is the
// request-scoped logger built at the entrypoint (carrying the trace id and the
// org/user context); it is held on the struct so no method has to reach for
// slog.Default.
func NewPublishingService(logger *slog.Logger, pubStore store.PublishingStore, pennsieve store.PennsievePublishingStore, notifier notification.Notifier) *publishingService {
	return &publishingService{
		logger:    logger,
		store:     pubStore,
		pennsieve: pennsieve,
		notifier:  notifier,
	}
}

type publishingService struct {
	logger    *slog.Logger
	store     store.PublishingStore
	pennsieve store.PennsievePublishingStore
	notifier  notification.Notifier
}

func usersName(user *pgdbModels.User) string {
	return fmt.Sprintf("%s %s", user.FirstName, user.LastName)
}

func sendEmail(ctx context.Context, logger *slog.Logger, sender string, recipients []string, subject string, body string) error {
	// send email message
	emailAgent := ses.MakeEmailer(logger)
	err := emailAgent.SendMessage(ctx, sender, recipients, subject, body, sesTypes.Text)
	if err != nil {
		logger.Error("failed to send email",
			slog.Int(logging.KeyCount, len(recipients)),
			slog.Any(logging.KeyError, err))
	}
	return err
}

func (s *publishingService) notifyPublishingTeam(proposal *models.DatasetProposal, action notification.Notification, repository *models.Repository) error {
	logger := s.logger.With(
		slog.String(logging.KeyNodeID, proposal.NodeId),
		slog.String(logging.KeyOrgNodeID, repository.OrganizationNodeId),
		slog.String(logging.KeyAction, action.String()))
	logger.Info("notifying publishing team")

	ctx := context.TODO()

	// get Publishing team for the Repository
	publishers, err := s.pennsieve.GetPublishingTeamMembers(ctx, repository)
	if err != nil {
		logger.Error("pennsieve.GetPublishingTeamMembers() failed", slog.Any(logging.KeyError, err))
		return err
	}
	logger.Debug("resolved publishing team", slog.Int(logging.KeyCount, len(publishers)))

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
	logger := s.logger.With(
		slog.String(logging.KeyNodeID, proposal.NodeId),
		slog.String(logging.KeyOrgNodeID, repository.OrganizationNodeId),
		slog.String(logging.KeyAction, action.String()))
	logger.Info("notifying proposal owner")

	ctx := context.TODO()

	// lookup the Welcome Workspace
	welcomeWorkspace, err := s.pennsieve.GetWelcomeWorkspace(ctx)
	if err != nil {
		logger.Error("pennsieve.GetWelcomeWorkspace() failed", slog.Any(logging.KeyError, err))
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
	var err error

	info, err := s.store.GetInfo()
	if err != nil {
		s.logger.Error("store.GetInfo() failed", slog.Any(logging.KeyError, err))
		return nil, err
	}

	var infoDTOs []dtos.InfoDTO
	for i := 0; i < len(info); i++ {
		infoDTOs = append(infoDTOs, dtos.BuildInfoDTO(s.logger, info[i]))
	}

	s.logger.Debug("retrieved publishing info", slog.Int(logging.KeyCount, len(infoDTOs)))
	return infoDTOs, nil
}

func (s *publishingService) GetPublishingRepositories() ([]dtos.RepositoryDTO, error) {
	var err error

	repositories, err := s.store.GetRepositories()
	if err != nil {
		s.logger.Error("store.GetRepositories() failed", slog.Any(logging.KeyError, err))
		return nil, err
	}

	questions, err := s.store.GetQuestions()
	if err != nil {
		s.logger.Error("store.GetQuestions() failed", slog.Any(logging.KeyError, err))
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
		repositoryDTOs = append(repositoryDTOs, dtos.BuildRepositoryDTO(s.logger, repositories[i], questionMap))
	}
	s.logger.Debug("retrieved publishing repositories", slog.Int(logging.KeyCount, len(repositoryDTOs)))
	return repositoryDTOs, nil
}

func (s *publishingService) GetProposalQuestions() ([]dtos.QuestionDTO, error) {
	var err error

	questions, err := s.store.GetQuestions()
	if err != nil {
		s.logger.Error("store.GetQuestions() failed", slog.Any(logging.KeyError, err))
		return nil, err
	}

	var questionDTOs []dtos.QuestionDTO
	for i := 0; i < len(questions); i++ {
		questionDTOs = append(questionDTOs, dtos.QuestionDTO{
			Id:       questions[i].Id,
			Question: questions[i].Question,
		})
	}

	s.logger.Debug("retrieved proposal questions", slog.Int(logging.KeyCount, len(questionDTOs)))
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
	proposal, err := s.store.GetDatasetProposal(userId, nodeId)
	if err != nil {
		// TODO: fix this, we should not return anything for the value
		s.logger.Error("store.GetDatasetProposal() failed",
			slog.Int(logging.KeyUserID, userId),
			slog.String(logging.KeyNodeID, nodeId),
			slog.Any(logging.KeyError, err))
		return dtos.DatasetProposalDTO{}, err
	}

	proposalDTO := dtos.BuildDatasetProposalDTO(proposal)

	return proposalDTO, nil
}

func (s *publishingService) GetDatasetProposalsForUser(userId int64) ([]dtos.DatasetProposalDTO, error) {
	proposals, err := s.store.GetDatasetProposalsForUser(userId)
	if err != nil {
		s.logger.Error("store.GetDatasetProposalsForUser() failed",
			slog.Int64(logging.KeyUserID, userId),
			slog.Any(logging.KeyError, err))
		return nil, err
	}

	s.logger.Debug("retrieved proposals for user",
		slog.Int64(logging.KeyUserID, userId),
		slog.Int(logging.KeyCount, len(proposals)))
	return proposalDTOsList(proposals), nil
}

func (s *publishingService) GetDatasetProposalsForWorkspace(orgNodeId string, status string) ([]dtos.DatasetProposalDTO, error) {
	// TODO: verify that status is one of: SUBMITTED, ACCEPTED, REJECTED

	proposals, err := s.store.GetDatasetProposalsForWorkspace(orgNodeId, status)
	if err != nil {
		s.logger.Error("store.GetDatasetProposalsForWorkspace() failed",
			slog.String(logging.KeyOrgNodeID, orgNodeId),
			slog.String(logging.KeyProposalStatus, status),
			slog.Any(logging.KeyError, err))
		return nil, err
	}

	s.logger.Debug("retrieved proposals for workspace",
		slog.String(logging.KeyOrgNodeID, orgNodeId),
		slog.String(logging.KeyProposalStatus, status),
		slog.Int(logging.KeyCount, len(proposals)))
	return proposalDTOsList(proposals), nil
}

// TODO: validate RepositoryId, ensure it is in Repositories table
// TODO: move generating ProposalNodeId string elsewhere (pennsieve-core?)
// TODO: refactor Create..() and Update..() to use common code
func (s *publishingService) CreateDatasetProposal(userId int64, dto dtos.DatasetProposalDTO) (*dtos.DatasetProposalDTO, error) {
	user, err := s.pennsieve.GetProposalUser(context.TODO(), userId)
	if err != nil {
		s.logger.Error("pennsieve.GetProposalUser() failed",
			slog.Int64(logging.KeyUserID, userId),
			slog.Any(logging.KeyError, err))
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
	_, err = s.store.CreateDatasetProposal(proposal)
	if err != nil {
		s.logger.Error("store.CreateDatasetProposal() failed",
			slog.String(logging.KeyNodeID, proposal.NodeId),
			slog.Any(logging.KeyError, err))
		return nil, err
	}
	s.logger.Info("created dataset proposal",
		slog.String(logging.KeyNodeID, proposal.NodeId),
		slog.String(logging.KeyOrgNodeID, proposal.OrganizationNodeId))

	dtoResult := dtos.BuildDatasetProposalDTO(proposal)
	return &dtoResult, nil
}

func (s *publishingService) UpdateDatasetProposal(userId int64, existing dtos.DatasetProposalDTO, update dtos.DatasetProposalDTO) (*dtos.DatasetProposalDTO, error) {
	user, err := s.pennsieve.GetProposalUser(context.TODO(), userId)
	if err != nil {
		s.logger.Error("pennsieve.GetProposalUser() failed",
			slog.Int64(logging.KeyUserID, userId),
			slog.Any(logging.KeyError, err))
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
	_, err = s.store.UpdateDatasetProposal(updated)
	if err != nil {
		s.logger.Error("store.UpdateDatasetProposal() failed",
			slog.String(logging.KeyNodeID, updated.NodeId),
			slog.Any(logging.KeyError, err))
		return nil, err
	}
	s.logger.Info("updated dataset proposal", slog.String(logging.KeyNodeID, updated.NodeId))

	dtoResult := dtos.BuildDatasetProposalDTO(updated)
	return &dtoResult, nil
}

func (s *publishingService) DeleteDatasetProposal(proposalDTO dtos.DatasetProposalDTO) (bool, error) {
	proposal := dtos.BuildDatasetProposal(proposalDTO)

	err := s.store.DeleteDatasetProposal(proposal)
	if err != nil {
		s.logger.Error("store.DeleteDatasetProposal() failed",
			slog.String(logging.KeyNodeID, proposal.NodeId),
			slog.Any(logging.KeyError, err))
		return false, err
	}

	s.logger.Info("deleted dataset proposal", slog.String(logging.KeyNodeID, proposal.NodeId))
	return true, nil
}

func (s *publishingService) SubmitDatasetProposal(userId int, nodeId string) (*dtos.DatasetProposalDTO, error) {
	logger := s.logger.With(slog.String(logging.KeyNodeID, nodeId))

	// get Dataset Proposal by User Id and Node Id
	proposal, err := s.store.GetDatasetProposal(userId, nodeId)
	if err != nil {
		logger.Error("store.GetDatasetProposal() failed",
			slog.Int(logging.KeyUserID, userId),
			slog.Any(logging.KeyError, err))
		return nil, err
	}

	// verify that the Dataset Proposal Status is “DRAFT”
	if proposal.ProposalStatus != "DRAFT" {
		logger.Warn("proposal is not in DRAFT status",
			slog.String(logging.KeyProposalStatus, proposal.ProposalStatus))
		return nil, fmt.Errorf("invalid action: proposal.status must be DRAFT in order to submit")
	}

	// get the Repository using the Organization Node Id on the Dataset Proposal
	repository, err := s.store.GetRepository(proposal.OrganizationNodeId)
	if err != nil {
		logger.Error("store.GetRepository() failed",
			slog.String(logging.KeyOrgNodeID, proposal.OrganizationNodeId),
			slog.Any(logging.KeyError, err))
		return nil, err
	}

	// verify that Organization NodeId is the same on the Repository and the Dataset Proposal (extra check)
	if proposal.OrganizationNodeId != repository.OrganizationNodeId {
		logger.Error("proposal organization node id does not match the repository",
			slog.String(logging.KeyOrgNodeID, proposal.OrganizationNodeId),
			slog.String(logging.KeyRepository, repository.OrganizationNodeId))
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
		logger.Warn("proposal does not answer all repository questions")
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
		logger.Error("store.UpdateDatasetProposal() failed submitting proposal", slog.Any(logging.KeyError, err))
		return nil, err
	}
	logger.Info("submitted dataset proposal")

	// send email to Repository Publishers Team
	err = s.notifyPublishingTeam(submitted, notification.Submitted, repository)
	if err != nil {
		// A notification failure does not fail the submission itself.
		logger.Error("failed to notify publishing team of submission", slog.Any(logging.KeyError, err))
	}

	dtoResult := dtos.BuildDatasetProposalDTO(updated)
	return &dtoResult, nil
}

func (s *publishingService) WithdrawDatasetProposal(userId int, nodeId string) (*dtos.DatasetProposalDTO, error) {
	logger := s.logger.With(slog.String(logging.KeyNodeID, nodeId))

	// get Dataset Proposal by User Id and Node Id
	proposal, err := s.store.GetDatasetProposal(userId, nodeId)
	if err != nil {
		logger.Error("store.GetDatasetProposal() failed",
			slog.Int(logging.KeyUserID, userId),
			slog.Any(logging.KeyError, err))
		return nil, err
	}

	// verify that the Dataset Proposal Status is “SUBMITTED”
	if proposal.ProposalStatus != "SUBMITTED" {
		logger.Warn("proposal is not in SUBMITTED status",
			slog.String(logging.KeyProposalStatus, proposal.ProposalStatus))
		return nil, fmt.Errorf("invalid action: proposal.status must be SUBMITTED in order to withdraw")
	}

	// get the Repository using the Organization Node Id on the Dataset Proposal
	repository, err := s.store.GetRepository(proposal.OrganizationNodeId)
	if err != nil {
		logger.Error("store.GetRepository() failed",
			slog.String(logging.KeyOrgNodeID, proposal.OrganizationNodeId),
			slog.Any(logging.KeyError, err))
		return nil, err
	}

	// update Dataset Proposal
	currentTime := time.Now().Unix()
	withdrawn := proposal
	withdrawn.ProposalStatus = "WITHDRAWN"
	withdrawn.UpdatedAt = currentTime
	withdrawn.WithdrawnAt = currentTime

	updated, err := s.store.UpdateDatasetProposal(withdrawn)
	if err != nil {
		logger.Error("store.UpdateDatasetProposal() failed withdrawing proposal", slog.Any(logging.KeyError, err))
		return nil, err
	}
	logger.Info("withdrew dataset proposal")

	// send email to Repository Publishers Team
	err = s.notifyPublishingTeam(withdrawn, notification.Withdrawn, repository)
	if err != nil {
		// A notification failure does not fail the withdrawal itself.
		logger.Error("failed to notify publishing team of withdrawal", slog.Any(logging.KeyError, err))
	}

	dtoResult := dtos.BuildDatasetProposalDTO(updated)
	return &dtoResult, nil
}

func (s *publishingService) AcceptDatasetProposal(orgNodeId string, nodeId string) (*dtos.DatasetProposalDTO, error) {
	logger := s.logger.With(
		slog.String(logging.KeyOrgNodeID, orgNodeId),
		slog.String(logging.KeyNodeID, nodeId))

	// get Dataset Proposal by Repository Id and Node Id
	proposal, err := s.store.GetDatasetProposalForRepository(orgNodeId, "SUBMITTED", nodeId)
	if err != nil {
		logger.Error("store.GetDatasetProposalForRepository() failed", slog.Any(logging.KeyError, err))
		return nil, err
	}

	// verify that the Dataset Proposal Status is “SUBMITTED”
	if proposal.ProposalStatus != "SUBMITTED" {
		logger.Warn("proposal is not in SUBMITTED status",
			slog.String(logging.KeyProposalStatus, proposal.ProposalStatus))
		return nil, fmt.Errorf("invalid action: proposal.status must be SUBMITTED in order to accept")
	}

	// get the Repository using the Organization Node Id on the Dataset Proposal
	repository, err := s.store.GetRepository(proposal.OrganizationNodeId)
	if err != nil {
		logger.Error("store.GetRepository() failed",
			slog.String(logging.KeyRepository, proposal.OrganizationNodeId),
			slog.Any(logging.KeyError, err))
		return nil, err
	}

	// create dataset
	result, err := s.pennsieve.CreateDatasetForAcceptedProposal(context.TODO(), proposal)
	if err != nil {
		logger.Error("pennsieve.CreateDatasetForAcceptedProposal() failed", slog.Any(logging.KeyError, err))
		return nil, fmt.Errorf("failed to CreateDatasetForAcceptedProposal: %w", err)
	}
	logger.Info("created dataset for accepted proposal",
		slog.String(logging.KeyDatasetID, result.Dataset.NodeId.String))

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
		logger.Error("store.UpdateDatasetProposal() failed accepting proposal", slog.Any(logging.KeyError, err))
		return nil, err
	}
	logger.Info("accepted dataset proposal")

	// send email to Dataset Proposal author/originator
	err = s.notifyProposalOwner(accepted, notification.Accepted, repository)
	if err != nil {
		// A notification failure does not fail the acceptance itself.
		logger.Error("failed to notify proposal owner of acceptance", slog.Any(logging.KeyError, err))
	}

	dtoResult := dtos.BuildDatasetProposalDTO(updated)
	return &dtoResult, nil
}

func (s *publishingService) RejectDatasetProposal(orgNodeId string, nodeId string) (*dtos.DatasetProposalDTO, error) {
	logger := s.logger.With(
		slog.String(logging.KeyOrgNodeID, orgNodeId),
		slog.String(logging.KeyNodeID, nodeId))

	// get Dataset Proposal by Repository Id and Node Id
	proposal, err := s.store.GetDatasetProposalForRepository(orgNodeId, "SUBMITTED", nodeId)
	if err != nil {
		logger.Error("store.GetDatasetProposalForRepository() failed", slog.Any(logging.KeyError, err))
		return nil, err
	}

	// verify that the Dataset Proposal Status is “SUBMITTED”
	if proposal.ProposalStatus != "SUBMITTED" {
		logger.Warn("proposal is not in SUBMITTED status",
			slog.String(logging.KeyProposalStatus, proposal.ProposalStatus))
		return nil, fmt.Errorf("invalid action: proposal.status must be SUBMITTED in order to reject")
	}

	// get the Repository using the Organization Node Id on the Dataset Proposal
	repository, err := s.store.GetRepository(proposal.OrganizationNodeId)
	if err != nil {
		logger.Error("store.GetRepository() failed",
			slog.String(logging.KeyRepository, proposal.OrganizationNodeId),
			slog.Any(logging.KeyError, err))
		return nil, err
	}

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
		logger.Error("store.UpdateDatasetProposal() failed rejecting proposal", slog.Any(logging.KeyError, err))
		return nil, err
	}
	logger.Info("rejected dataset proposal")

	// send email to Dataset Proposal author/originator
	err = s.notifyProposalOwner(rejected, notification.Rejected, repository)
	if err != nil {
		// A notification failure does not fail the rejection itself.
		logger.Error("failed to notify proposal owner of rejection", slog.Any(logging.KeyError, err))
	}

	dtoResult := dtos.BuildDatasetProposalDTO(updated)
	return &dtoResult, nil
}

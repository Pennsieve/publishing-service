package dtos

import (
	"log/slog"
	"time"

	"github.com/pennsieve/publishing-service/api/aws/s3"
	"github.com/pennsieve/publishing-service/api/logging"
	"github.com/pennsieve/publishing-service/api/models"
)

// presignedURL returns a 12-hour presigned GET URL for an S3 file, or the empty
// string if presigning fails. The presigned request was previously dereferenced
// without checking the error, which panics on any presign failure.
func presignedURL(logger *slog.Logger, presigner *s3.Presigner, file models.S3Location) string {
	request, err := presigner.GetObject(file.S3Bucket, file.S3Key, 12*3600)
	if err != nil || request == nil {
		logger.Error("failed to presign a file URL",
			slog.String(logging.KeyS3Bucket, file.S3Bucket),
			slog.String(logging.KeyS3Key, file.S3Key),
			slog.Any(logging.KeyError, err))
		return ""
	}
	return request.URL
}

func BuildQuestionDTO(question models.Question) QuestionDTO {
	return QuestionDTO{
		Id:       question.Id,
		Question: question.Question,
		Type:     question.Type,
	}
}

func BuildSurveyDTO(survey models.Survey) SurveyDTO {
	return SurveyDTO{
		QuestionId: survey.QuestionId,
		Response:   survey.Response,
	}
}

func BuildSurvey(survey SurveyDTO) models.Survey {
	return models.Survey{
		QuestionId: survey.QuestionId,
		Response:   survey.Response,
	}
}

func BuildContributorDTO(contributor models.Contributor) ContributorDTO {
	return ContributorDTO{
		FirstName:    contributor.FirstName,
		LastName:     contributor.LastName,
		EmailAddress: contributor.EmailAddress,
	}
}

func BuildContributor(contributor ContributorDTO) models.Contributor {
	return models.Contributor{
		FirstName:    contributor.FirstName,
		LastName:     contributor.LastName,
		EmailAddress: contributor.EmailAddress,
	}
}

func BuildInfoDTO(logger *slog.Logger, info models.Info) InfoDTO {
	presigner := s3.MakePresigner(logger)

	return InfoDTO{
		Tag:  info.Tag,
		Type: info.Type,
		URL:  presignedURL(logger, presigner, info.File),
	}
}

// TODO: can we better abstract the type for questionMap?
func BuildRepositoryDTO(logger *slog.Logger, repository models.Repository, questionMap map[int]QuestionDTO) RepositoryDTO {
	// build list of selected Questions for the Repository
	var questionDTOs []QuestionDTO
	for i := 0; i < len(repository.Questions); i++ {
		questionNumber := repository.Questions[i]
		questionDTOs = append(questionDTOs, questionMap[questionNumber])
	}

	presigner := s3.MakePresigner(logger)

	return RepositoryDTO{
		OrganizationNodeId:  repository.OrganizationNodeId,
		Name:                repository.Name,
		DisplayName:         repository.DisplayName,
		Type:                repository.Type,
		Description:         repository.Description,
		URL:                 repository.URL,
		OverviewDocumentUrl: presignedURL(logger, presigner, repository.OverviewDocument),
		LogoFileUrl:         presignedURL(logger, presigner, repository.LogoFile),
		Questions:           questionDTOs,
		CreatedAt:           repository.CreatedAt,
		UpdatedAt:           repository.UpdatedAt,
	}
}

func BuildDatasetProposalDTO(proposal *models.DatasetProposal) DatasetProposalDTO {
	var surveyDTOs []SurveyDTO
	for i := 0; i < len(proposal.Survey); i++ {
		surveyDTOs = append(surveyDTOs, BuildSurveyDTO(proposal.Survey[i]))
	}

	var contributorDTOs []ContributorDTO
	for i := 0; i < len(proposal.Contributors); i++ {
		contributorDTOs = append(contributorDTOs, BuildContributorDTO(proposal.Contributors[i]))
	}

	return DatasetProposalDTO{
		UserId:             proposal.UserId,
		NodeId:             proposal.NodeId,
		OwnerName:          proposal.OwnerName,
		EmailAddress:       proposal.EmailAddress,
		Name:               proposal.Name,
		Description:        proposal.Description,
		OrganizationNodeId: proposal.OrganizationNodeId,
		ProposalStatus:     proposal.ProposalStatus,
		DatasetNodeId:      proposal.DatasetNodeId,
		Survey:             surveyDTOs,
		Contributors:       contributorDTOs,
		CreatedAt:          proposal.CreatedAt,
		UpdatedAt:          proposal.UpdatedAt,
		SubmittedAt:        proposal.SubmittedAt,
		WithdrawnAt:        proposal.WithdrawnAt,
		AcceptedAt:         proposal.AcceptedAt,
		RejectedAt:         proposal.RejectedAt,
	}
}

func BuildDatasetProposal(dto DatasetProposalDTO) *models.DatasetProposal {
	var survey []models.Survey
	for i := 0; i < len(dto.Survey); i++ {
		survey = append(survey, BuildSurvey(dto.Survey[i]))
	}

	var contributors []models.Contributor
	for i := 0; i < len(dto.Contributors); i++ {
		contributors = append(contributors, BuildContributor(dto.Contributors[i]))
	}

	currentTime := time.Now().Unix()

	proposal := &models.DatasetProposal{
		UserId:             dto.UserId,
		NodeId:             dto.NodeId,
		OwnerName:          dto.OwnerName,
		EmailAddress:       dto.EmailAddress,
		Name:               dto.Name,
		Description:        dto.Description,
		OrganizationNodeId: dto.OrganizationNodeId,
		ProposalStatus:     dto.ProposalStatus,
		Survey:             survey,
		Contributors:       contributors,
		CreatedAt:          currentTime,
		UpdatedAt:          currentTime,
	}

	return proposal
}

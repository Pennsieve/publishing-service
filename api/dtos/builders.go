package dtos

import (
	"github.com/pennsieve/publishing-service/api/aws/s3"
	"github.com/pennsieve/publishing-service/api/models"
)

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

// TODO: can we better abstract the type for questionMap?
func BuildRepositoryDTO(repository models.Repository, questionMap map[int]QuestionDTO) RepositoryDTO {
	// build list of selected Questions for the Repository
	var questionDTOs []QuestionDTO
	for i := 0; i < len(repository.Questions); i++ {
		questionNumber := repository.Questions[i]
		questionDTOs = append(questionDTOs, questionMap[questionNumber])
	}

	presigner := s3.MakePresigner()

	overviewDocument, _ := presigner.GetObject(
		repository.OverviewDocument.S3Bucket,
		repository.OverviewDocument.S3Key,
		3600,
	)

	logoFile, _ := presigner.GetObject(
		repository.LogoFile.S3Bucket,
		repository.LogoFile.S3Key,
		3600,
	)

	return RepositoryDTO{
		OrganizationNodeId:  repository.OrganizationNodeId,
		Name:                repository.Name,
		DisplayName:         repository.DisplayName,
		WorkspaceId:         repository.WorkspaceId,
		Type:                repository.Type,
		Description:         repository.Description,
		URL:                 repository.URL,
		OverviewDocumentUrl: overviewDocument.URL,
		LogoFileUrl:         logoFile.URL,
		Questions:           questionDTOs,
		CreatedAt:           repository.CreatedAt,
		UpdatedAt:           repository.UpdatedAt,
	}
}

func BuildDatasetProposalDTO(proposal models.DatasetProposal) DatasetProposalDTO {
	var surveyDTOs []SurveyDTO
	for i := 0; i < len(proposal.Survey); i++ {
		surveyDTOs = append(surveyDTOs, BuildSurveyDTO(proposal.Survey[i]))
	}
	return DatasetProposalDTO{
		UserId:         proposal.UserId,
		ProposalNodeId: proposal.ProposalNodeId,
		Name:           proposal.Name,
		Description:    proposal.Description,
		RepositoryId:   proposal.RepositoryId,
		Status:         proposal.Status,
		Survey:         surveyDTOs,
	}
}

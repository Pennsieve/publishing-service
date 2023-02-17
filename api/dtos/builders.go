package dtos

import (
	"github.com/pennsieve/publishing-service/api/aws/s3"
	"github.com/pennsieve/publishing-service/api/models"
)

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

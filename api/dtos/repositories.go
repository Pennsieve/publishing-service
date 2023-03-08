package dtos

// Notes:
//   - OrganizationNodeId is the Pennsieve Organization NodeId
//   - RepositoryId is the Pennsieve Organization Id

type RepositoryDTO struct {
	OrganizationNodeId  string        `json:"organizationNodeId"`
	Name                string        `json:"name"`
	DisplayName         string        `json:"displayName"`
	RepositoryId        int64         `json:"repositoryId"`
	Type                string        `json:"type"`
	Description         string        `json:"description"`
	URL                 string        `json:"url"`
	OverviewDocumentUrl string        `json:"overviewDocument"`
	LogoFileUrl         string        `json:"logoFile"`
	Questions           []QuestionDTO `json:"questions"`
	CreatedAt           int64         `json:"createdAt"`
	UpdatedAt           int64         `json:"updatedAt"`
}

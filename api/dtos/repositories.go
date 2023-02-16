package dtos

type RepositoryDTO struct {
	OrganizationNodeId  string        `json:"OrganizationNodeId"`
	Name                string        `json:"Name"`
	DisplayName         string        `json:"DisplayName"`
	WorkspaceId         int64         `json:"WorkspaceId"`
	Type                string        `json:"Type"`
	Description         string        `json:"Description"`
	URL                 string        `json:"URL"`
	OverviewDocumentUrl string        `json:"OverviewDocument"`
	LogoFileUrl         string        `json:"LogoFile"`
	Questions           []QuestionDTO `json:"Questions"`
	CreatedAt           int64         `json:"CreatedAt"`
	UpdatedAt           int64         `json:"UpdatedAt"`
}

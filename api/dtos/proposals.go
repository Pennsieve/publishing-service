package dtos

type SurveyDTO struct {
	QuestionId int    `json:"questionId"`
	Response   string `json:"response"`
}

type DatasetProposalDTO struct {
	UserId             int              `json:"userId"`
	NodeId             string           `json:"nodeId"`
	OwnerName          string           `json:"ownerName"`
	Name               string           `json:"name"`
	Description        string           `json:"description"`
	RepositoryId       int              `json:"repositoryId"`
	OrganizationNodeId string           `json:"organizationNodeId"`
	DatasetNodeId      string           `json:"datasetNodeId"`
	Status             string           `json:"status"`
	Survey             []SurveyDTO      `json:"survey"`
	Contributors       []ContributorDTO `json:"contributors"`
	CreatedAt          int64            `json:"createdAt"`
	UpdatedAt          int64            `json:"updatedAt"`
}

type DatasetSubmissionsDTO struct {
	TotalCount int                  `json:"totalCount"`
	Proposals  []DatasetProposalDTO `json:"proposals"`
}

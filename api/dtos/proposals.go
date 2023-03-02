package dtos

type SurveyDTO struct {
	QuestionId int    `json:"QuestionId"`
	Response   string `json:"Response"`
}

type DatasetProposalDTO struct {
	UserId             int              `json:"UserId"`
	ProposalNodeId     string           `json:"ProposalNodeId"`
	Name               string           `json:"Name"`
	Description        string           `json:"Description"`
	RepositoryId       int              `json:"RepositoryId"`
	OrganizationNodeId string           `json:"OrganizationNodeId"`
	DatasetNodeId      string           `json:"DatasetNodeId"`
	Status             string           `json:"Status"`
	Survey             []SurveyDTO      `json:"Survey"`
	Contributors       []ContributorDTO `json:"Contributors"`
	CreatedAt          int64            `json:"CreatedAt"`
	UpdatedAt          int64            `json:"UpdatedAt"`
}

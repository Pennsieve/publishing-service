package dtos

type SurveyDTO struct {
	QuestionId int    `json:"questionId"`
	Response   string `json:"response"`
}

type DatasetProposalDTO struct {
	UserId             int              `json:"userId"`
	NodeId             string           `json:"nodeId"`
	OwnerName          string           `json:"ownerName"`
	EmailAddress       string           `json:"emailAddress"`
	Name               string           `json:"name"`
	Description        string           `json:"description"`
	OrganizationNodeId string           `json:"organizationNodeId"`
	DatasetNodeId      string           `json:"datasetNodeId"`
	ProposalStatus     string           `json:"proposalStatus"`
	Survey             []SurveyDTO      `json:"survey"`
	Contributors       []ContributorDTO `json:"contributors"`
	CreatedAt          int64            `json:"createdAt"`
	UpdatedAt          int64            `json:"updatedAt"`
	SubmittedAt        int64            `json:"submittedAt"`
	WithdrawnAt        int64            `json:"withdrawnAt"`
	AcceptedAt         int64            `json:"acceptedAt"`
	RejectedAt         int64            `json:"rejectedAt"`
}

type DatasetSubmissionsDTO struct {
	TotalCount int                  `json:"totalCount"`
	Proposals  []DatasetProposalDTO `json:"proposals"`
}

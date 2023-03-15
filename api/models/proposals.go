package models

type Survey struct {
	QuestionId int    `dynamodbav:"QuestionId"`
	Response   string `dynamodbav:"Response"`
}

type DatasetProposal struct {
	UserId             int           `dynamodbav:"UserId"`
	NodeId             string        `dynamodbav:"NodeId"`
	OwnerName          string        `dynamodbav:"OwnerName"`
	Name               string        `dynamodbav:"Name"`
	Description        string        `dynamodbav:"Description"`
	RepositoryId       int           `dynamodbav:"RepositoryId"`
	OrganizationNodeId string        `dynamodbav:"OrganizationNodeId"`
	DatasetNodeId      string        `dynamodbav:"DatasetNodeId"`
	ProposalStatus     string        `dynamodbav:"ProposalStatus"`
	Survey             []Survey      `dynamodbav:"Survey"`
	Contributors       []Contributor `dynamodbav:"Contributors"`
	CreatedAt          int64         `dynamodbav:"CreatedAt"`
	UpdatedAt          int64         `dynamodbav:"UpdatedAt"`
	SubmittedAt        int64         `dynamodbav:"SubmittedAt"`
}

type DatasetProposalKey struct {
	UserId int    `json:"UserId"`
	NodeId string `json:"NodeId"`
}

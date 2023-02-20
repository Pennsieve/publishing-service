package models

type Survey struct {
	QuestionId int    `dynamodbav:"QuestionId"`
	Response   string `dynamodbav:"Response"`
}

type DatasetProposal struct {
	UserId         int      `dynamodbav:"UserId"`
	ProposalId     int      `dynamodbav:"ProposalId"`
	ProposalNodeId string   `dynamodbav:"ProposalNodeId"`
	Name           string   `dynamodbav:"Name"`
	Description    string   `dynamodbav:"Description"`
	RepositoryId   int      `dynamodbav:"RepositoryId"`
	Status         string   `dynamodbav:"Status"`
	Survey         []Survey `dynamodbav:"Survey"`
}

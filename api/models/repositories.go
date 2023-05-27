package models

type S3Location struct {
	S3Bucket string `dynamodbav:"s3bucket"`
	S3Key    string `dynamodbav:"s3Key"`
}

type Repository struct {
	OrganizationNodeId string     `dynamodbav:"OrganizationNodeId"`
	Name               string     `dynamodbav:"Name"`
	DisplayName        string     `dynamodbav:"DisplayName"`
	Type               string     `dynamodbav:"Type"`
	Description        string     `dynamodbav:"Description"`
	URL                string     `dynamodbav:"URL"`
	OverviewDocument   S3Location `dynamodbav:"OverviewDocument"`
	LogoFile           S3Location `dynamodbav:"LogoFile"`
	Questions          []int      `dynamodbav:"Questions"`
	CreatedAt          int64      `dynamodbav:"CreatedAt"`
	UpdatedAt          int64      `dynamodbav:"UpdatedAt"`
}

package models

type Info struct {
	Tag      string     `dynamodbav:"Tag"`
	Document S3Location `dynamodbav:"URL"`
}

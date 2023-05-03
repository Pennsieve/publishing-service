package models

type Info struct {
	Tag  string     `dynamodbav:"Tag"`
	Type string     `dynamodbav:"Type"`
	File S3Location `dynamodbav:"File"`
}

package models

type Question struct {
	Id       int64  `dynamodbav:"Id"`
	Question string `dynamodbav:"Question"`
}

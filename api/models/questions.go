package models

type Question struct {
	Id       int    `dynamodbav:"Id"`
	Question string `dynamodbav:"Question"`
}

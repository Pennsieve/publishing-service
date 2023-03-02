package models

type Contributor struct {
	FirstName    string `dynamodbav:"FirstName"`
	LastName     string `dynamodbav:"LastName"`
	EmailAddress string `dynamodbav:"EmailAddress"`
}

package dtos

type QuestionDTO struct {
	Id       int    `json:"Id"`
	Question string `json:"Question"`
	Type     string `json:"Type"`
}

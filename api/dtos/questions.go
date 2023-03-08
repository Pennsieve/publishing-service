package dtos

type QuestionDTO struct {
	Id       int    `json:"id"`
	Question string `json:"question"`
	Type     string `json:"type"`
}

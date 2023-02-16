package store

import (
	"context"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	log "github.com/sirupsen/logrus"
	"os"
)

type PublishingStore interface {
	GetRepositories() (*dynamodb.ScanOutput, error)
	GetQuestions() (*dynamodb.ScanOutput, error)
	//GetQuestion(id int64) (string, error)
}

func getTableName(tableName string) string {
	table := os.Getenv(tableName)
	return table
}

func NewPublishingStore() *publishingStore {
	// TODO: handle and/or propagate errors
	cfg, err := config.LoadDefaultConfig(context.Background())
	if err != nil {
		// TODO: handle error
	}

	db := dynamodb.NewFromConfig(cfg)

	return &publishingStore{
		db:                db,
		repositoriesTable: getTableName("REPOSITORIES_TABLE"),
		questionsTable:    getTableName("REPOSITORY_QUESTIONS_TABLE"),
	}
}

type publishingStore struct {
	db                *dynamodb.Client
	repositoriesTable string
	questionsTable    string
}

func (s *publishingStore) GetRepositories() (*dynamodb.ScanOutput, error) {
	log.Println("GetRepositories()")
	scanInput := dynamodb.ScanInput{
		TableName: aws.String(s.repositoriesTable),
	}
	log.Println("GetRepositories() scanInput: ", scanInput)

	result, err := s.db.Scan(context.TODO(), &scanInput)
	if err != nil {
		log.Fatalln("GetRepositories() err: ", err)
		return nil, err
	}

	return result, nil
}

func (s *publishingStore) GetQuestions() (*dynamodb.ScanOutput, error) {
	log.Println("GetQuestions()")
	scanInput := dynamodb.ScanInput{
		TableName: aws.String(s.questionsTable),
	}
	log.Println("GetQuestions() scanInput: ", scanInput)

	result, err := s.db.Scan(context.TODO(), &scanInput)
	if err != nil {
		log.Fatalln("GetQuestions() err: ", err)
		return nil, err
	}

	return result, nil
}

//func (s *publishingStore) GetQuestion(id int64) (*dynamodb.ScanOutput, error) {
//
//}

package store

import (
	"context"
	"fmt"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/pennsieve/publishing-service/api/models"
	log "github.com/sirupsen/logrus"
	"os"
)

type PublishingStore interface {
	GetRepositories() ([]models.Repository, error)
	GetQuestions() ([]models.Question, error)
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

func scan(client *dynamodb.Client, tableName string) (*dynamodb.ScanOutput, error) {
	log.Println("scan() tableName: ", tableName)

	scanInput := dynamodb.ScanInput{
		TableName: aws.String(tableName),
	}
	log.Println("scan() scanInput: ", scanInput)

	result, err := client.Scan(context.TODO(), &scanInput)
	if err != nil {
		log.Fatalln("scan() err: ", err)
		return nil, err
	}

	return result, nil
}

// TODO: figure out struct embedding to simplify list of types allowed?
func transform[T models.Repository | models.Question](items []map[string]types.AttributeValue) ([]T, error) {
	var results []T
	for _, item := range items {
		var result T
		err := attributevalue.UnmarshalMap(item, &result)
		if err != nil {
			return nil, fmt.Errorf("UnmarshalMap: %v\n", err)
		}
		results = append(results, result)
	}
	return results, nil
}

func fetch[T models.Repository | models.Question](client *dynamodb.Client, tableName string) ([]T, error) {
	log.Println("fetch() tableName: ", tableName)
	var err error

	// get all Items from the table via Scan operation
	output, err := scan(client, tableName)
	if err != nil {
		log.Fatalln("fetch() - scan() err: ", err)
		return nil, err
	}

	// transform each Item in output from DynamoDB to type T
	results, err := transform[T](output.Items)
	if err != nil {
		log.Fatalln("fetch() - transform() err: ", err)
		return nil, err
	}

	return results, nil
}

func (s *publishingStore) GetRepositories() ([]models.Repository, error) {
	log.Println("GetRepositories()")
	return fetch[models.Repository](s.db, s.repositoriesTable)
}

func (s *publishingStore) GetQuestions() ([]models.Question, error) {
	log.Println("GetQuestions()")
	return fetch[models.Question](s.db, s.questionsTable)
}

//func (s *publishingStore) GetQuestion(id int64) (*dynamodb.ScanOutput, error) {
//
//}

package store

import (
	"context"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/pennsieve/pennsieve-go-api/pkg/core"
	"os"
)

type PublishingStore interface {
	GetRepositories() (*dynamodb.QueryOutput, error)
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
	}
}

type publishingStore struct {
	db                core.DynamoDBAPI
	repositoriesTable string
}

func (s *publishingStore) GetRepositories() (*dynamodb.QueryOutput, error) {
	queryInput := dynamodb.QueryInput{
		TableName: aws.String(s.repositoriesTable),
		Select:    "ALL_ATTRIBUTES",
	}

	result, err := s.db.Query(context.Background(), &queryInput)
	if err != nil {
		return nil, err
	}

	return result, nil
}

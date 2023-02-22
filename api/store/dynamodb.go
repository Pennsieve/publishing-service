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
	GetDatasetProposalsForUser(id int64) ([]models.DatasetProposal, error)
	GetDatasetProposalsForWorkspace(id int64) ([]models.DatasetProposal, error)
	CreateDatasetProposal(proposal *models.DatasetProposal) (*models.DatasetProposal, error)
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
		db:                    db,
		repositoriesTable:     getTableName("REPOSITORIES_TABLE"),
		questionsTable:        getTableName("REPOSITORY_QUESTIONS_TABLE"),
		datasetProposalsTable: getTableName("DATASET_PROPOSAL_TABLE"),
	}
}

type publishingStore struct {
	db                    *dynamodb.Client
	repositoriesTable     string
	questionsTable        string
	datasetProposalsTable string
}

func (s *publishingStore) GetDatasetProposals() {
	//TODO implement me
	panic("implement me")
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

func query(client *dynamodb.Client, predicate *dynamodb.QueryInput) (*dynamodb.QueryOutput, error) {
	log.Println("query() predicate: ", predicate)
	result, err := client.Query(context.TODO(), predicate)
	if err != nil {
		log.Fatalln("query() err: ", err)
		return nil, err
	}

	return result, nil
}

// TODO: figure out struct embedding to simplify list of types allowed?
func transform[T models.Repository | models.Question | models.DatasetProposal](items []map[string]types.AttributeValue) ([]T, error) {
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

func find[T models.DatasetProposal](client *dynamodb.Client, queryInput *dynamodb.QueryInput) ([]T, error) {
	log.Println("find() queryInput: ", queryInput)
	var err error

	output, err := query(client, queryInput)
	if err != nil {
		log.Fatalln("find() - query() err: ", err)
		return nil, err
	}

	// transform each Item in output from DynamoDB to type T
	results, err := transform[T](output.Items)
	if err != nil {
		log.Fatalln("find() - transform() err: ", err)
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

func (s *publishingStore) GetDatasetProposalsForUser(id int64) ([]models.DatasetProposal, error) {
	queryInput := dynamodb.QueryInput{
		TableName:              aws.String(s.datasetProposalsTable),
		KeyConditionExpression: aws.String("UserId = :id"),
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":id": &types.AttributeValueMemberN{Value: string(id)},
		},
		Select: "ALL_ATTRIBUTES",
	}
	return find[models.DatasetProposal](s.db, &queryInput)
}

func (s *publishingStore) GetDatasetProposalsForWorkspace(id int64) ([]models.DatasetProposal, error) {
	queryInput := dynamodb.QueryInput{
		TableName:              aws.String(s.datasetProposalsTable),
		IndexName:              aws.String("RepositoryIdIndex"),
		KeyConditionExpression: aws.String("RepositoryId = :id"),
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":id": &types.AttributeValueMemberN{Value: string(id)},
		},
		Select: "ALL_PROJECTED_ATTRIBUTES",
	}
	return find[models.DatasetProposal](s.db, &queryInput)
}

func (s *publishingStore) CreateDatasetProposal(proposal *models.DatasetProposal) (*models.DatasetProposal, error) {
	log.Println("store.CreateDatasetProposal()")

	var err error
	data, err := attributevalue.MarshalMap(proposal)
	if err != nil {
		log.Fatalln("store.CreateDatasetProposal() - attributevalue.MarshalMap() failed: ", err)
		return nil, err
	}
	log.Println("store.CreateDatasetProposal() data: ", data)

	output, err := s.db.PutItem(context.TODO(), &dynamodb.PutItemInput{
		TableName:    aws.String(s.datasetProposalsTable),
		Item:         data,
		ReturnValues: "ALL_OLD",
	})
	if err != nil {
		log.Fatalln("store.CreateDatasetProposal() - s.db.PutItem() failed: ", err)
		return nil, err
	}
	log.Println("store.CreateDatasetProposal() output: ", output)

	var item models.DatasetProposal
	err = attributevalue.UnmarshalMap(output.Attributes, &item)
	if err != nil {
		log.Fatalln("store.CreateDatasetProposal() - attributevalue.UnmarshalMap() failed: ", err)
		return nil, err
	}
	log.Println("store.CreateDatasetProposal() item: ", item)

	return &item, nil
}

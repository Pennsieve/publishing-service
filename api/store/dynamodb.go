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
	GetInfo() ([]models.Info, error)
	GetRepositories() ([]models.Repository, error)
	GetRepository(organizationNodeId string) (*models.Repository, error)
	GetQuestions() ([]models.Question, error)
	GetDatasetProposal(userId int, nodeId string) (*models.DatasetProposal, error)
	GetDatasetProposalsForUser(userId int64) ([]models.DatasetProposal, error)
	GetDatasetProposalsForWorkspace(workspaceId int64, status string) ([]models.DatasetProposal, error)
	GetDatasetProposalForRepository(repositoryId int, status string, nodeId string) (*models.DatasetProposal, error)
	CreateDatasetProposal(proposal *models.DatasetProposal) (*models.DatasetProposal, error)
	UpdateDatasetProposal(proposal *models.DatasetProposal) (*models.DatasetProposal, error)
	DeleteDatasetProposal(proposal *models.DatasetProposal) error
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
		infoTable:             getTableName("PUBLISHING_INFO_TABLE"),
		repositoriesTable:     getTableName("REPOSITORIES_TABLE"),
		questionsTable:        getTableName("REPOSITORY_QUESTIONS_TABLE"),
		datasetProposalsTable: getTableName("DATASET_PROPOSAL_TABLE"),
	}
}

type publishingStore struct {
	db                    *dynamodb.Client
	infoTable             string
	repositoriesTable     string
	questionsTable        string
	datasetProposalsTable string
}

func intToString(i int) string {
	return fmt.Sprintf("%d", i)
}

func int64ToString(i int64) string {
	return fmt.Sprintf("%d", i)
}

func scan(client *dynamodb.Client, tableName string) (*dynamodb.ScanOutput, error) {
	log.WithFields(log.Fields{"tableName": tableName}).Debug("scan()")

	scanInput := dynamodb.ScanInput{
		TableName: aws.String(tableName),
	}
	log.WithFields(log.Fields{"scanInput": fmt.Sprintf("%+v", scanInput)}).Debug("scan()")

	result, err := client.Scan(context.TODO(), &scanInput)
	if err != nil {
		log.Error("scan() err: ", err)
		return nil, err
	}

	return result, nil
}

func query(client *dynamodb.Client, queryInput *dynamodb.QueryInput) (*dynamodb.QueryOutput, error) {
	log.WithFields(log.Fields{"queryInput": fmt.Sprintf("%#v", queryInput)}).Debug("query()")
	result, err := client.Query(context.TODO(), queryInput)
	if err != nil {
		log.Error("query() err: ", err)
		return nil, err
	}

	return result, nil
}

type PublishingTypes interface {
	models.Info | models.Repository | models.Question | models.DatasetProposal
}

// TODO: figure out struct embedding to simplify list of types allowed?
func transform[T PublishingTypes](items []map[string]types.AttributeValue) ([]T, error) {
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

func fetch[T PublishingTypes](client *dynamodb.Client, tableName string) ([]T, error) {
	log.WithFields(log.Fields{"tableName": tableName}).Debug("fetch()")
	var err error

	// get all Items from the table via Scan operation
	output, err := scan(client, tableName)
	if err != nil {
		log.Error("fetch() - scan() err: ", err)
		return nil, err
	}

	// transform each Item in output from DynamoDB to type T
	results, err := transform[T](output.Items)
	if err != nil {
		log.Error("fetch() - transform() err: ", err)
		return nil, err
	}

	return results, nil
}

func find[T PublishingTypes](client *dynamodb.Client, queryInput *dynamodb.QueryInput) ([]T, error) {
	log.WithFields(log.Fields{"queryInput": fmt.Sprintf("%#v", queryInput)}).Debug("find()")
	var err error

	output, err := query(client, queryInput)
	if err != nil {
		log.Error("find() - query() err: ", err)
		return nil, err
	}

	// transform each Item in output from DynamoDB to type T
	results, err := transform[T](output.Items)
	if err != nil {
		log.Error("find() - transform() err: ", err)
		return nil, err
	}

	return results, nil
}

func get[T PublishingTypes](client *dynamodb.Client, queryInput *dynamodb.QueryInput) (*T, error) {
	log.WithFields(log.Fields{"queryInput": fmt.Sprintf("%#v", queryInput)}).Debug("get()")
	results, err := find[T](client, queryInput)
	if err != nil {
		log.Error("get() - find() err: ", err)
		return nil, err
	}

	if len(results) == 0 {
		return nil, fmt.Errorf("item not found")
	}

	if len(results) > 1 {
		return nil, fmt.Errorf("singleton get returned more than one item")
	}

	return &results[0], nil
}

// TODO: make this function a generic ~> item T[]
func store(client *dynamodb.Client, table string, item *models.DatasetProposal) (*dynamodb.PutItemOutput, error) {
	log.WithFields(log.Fields{"table": table, "item": fmt.Sprintf("%#v", item)}).Debug("store()")

	var err error
	data, err := attributevalue.MarshalMap(item)
	if err != nil {
		log.Fatalln("store.CreateDatasetProposal() - attributevalue.MarshalMap() failed: ", err)
		return nil, err
	}
	log.WithFields(log.Fields{"data": fmt.Sprintf("%+v", data)}).Debug("store.CreateDatasetProposal()")

	return client.PutItem(context.TODO(), &dynamodb.PutItemInput{
		TableName: aws.String(table),
		Item:      data,
	})
}

func (s *publishingStore) GetInfo() ([]models.Info, error) {
	log.Info("store.GetInfo()")
	return fetch[models.Info](s.db, s.infoTable)
}

func (s *publishingStore) GetRepositories() ([]models.Repository, error) {
	log.Info("store.GetRepositories()")
	return fetch[models.Repository](s.db, s.repositoriesTable)
}

func (s *publishingStore) GetRepository(organizationNodeId string) (*models.Repository, error) {
	log.WithFields(log.Fields{"organizationNodeId": organizationNodeId}).Info("GetRepository()")
	queryInput := dynamodb.QueryInput{
		TableName:              aws.String(s.repositoriesTable),
		KeyConditionExpression: aws.String("OrganizationNodeId = :organizationNodeId"),
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":organizationNodeId": &types.AttributeValueMemberS{
				Value: organizationNodeId,
			},
		},
	}
	return get[models.Repository](s.db, &queryInput)
}

func (s *publishingStore) GetQuestions() ([]models.Question, error) {
	log.Info("store.GetQuestions()")
	return fetch[models.Question](s.db, s.questionsTable)
}

func (s *publishingStore) GetDatasetProposal(userId int, nodeId string) (*models.DatasetProposal, error) {
	log.WithFields(log.Fields{"userId": userId, "nodeId": nodeId}).Info("store.GetDatasetProposal()")
	queryInput := dynamodb.QueryInput{
		TableName:              aws.String(s.datasetProposalsTable),
		KeyConditionExpression: aws.String("UserId = :userId AND NodeId = :nodeId"),
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":userId": &types.AttributeValueMemberN{
				Value: intToString(userId),
			},
			":nodeId": &types.AttributeValueMemberS{
				Value: nodeId,
			},
		},
	}
	return get[models.DatasetProposal](s.db, &queryInput)
}

func (s *publishingStore) GetDatasetProposalsForUser(userId int64) ([]models.DatasetProposal, error) {
	log.WithFields(log.Fields{"userId": userId}).Info("store.GetDatasetProposalsForUser()")
	queryInput := dynamodb.QueryInput{
		TableName:              aws.String(s.datasetProposalsTable),
		KeyConditionExpression: aws.String("UserId = :userId"),
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":userId": &types.AttributeValueMemberN{
				Value: int64ToString(userId),
			},
		},
	}
	return find[models.DatasetProposal](s.db, &queryInput)
}

func (s *publishingStore) GetDatasetProposalsForWorkspace(workspaceId int64, status string) ([]models.DatasetProposal, error) {
	log.WithFields(log.Fields{"workspaceId": workspaceId}).Info("store.GetDatasetProposalsForWorkspace()")
	queryInput := dynamodb.QueryInput{
		TableName:              aws.String(s.datasetProposalsTable),
		IndexName:              aws.String("RepositoryProposalStatusIndex"),
		KeyConditionExpression: aws.String("RepositoryId = :workspaceId AND ProposalStatus = :status"),
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":workspaceId": &types.AttributeValueMemberN{
				Value: int64ToString(workspaceId),
			},
			":status": &types.AttributeValueMemberS{
				Value: status,
			},
		},
		Select: "ALL_PROJECTED_ATTRIBUTES",
	}
	return find[models.DatasetProposal](s.db, &queryInput)
}

func (s *publishingStore) CreateDatasetProposal(proposal *models.DatasetProposal) (*models.DatasetProposal, error) {
	log.Info("store.CreateDatasetProposal()")

	result, err := store(s.db, s.datasetProposalsTable, proposal)
	if err != nil {
		log.Fatalln("store.CreateDatasetProposal() - store() failed: ", err)
		return nil, err
	}
	log.WithFields(log.Fields{"result": fmt.Sprintf("%+v", result)}).Debug("store.CreateDatasetProposal()")

	return proposal, nil
}

func (s *publishingStore) UpdateDatasetProposal(proposal *models.DatasetProposal) (*models.DatasetProposal, error) {
	log.Info("store.UpdateDatasetProposal()")

	result, err := store(s.db, s.datasetProposalsTable, proposal)
	if err != nil {
		log.Fatalln("store.UpdateDatasetProposal() - store() failed: ", err)
		return nil, err
	}
	log.WithFields(log.Fields{"result": fmt.Sprintf("%+v", result)}).Debug("store.UpdateDatasetProposal()")

	return proposal, nil
}

func (s *publishingStore) DeleteDatasetProposal(proposal *models.DatasetProposal) error {
	log.WithFields(log.Fields{"proposal": fmt.Sprintf("%+v", proposal)}).Info("store.DeleteDatasetProposal()")

	var err error
	proposalKey, err := attributevalue.MarshalMap(models.DatasetProposalKey{
		UserId: proposal.UserId,
		NodeId: proposal.NodeId,
	})
	if err != nil {
		log.Fatalln("store.DeleteDatasetProposal() - MarshalMap() failed: ", err)
		return err
	}
	log.WithFields(log.Fields{"proposalKey": fmt.Sprintf("%+v", proposalKey)}).Debug("store.DeleteDatasetProposal()")

	_, err = s.db.DeleteItem(context.TODO(), &dynamodb.DeleteItemInput{
		TableName: aws.String(s.datasetProposalsTable),
		Key:       proposalKey,
	})

	if err != nil {
		log.Fatalln("store.DeleteDatasetProposal() - DeleteItem() failed: ", err)
		return err
	}

	return nil
}

func (s *publishingStore) GetDatasetProposalForRepository(repositoryId int, status string, nodeId string) (*models.DatasetProposal, error) {
	log.WithFields(log.Fields{"repositoryId": repositoryId, "status": status, "nodeId": nodeId}).Info("store.GetDatasetProposalForRepositoryWithStatus()")

	queryInput := dynamodb.QueryInput{
		TableName:              aws.String(s.datasetProposalsTable),
		IndexName:              aws.String("RepositoryProposalStatusIndex"),
		KeyConditionExpression: aws.String("RepositoryId = :repositoryId AND ProposalStatus = :status"),
		FilterExpression:       aws.String("NodeId = :nodeId"),
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":repositoryId": &types.AttributeValueMemberN{
				Value: intToString(repositoryId),
			},
			":status": &types.AttributeValueMemberS{
				Value: status,
			},
			":nodeId": &types.AttributeValueMemberS{
				Value: nodeId,
			},
		},
		Select: "ALL_PROJECTED_ATTRIBUTES",
	}
	return get[models.DatasetProposal](s.db, &queryInput)
}

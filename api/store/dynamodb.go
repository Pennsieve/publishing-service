package store

import (
	"context"
	"fmt"
	"log/slog"
	"os"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/pennsieve/publishing-service/api/logging"
	"github.com/pennsieve/publishing-service/api/models"
)

type PublishingStore interface {
	GetInfo() ([]models.Info, error)
	GetRepositories() ([]models.Repository, error)
	GetRepository(organizationNodeId string) (*models.Repository, error)
	GetQuestions() ([]models.Question, error)
	GetDatasetProposal(userId int, nodeId string) (*models.DatasetProposal, error)
	GetDatasetProposalsForUser(userId int64) ([]models.DatasetProposal, error)
	GetDatasetProposalsForWorkspace(orgNodeId string, status string) ([]models.DatasetProposal, error)
	GetDatasetProposalForRepository(orgNodeId string, status string, nodeId string) (*models.DatasetProposal, error)
	CreateDatasetProposal(proposal *models.DatasetProposal) (*models.DatasetProposal, error)
	UpdateDatasetProposal(proposal *models.DatasetProposal) (*models.DatasetProposal, error)
	DeleteDatasetProposal(proposal *models.DatasetProposal) error
}

func getTableName(tableName string) string {
	table := os.Getenv(tableName)
	return table
}

// NewPublishingStore builds the DynamoDB-backed store for one request. logger
// is the request-scoped logger built at the entrypoint, held on the struct so
// no method has to reach for slog.Default.
func NewPublishingStore(logger *slog.Logger) *publishingStore {
	// TODO: handle and/or propagate errors
	cfg, err := config.LoadDefaultConfig(context.Background())
	if err != nil {
		logger.Error("config.LoadDefaultConfig() failed building publishing store", slog.Any(logging.KeyError, err))
	}

	db := dynamodb.NewFromConfig(cfg)

	return &publishingStore{
		logger:                logger,
		db:                    db,
		infoTable:             getTableName("PUBLISHING_INFO_TABLE"),
		repositoriesTable:     getTableName("REPOSITORIES_TABLE"),
		questionsTable:        getTableName("REPOSITORY_QUESTIONS_TABLE"),
		datasetProposalsTable: getTableName("DATASET_PROPOSAL_TABLE"),
	}
}

type publishingStore struct {
	logger                *slog.Logger
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

func scan(logger *slog.Logger, client *dynamodb.Client, tableName string) (*dynamodb.ScanOutput, error) {
	scanInput := dynamodb.ScanInput{
		TableName: aws.String(tableName),
	}

	result, err := client.Scan(context.TODO(), &scanInput)
	if err != nil {
		logger.Error("dynamodb Scan() failed",
			slog.String(logging.KeyTable, tableName),
			slog.Any(logging.KeyError, err))
		return nil, err
	}

	return result, nil
}

func query(logger *slog.Logger, client *dynamodb.Client, queryInput *dynamodb.QueryInput) (*dynamodb.QueryOutput, error) {
	result, err := client.Query(context.TODO(), queryInput)
	if err != nil {
		logger.Error("dynamodb Query() failed",
			slog.String(logging.KeyTable, aws.ToString(queryInput.TableName)),
			slog.String(logging.KeyIndex, aws.ToString(queryInput.IndexName)),
			slog.Any(logging.KeyError, err))
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

func fetch[T PublishingTypes](logger *slog.Logger, client *dynamodb.Client, tableName string) ([]T, error) {
	var err error

	// get all Items from the table via Scan operation
	output, err := scan(logger, client, tableName)
	if err != nil {
		return nil, err
	}

	// transform each Item in output from DynamoDB to type T
	results, err := transform[T](output.Items)
	if err != nil {
		logger.Error("failed to unmarshal scanned dynamodb items",
			slog.String(logging.KeyTable, tableName),
			slog.Any(logging.KeyError, err))
		return nil, err
	}

	return results, nil
}

func find[T PublishingTypes](logger *slog.Logger, client *dynamodb.Client, queryInput *dynamodb.QueryInput) ([]T, error) {
	var err error

	output, err := query(logger, client, queryInput)
	if err != nil {
		return nil, err
	}

	// transform each Item in output from DynamoDB to type T
	results, err := transform[T](output.Items)
	if err != nil {
		logger.Error("failed to unmarshal queried dynamodb items",
			slog.String(logging.KeyTable, aws.ToString(queryInput.TableName)),
			slog.Any(logging.KeyError, err))
		return nil, err
	}

	return results, nil
}

func get[T PublishingTypes](logger *slog.Logger, client *dynamodb.Client, queryInput *dynamodb.QueryInput) (*T, error) {
	results, err := find[T](logger, client, queryInput)
	if err != nil {
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
func store(logger *slog.Logger, client *dynamodb.Client, table string, item *models.DatasetProposal) (*dynamodb.PutItemOutput, error) {
	var err error
	data, err := attributevalue.MarshalMap(item)
	if err != nil {
		logger.Error("attributevalue.MarshalMap() failed marshalling proposal",
			slog.String(logging.KeyTable, table),
			slog.String(logging.KeyNodeID, item.NodeId),
			slog.Any(logging.KeyError, err))
		return nil, err
	}

	return client.PutItem(context.TODO(), &dynamodb.PutItemInput{
		TableName: aws.String(table),
		Item:      data,
	})
}

func (s *publishingStore) GetInfo() ([]models.Info, error) {
	return fetch[models.Info](s.logger, s.db, s.infoTable)
}

func (s *publishingStore) GetRepositories() ([]models.Repository, error) {
	return fetch[models.Repository](s.logger, s.db, s.repositoriesTable)
}

func (s *publishingStore) GetRepository(organizationNodeId string) (*models.Repository, error) {
	queryInput := dynamodb.QueryInput{
		TableName:              aws.String(s.repositoriesTable),
		KeyConditionExpression: aws.String("OrganizationNodeId = :organizationNodeId"),
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":organizationNodeId": &types.AttributeValueMemberS{
				Value: organizationNodeId,
			},
		},
	}
	return get[models.Repository](s.logger, s.db, &queryInput)
}

func (s *publishingStore) GetQuestions() ([]models.Question, error) {
	return fetch[models.Question](s.logger, s.db, s.questionsTable)
}

func (s *publishingStore) GetDatasetProposal(userId int, nodeId string) (*models.DatasetProposal, error) {
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
	return get[models.DatasetProposal](s.logger, s.db, &queryInput)
}

func (s *publishingStore) GetDatasetProposalsForUser(userId int64) ([]models.DatasetProposal, error) {
	queryInput := dynamodb.QueryInput{
		TableName:              aws.String(s.datasetProposalsTable),
		KeyConditionExpression: aws.String("UserId = :userId"),
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":userId": &types.AttributeValueMemberN{
				Value: int64ToString(userId),
			},
		},
	}
	return find[models.DatasetProposal](s.logger, s.db, &queryInput)
}

func (s *publishingStore) GetDatasetProposalsForWorkspace(orgNodeId string, status string) ([]models.DatasetProposal, error) {
	queryInput := dynamodb.QueryInput{
		TableName:              aws.String(s.datasetProposalsTable),
		IndexName:              aws.String("RepositoryProposalStatusIndex"),
		KeyConditionExpression: aws.String("OrganizationNodeId = :orgNodeId AND ProposalStatus = :status"),
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":orgNodeId": &types.AttributeValueMemberS{
				Value: orgNodeId,
			},
			":status": &types.AttributeValueMemberS{
				Value: status,
			},
		},
		Select: "ALL_PROJECTED_ATTRIBUTES",
	}
	return find[models.DatasetProposal](s.logger, s.db, &queryInput)
}

func (s *publishingStore) CreateDatasetProposal(proposal *models.DatasetProposal) (*models.DatasetProposal, error) {
	_, err := store(s.logger, s.db, s.datasetProposalsTable, proposal)
	if err != nil {
		s.logger.Error("PutItem() failed creating dataset proposal",
			slog.String(logging.KeyTable, s.datasetProposalsTable),
			slog.String(logging.KeyNodeID, proposal.NodeId),
			slog.Any(logging.KeyError, err))
		return nil, err
	}

	return proposal, nil
}

func (s *publishingStore) UpdateDatasetProposal(proposal *models.DatasetProposal) (*models.DatasetProposal, error) {
	_, err := store(s.logger, s.db, s.datasetProposalsTable, proposal)
	if err != nil {
		s.logger.Error("PutItem() failed updating dataset proposal",
			slog.String(logging.KeyTable, s.datasetProposalsTable),
			slog.String(logging.KeyNodeID, proposal.NodeId),
			slog.Any(logging.KeyError, err))
		return nil, err
	}

	return proposal, nil
}

func (s *publishingStore) DeleteDatasetProposal(proposal *models.DatasetProposal) error {
	var err error
	proposalKey, err := attributevalue.MarshalMap(models.DatasetProposalKey{
		UserId: proposal.UserId,
		NodeId: proposal.NodeId,
	})
	if err != nil {
		s.logger.Error("attributevalue.MarshalMap() failed marshalling proposal key",
			slog.String(logging.KeyNodeID, proposal.NodeId),
			slog.Any(logging.KeyError, err))
		return err
	}

	_, err = s.db.DeleteItem(context.TODO(), &dynamodb.DeleteItemInput{
		TableName: aws.String(s.datasetProposalsTable),
		Key:       proposalKey,
	})

	if err != nil {
		s.logger.Error("DeleteItem() failed deleting dataset proposal",
			slog.String(logging.KeyTable, s.datasetProposalsTable),
			slog.String(logging.KeyNodeID, proposal.NodeId),
			slog.Any(logging.KeyError, err))
		return err
	}

	return nil
}

func (s *publishingStore) GetDatasetProposalForRepository(orgNodeId string, status string, nodeId string) (*models.DatasetProposal, error) {
	queryInput := dynamodb.QueryInput{
		TableName:              aws.String(s.datasetProposalsTable),
		IndexName:              aws.String("RepositoryProposalStatusIndex"),
		KeyConditionExpression: aws.String("OrganizationNodeId = :orgNodeId AND ProposalStatus = :status"),
		FilterExpression:       aws.String("NodeId = :nodeId"),
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":orgNodeId": &types.AttributeValueMemberS{
				Value: orgNodeId,
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
	return get[models.DatasetProposal](s.logger, s.db, &queryInput)
}

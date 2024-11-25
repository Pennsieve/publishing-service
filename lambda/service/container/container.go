package container

import (
	"context"
	"database/sql"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/aws/retry"
	awsConfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/pennsieve/pennsieve-go-core/pkg/queries/pgdb"
	"github.com/pennsieve/publishing-service/api/notification"
	"github.com/pennsieve/publishing-service/api/service"
	"github.com/pennsieve/publishing-service/api/store"
	"github.com/pennsieve/publishing-service/service/config"
)

type PublishingServices interface {
	Postgres() *sql.DB
	PublishingStore() *store.ThePublishingStore
	PennsieveStore() *store.ThePennsieveStore
	EmailNotifier() *notification.EmailNotifier
	PublishingService() *service.ThePublishingService
}

type Container struct {
	AwsConfig         aws.Config
	Config            config.Config
	OrganizationId    int64
	postgres          *sql.DB
	dynamoDb          *dynamodb.Client
	publishingStore   *store.ThePublishingStore
	pennsieveStore    *store.ThePennsieveStore
	emailNotifier     *notification.EmailNotifier
	publishingService *service.ThePublishingService
}

func NewContainer(ctx context.Context) (*Container, error) {
	var organizationId int64
	organizationIdValue := ctx.Value("organizationId")
	if organizationId == nil {
		organizationId = 0
	} else {
		organizationId = organizationIdValue.(int64)
	}

	config := config.LoadConfig()

	awsConfig, err := awsConfig.LoadDefaultConfig(context.Background())
	if err != nil {
		return nil, err
	}
	awsConfig.Retryer = func() aws.Retryer {
		return retry.NewStandard()
	}

	return NewContainerFromConfig(config, awsConfig, organizationId), nil
}

func NewContainerFromConfig(config config.Config, awsConfig aws.Config, organizationId int64) *Container {
	return &Container{
		Config:         config,
		AwsConfig:      awsConfig,
		OrganizationId: organizationId,
	}
}

func (c *Container) Postgres() *sql.DB {
	if c.postgres == nil {
		var err error
		var db *sql.DB
		if c.OrganizationId != 0 {
			db, err = pgdb.ConnectRDSWithOrg(int(c.OrganizationId))
		} else {
			db, err = pgdb.ConnectRDS()
		}
		if err != nil {
			panic(err)
		}
		c.postgres = db
	}
	return c.postgres
}

func (c *Container) PublishingStore() *store.ThePublishingStore {
	if c.publishingStore == nil {
		c.publishingStore = store.NewPublishingStore()
	}
	return c.publishingStore
}

func (c *Container) PennsieveStore() *store.ThePennsieveStore {
	if c.pennsieveStore == nil {
		c.pennsieveStore = store.NewPennsieveStore(c.Postgres(), c.OrganizationId)
	}
	return c.pennsieveStore
}

func (c *Container) EmailNotifier() *notification.EmailNotifier {
	if c.emailNotifier == nil {
		c.emailNotifier = notification.NewEmailNotifier(context.TODO())
	}
	return c.emailNotifier
}

func (c *Container) PublishingService() *service.ThePublishingService {
	if c.publishingService == nil {
		pubStore := c.PublishingStore()
		pennsieve := c.PennsieveStore()
		notifier := c.EmailNotifier()
		c.publishingService = service.NewPublishingService(pubStore, pennsieve, notifier)
	}
	return c.publishingService
}

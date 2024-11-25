package config

import (
	"log"
	"os"
	"strconv"
)

const DynamoDBMaxBatchWriteItem = 25

type Config struct {
	DynamoDB   DynamoDBConfig
	PostgresDB PostgresDBConfig
	SNS        SNSConfig
	S3         S3Config
}

type DynamoDBConfig struct {
	Endpoint                string
	ImportManifestTableName string
}

type PostgresDBConfig struct {
	Host                 string
	Port                 int
	User                 string
	Password             *string
	TimeSeriesDatabase   string
	OrganizationDatabase string
}

type SNSConfig struct {
	TimeSeriesImporterTopicARN string
}

type S3Config struct {
	TimeSeriesBucket string
	ImportBucket     string
}

func LoadConfig() Config {
	return Config{
		DynamoDB: DynamoDBConfig{
			Endpoint:                getEnvOrDefault("DYNAMODB_ENDPOINT", "http://localhost:8000"),
			ImportManifestTableName: getEnv("IMPORT_MANIFEST_TABLE_NAME"),
		},
		PostgresDB: PostgresDBConfig{
			Host:                 getEnvOrDefault("POSTGRES_HOST", "localhost"),
			Port:                 Atoi(getEnvOrDefault("POSTGRES_PORT", "5432")),
			User:                 getEnv("POSTGRES_USER"),
			Password:             getEnvOrNil("POSTGRES_PASSWORD"),
			TimeSeriesDatabase:   getEnvOrDefault("POSTGRES_TIMESERIES_DATABASE", "data_postgres"),
			OrganizationDatabase: getEnvOrDefault("POSTGRES_ORGANIZATION_DATABASE", "pennsieve_postgres"),
		},
		SNS: SNSConfig{
			TimeSeriesImporterTopicARN: getEnv("IMPORT_TIMESERIES_IMPORTER_SNS_TOPIC_ARN"),
		},
		S3: S3Config{
			TimeSeriesBucket: getEnv("TIMESERIES_BUCKET"),
			ImportBucket:     getEnv("IMPORT_BUCKET"),
		},
	}
}

func getEnv(key string) string {
	value, exists := os.LookupEnv(key)

	if !exists {
		log.Fatalf("Failed to load '%s' from environment", key)
	}

	return value
}

func getEnvOrDefault(key string, defaultValue string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	} else {
		return defaultValue
	}
}

func getEnvOrNil(key string) *string {
	if value, exists := os.LookupEnv(key); exists {
		return &value
	} else {
		return nil
	}
}

func Atoi(value string) int {
	i, err := strconv.Atoi(value)

	if err != nil {
		log.Fatalf("Failed to convert '%s' integer", value)
	}

	return i
}

package s3

import (
	"context"
	"fmt"
	"io"
	"log/slog"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/pennsieve/publishing-service/api/logging"
)

// MakeFileReader builds a FileReader. logger is the request-scoped logger built
// at the entrypoint, held on the struct so no method has to reach for
// slog.Default.
func MakeFileReader(logger *slog.Logger) *FileReader {
	cfg, err := config.LoadDefaultConfig(context.Background())
	if err != nil {
		logger.Error("config.LoadDefaultConfig() failed building file reader", slog.Any(logging.KeyError, err))
	}
	s3Client := s3.NewFromConfig(cfg)

	return &FileReader{logger: logger, s3Client: s3Client}
}

type FileReader struct {
	logger   *slog.Logger
	s3Client *s3.Client
}

func (reader *FileReader) ReadFile(ctx context.Context, s3Bucket string, s3Key string) (string, error) {
	s3GetObjectInput := &s3.GetObjectInput{
		Bucket: aws.String(s3Bucket),
		Key:    aws.String(s3Key),
	}

	result, err := reader.s3Client.GetObject(ctx, s3GetObjectInput)
	if err != nil {
		// TODO: better error handling
		reader.logger.Error("s3 GetObject() failed",
			slog.String(logging.KeyS3Bucket, s3Bucket),
			slog.String(logging.KeyS3Key, s3Key),
			slog.Any(logging.KeyError, err))
		return "", err
	}

	defer result.Body.Close()
	body, err := io.ReadAll(result.Body)
	if err != nil {
		// TODO: better error handling
		reader.logger.Error("failed to read the S3 object body",
			slog.String(logging.KeyS3Bucket, s3Bucket),
			slog.String(logging.KeyS3Key, s3Key),
			slog.Any(logging.KeyError, err))
		return "", err
	}
	bodyString := fmt.Sprintf("%s", body)

	return bodyString, nil
}

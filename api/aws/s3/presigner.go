package s3

// Reference:
//     https://docs.aws.amazon.com/code-library/latest/ug/go_2_s3_code_examples.html

import (
	"context"
	"log/slog"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	v4 "github.com/aws/aws-sdk-go-v2/aws/signer/v4"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/pennsieve/publishing-service/api/logging"
)

// MakePresigner builds a Presigner. logger is the request-scoped logger built
// at the entrypoint, held on the struct so no method has to reach for
// slog.Default.
func MakePresigner(logger *slog.Logger) *Presigner {
	cfg, err := config.LoadDefaultConfig(context.Background())
	if err != nil {
		logger.Error("config.LoadDefaultConfig() failed building presigner", slog.Any(logging.KeyError, err))
	}
	s3Client := s3.NewFromConfig(cfg)
	presignClient := s3.NewPresignClient(s3Client)
	return &Presigner{
		logger:        logger,
		PresignClient: presignClient,
	}
}

// Presigner encapsulates the Amazon Simple Storage Service (Amazon S3) presign actions
// used in the examples.
// It contains PresignClient, a client that is used to presign requests to Amazon S3.
// Presigned requests contain temporary credentials and can be made from any HTTP client.
type Presigner struct {
	logger        *slog.Logger
	PresignClient *s3.PresignClient
}

// GetObject makes a presigned request that can be used to get an object from a bucket.
// The presigned request is valid for the specified number of seconds.
func (presigner Presigner) GetObject(
	bucketName string, objectKey string, lifetimeSecs int64) (*v4.PresignedHTTPRequest, error) {
	request, err := presigner.PresignClient.PresignGetObject(context.TODO(), &s3.GetObjectInput{
		Bucket: aws.String(bucketName),
		Key:    aws.String(objectKey),
	}, func(opts *s3.PresignOptions) {
		opts.Expires = time.Duration(lifetimeSecs * int64(time.Second))
	})
	if err != nil {
		presigner.logger.Error("failed to presign an S3 GetObject request",
			slog.String(logging.KeyS3Bucket, bucketName),
			slog.String(logging.KeyS3Key, objectKey),
			slog.Any(logging.KeyError, err))
	}
	return request, err
}

// PutObject makes a presigned request that can be used to put an object in a bucket.
// The presigned request is valid for the specified number of seconds.
func (presigner Presigner) PutObject(
	bucketName string, objectKey string, lifetimeSecs int64) (*v4.PresignedHTTPRequest, error) {
	request, err := presigner.PresignClient.PresignPutObject(context.TODO(), &s3.PutObjectInput{
		Bucket: aws.String(bucketName),
		Key:    aws.String(objectKey),
	}, func(opts *s3.PresignOptions) {
		opts.Expires = time.Duration(lifetimeSecs * int64(time.Second))
	})
	if err != nil {
		presigner.logger.Error("failed to presign an S3 PutObject request",
			slog.String(logging.KeyS3Bucket, bucketName),
			slog.String(logging.KeyS3Key, objectKey),
			slog.Any(logging.KeyError, err))
	}
	return request, err
}

// DeleteObject makes a presigned request that can be used to delete an object from a bucket.
func (presigner Presigner) DeleteObject(bucketName string, objectKey string) (*v4.PresignedHTTPRequest, error) {
	request, err := presigner.PresignClient.PresignDeleteObject(context.TODO(), &s3.DeleteObjectInput{
		Bucket: aws.String(bucketName),
		Key:    aws.String(objectKey),
	})
	if err != nil {
		presigner.logger.Error("failed to presign an S3 DeleteObject request",
			slog.String(logging.KeyS3Bucket, bucketName),
			slog.String(logging.KeyS3Key, objectKey),
			slog.Any(logging.KeyError, err))
	}
	return request, err
}

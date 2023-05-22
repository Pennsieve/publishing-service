package s3

import (
	"context"
	"fmt"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"io/ioutil"
)

func MakeFileReader() *FileReader {
	cfg, err := config.LoadDefaultConfig(context.Background())
	if err != nil {
		// TODO: handle error
	}
	s3Client := s3.NewFromConfig(cfg)

	return &FileReader{s3Client: s3Client}
}

type FileReader struct {
	s3Client *s3.Client
}

func (reader *FileReader) ReadFile(ctx context.Context, s3Bucket string, s3Key string) (*string, error) {
	s3GetObjectInput := &s3.GetObjectInput{
		Bucket: aws.String(s3Bucket),
		Key:    aws.String(s3Key),
	}

	result, err := reader.s3Client.GetObject(ctx, s3GetObjectInput)
	if err != nil {
		// TODO: better error handling
		return nil, err
	}

	defer result.Body.Close()
	body, err := ioutil.ReadAll(result.Body)
	if err != nil {
		// TODO: better error handling
		return nil, err
	}
	bodyString := fmt.Sprintf("%s", body)

	return &bodyString, nil
}

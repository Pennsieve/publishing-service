package handler

import (
	"context"
	"github.com/aws/aws-lambda-go/events"
	"github.com/pennsieve/publishing-service/service/container"
)

func SubmitDatasetProposal(ctx context.Context, request events.APIGatewayV2HTTPRequest, container *container.Container) (events.APIGatewayV2HTTPResponse, error) {
	return events.APIGatewayV2HTTPResponse{}, nil
}

func WithdrawDatasetProposal(ctx context.Context, request events.APIGatewayV2HTTPRequest, container *container.Container) (events.APIGatewayV2HTTPResponse, error) {
	return events.APIGatewayV2HTTPResponse{}, nil
}

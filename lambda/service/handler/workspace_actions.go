package handler

import (
	"context"
	"github.com/aws/aws-lambda-go/events"
	"github.com/pennsieve/publishing-service/service/container"
)

func GetWorkspaceDatasetProposals(ctx context.Context, request events.APIGatewayV2HTTPRequest, container *container.Container) (events.APIGatewayV2HTTPResponse, error) {
	return events.APIGatewayV2HTTPResponse{}, nil
}

func AcceptDatasetProposal(ctx context.Context, request events.APIGatewayV2HTTPRequest, container *container.Container) (events.APIGatewayV2HTTPResponse, error) {
	return events.APIGatewayV2HTTPResponse{}, nil
}

func RejectDatasetProposal(ctx context.Context, request events.APIGatewayV2HTTPRequest, container *container.Container) (events.APIGatewayV2HTTPResponse, error) {
	return events.APIGatewayV2HTTPResponse{}, nil
}

package handler

import (
	"context"
	"encoding/json"
	"github.com/aws/aws-lambda-go/events"
	"github.com/pennsieve/publishing-service/service/container"
	"github.com/pennsieve/publishing-service/service/utils"
	"net/http"
)

func GetPublishingQuestions(ctx context.Context, request events.APIGatewayV2HTTPRequest, container *container.Container) (events.APIGatewayV2HTTPResponse, error) {
	// handlerName := "GetPublishingQuestions"
	service := container.PublishingService()
	result, err := service.GetProposalQuestions()
	if err != nil {
		return events.APIGatewayV2HTTPResponse{}, err
	}

	jsonBody, err := json.Marshal(result)
	if err != nil {
		return events.APIGatewayV2HTTPResponse{}, err
	}

	return events.APIGatewayV2HTTPResponse{
		StatusCode: http.StatusOK,
		Body:       string(jsonBody),
		Headers:    utils.StandardResponseHeaders(nil),
	}, nil
}

package handler

import (
	"context"
	"encoding/json"
	"github.com/aws/aws-lambda-go/events"
	"github.com/pennsieve/pennsieve-go-core/pkg/authorizer"
	"github.com/pennsieve/publishing-service/service/container"
	"github.com/pennsieve/publishing-service/service/utils"
	log "github.com/sirupsen/logrus"
	"net/http"
)

func GetUserDatasetProposals(ctx context.Context, request events.APIGatewayV2HTTPRequest, container *container.Container) (events.APIGatewayV2HTTPResponse, error) {
	claims := authorizer.ParseClaims(request.RequestContext.Authorizer.Lambda)

	// get user id from User Claim
	userId := claims.UserClaim.Id

	service := container.PublishingService()

	result, err := service.GetDatasetProposalsForUser(userId)
	if err != nil {
		log.Error("service.GetDatasetProposalsForUser() failed: ", err)
		return events.APIGatewayV2HTTPResponse{}, err
	}

	jsonBody, err := json.Marshal(result)
	if err != nil {
		log.Error("json.Marshal() failed: ", err)
		return events.APIGatewayV2HTTPResponse{}, err
	}

	return events.APIGatewayV2HTTPResponse{
		StatusCode: http.StatusOK,
		Body:       string(jsonBody),
		Headers:    utils.StandardResponseHeaders(nil),
	}, nil
}

func CreateDatasetProposal(ctx context.Context, request events.APIGatewayV2HTTPRequest, container *container.Container) (events.APIGatewayV2HTTPResponse, error) {
	return events.APIGatewayV2HTTPResponse{}, nil
}

func UpdateDatasetProposal(ctx context.Context, request events.APIGatewayV2HTTPRequest, container *container.Container) (events.APIGatewayV2HTTPResponse, error) {
	return events.APIGatewayV2HTTPResponse{}, nil
}

func DeleteDatasetProposal(ctx context.Context, request events.APIGatewayV2HTTPRequest, container *container.Container) (events.APIGatewayV2HTTPResponse, error) {
	return events.APIGatewayV2HTTPResponse{}, nil
}

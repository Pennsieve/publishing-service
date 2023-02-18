package handler

import (
	"encoding/json"
	"fmt"
	"github.com/aws/aws-lambda-go/events"
	"github.com/pennsieve/pennsieve-go-api/pkg/authorizer"
	"github.com/pennsieve/publishing-service/api/dtos"
	"github.com/pennsieve/publishing-service/api/service"
	"github.com/pennsieve/publishing-service/api/store"
	log "github.com/sirupsen/logrus"
	"os"
	"regexp"
	"strconv"
)

func init() {
	log.SetFormatter(&log.JSONFormatter{})
	ll, err := log.ParseLevel(os.Getenv("LOG_LEVEL"))
	if err != nil {
		log.SetLevel(log.InfoLevel)
	} else {
		log.SetLevel(ll)
	}
}

func PublishingServiceHandler(request events.APIGatewayV2HTTPRequest) (*events.APIGatewayV2HTTPResponse, error) {
	var err error
	var response *events.APIGatewayV2HTTPResponse

	log.Println("PublishingServiceHandler() ")

	store := store.NewPublishingStore()
	service := service.NewPublishingService(store)

	if err != nil {
		log.Fatalln("publishingService.GetPublishingRepositories() failed")
	}

	response, err = handleRequest(request, service)

	return response, err
}

func handleRequest(request events.APIGatewayV2HTTPRequest, service service.PublishingService) (*events.APIGatewayV2HTTPResponse, error) {
	log.Println("handleRequest()")

	var err error
	var jsonBody []byte

	claims := authorizer.ParseClaims(request.RequestContext.Authorizer.Lambda)

	r := regexp.MustCompile(`(?P<method>) (?P<pathKey>.*)`)
	routeKeyParts := r.FindStringSubmatch(request.RouteKey)
	routeKey := routeKeyParts[r.SubexpIndex("pathKey")]

	log.Println("handleRequest() routeKey: ", routeKey)

	// TODO: handle errors

	switch routeKey {
	case "/publishing/repositories":
		switch request.RequestContext.HTTP.Method {
		case "GET":
			result, _ := service.GetPublishingRepositories()
			// Parse response into JSON structure
			jsonBody, err = json.Marshal(result)
		}
	case "/publishing/questions":
		switch request.RequestContext.HTTP.Method {
		case "GET":
			result, _ := service.GetProposalQuestions()
			jsonBody, err = json.Marshal(result)
		}
	case "/publishing/proposals":
		switch request.RequestContext.HTTP.Method {
		case "GET":
			result, _ := handleGetDatasetProposals(request, claims, service)
			jsonBody, err = json.Marshal(result)
		}
	}

	jsonString := string(jsonBody)
	log.Println("handleRequest() jsonString: ", jsonString)

	response := events.APIGatewayV2HTTPResponse{Body: jsonString, StatusCode: 200}
	log.Println("handleRequest() response: ", response)

	return &response, err
}

func handleGetDatasetProposals(request events.APIGatewayV2HTTPRequest, claims *authorizer.Claims, service service.PublishingService) ([]dtos.DatasetProposalDTO, error) {
	// get query params
	queryParams := request.QueryStringParameters
	userIdString, userIdFound := queryParams["user_id"]
	workspaceIdString, workspaceIdFound := queryParams["workspace_id"]

	// if user_id and workspace_id are both present, then error
	if userIdFound && workspaceIdFound {
		return nil, fmt.Errorf("invalid request: cannot provide both user_id and workspace_id")
	}

	// if user_id and workspace_id are both absent, then error
	if !userIdFound && !workspaceIdFound {
		return nil, fmt.Errorf("invalid request: must provide user_id or workspace_id")
	}

	// if user_id provided, then validate authorized by User claim
	if userIdFound {
		userId := stringToInt64(userIdString)
		if isAuthorizedUser(userId, claims) {
			return service.GetDatasetProposalsForUser(userId)
		}

	}

	// if workspace_id provided, then validate authorized by Organization claim (and Team?)
	if workspaceIdFound {
		workspaceId := stringToInt64(workspaceIdString)
		if isAuthorizedWorkspace(workspaceId, claims) {
			return service.GetDatasetProposalsForWorkspace(workspaceId)
		}
		return nil, fmt.Errorf("unauthorized")
	}
}

func stringToInt64(value string) int64 {
	result, err := strconv.ParseInt(value, 10, 64)
	if err != nil {
		return -1
	}
	return result
}

func isAuthorizedUser(userId int64, claims *authorizer.Claims) bool {
	return userId == claims.UserClaim.Id
}

func isAuthorizedWorkspace(workspaceId int64, claims *authorizer.Claims) bool {
	// may need to iterate? (if a member of multiple workspaces)
	return workspaceId == claims.OrgClaim.IntId
}

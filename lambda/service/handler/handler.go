package handler

import (
	"encoding/json"
	"github.com/aws/aws-lambda-go/events"
	"github.com/pennsieve/pennsieve-go-api/pkg/authorizer"
	"github.com/pennsieve/publishing-service/api/dtos"
	"github.com/pennsieve/publishing-service/api/service"
	"github.com/pennsieve/publishing-service/api/store"
	log "github.com/sirupsen/logrus"
	"github.com/valyala/fastjson"
	"os"
	"regexp"
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
	var statusCode int
	var jsonBody []byte

	claims := authorizer.ParseClaims(request.RequestContext.Authorizer.Lambda)

	r := regexp.MustCompile(`(?P<method>) (?P<pathKey>.*)`)
	routeKeyParts := r.FindStringSubmatch(request.RouteKey)
	routeKey := routeKeyParts[r.SubexpIndex("pathKey")]

	log.Println("handleRequest() routeKey: ", routeKey)

	// TODO: create function for each invocation
	// TODO: each invocation function shall invoke the Service to process the request
	// TODO: each invocation function shall return an APIGatewayV2HTTPResponse, error
	// TODO: each invocation function shall format success responses (in JSON)
	// TODO: each invocation function shall generate error responses
	// TODO: if an invocation function returns an error, the top-level handler will generate a 500 with the error

	// TODO: figure out authorization

	switch routeKey {
	case "/publishing/repositories":
		switch request.RequestContext.HTTP.Method {
		case "GET":
			jsonBody, statusCode = handleGetPublishingRepositories(service)
		}
	case "/publishing/questions":
		switch request.RequestContext.HTTP.Method {
		case "GET":
			jsonBody, statusCode = handleGetProposalQuestions(service)
		}
	case "/publishing/proposal":
		switch request.RequestContext.HTTP.Method {
		case "GET":
			if ok := authorized(); ok {
				jsonBody, statusCode = handleGetUserDatasetProposals(claims, service)
			} else {
				jsonBody = nil
				statusCode = 401
			}
		case "POST":
			jsonBody, statusCode = handleCreateDatasetProposal(request, claims, service)
		}
	case "/publishing/submission":
		switch request.RequestContext.HTTP.Method {
		case "GET":
			if ok := authorized(); ok {
				jsonBody, statusCode = handleGetWorkspaceDatasetProposals(claims, service)
			} else {
				jsonBody = nil
				statusCode = 401
			}
		}
	}

	jsonString := string(jsonBody)
	log.Println("handleRequest() jsonString: ", jsonString)

	response := events.APIGatewayV2HTTPResponse{Body: jsonString, StatusCode: statusCode}
	log.Println("handleRequest() response: ", response)

	return &response, err
}

// TODO: figure out authorization
func authorized() bool {
	return true
}

//func handleTheRequest() ([]byte, int) {
//	// invoke service.Function()
//	// check return; map err to HTTP Status code
//
//	// marshall service response
//	jsonBody, err := json.Marshal(nil)
//	if err != nil {
//		return nil, 500
//	}
//	statusCode := 200
//	return jsonBody, statusCode
//}

func handleGetPublishingRepositories(service service.PublishingService) ([]byte, int) {
	result, err := service.GetPublishingRepositories()
	if err != nil {
		// TODO: provide a better response than nil on a 500
		return nil, 500
	}

	jsonBody, err := json.Marshal(result)
	if err != nil {
		// TODO: provide a better response than nil on a 500
		return nil, 500
	}

	return jsonBody, 200
}

func handleGetProposalQuestions(service service.PublishingService) ([]byte, int) {
	result, err := service.GetProposalQuestions()
	if err != nil {
		// TODO: provide a better response than nil on a 500
		return nil, 500
	}

	jsonBody, err := json.Marshal(result)
	if err != nil {
		// TODO: provide a better response than nil on a 500
		return nil, 500
	}

	return jsonBody, 200
}

func handleGetUserDatasetProposals(claims *authorizer.Claims, service service.PublishingService) ([]byte, int) {
	// get user id from User Claim
	id := claims.UserClaim.Id

	result, err := service.GetDatasetProposalsForUser(id)
	if err != nil {
		// TODO: provide a better response than nil on a 500
		return nil, 500
	}

	jsonBody, err := json.Marshal(result)
	if err != nil {
		// TODO: provide a better response than nil on a 500
		return nil, 500
	}

	return jsonBody, 200
}

// TODO: ensure the user in on Publishers Team in the Workspace
func handleGetWorkspaceDatasetProposals(claims *authorizer.Claims, service service.PublishingService) ([]byte, int) {
	// get workspace id from Organization Claim
	id := claims.OrgClaim.IntId

	result, err := service.GetDatasetProposalsForWorkspace(id)
	if err != nil {
		// TODO: provide a better response than nil on a 500
		return nil, 500
	}

	jsonBody, err := json.Marshal(result)
	if err != nil {
		// TODO: provide a better response than nil on a 500
		return nil, 500
	}

	return jsonBody, 200
}

func handleCreateDatasetProposal(request events.APIGatewayV2HTTPRequest, claims *authorizer.Claims, service service.PublishingService) ([]byte, int) {
	log.Println("handleCreateDatasetProposal()")
	err := fastjson.Validate(request.Body)
	if err != nil {
		log.Fatalln("handleCreateDatasetProposal() request body validation failed: ", err)
		return nil, 500
	}

	// Unmarshal JSON into Dataset Proposal DTO
	bytes := []byte(request.Body)
	var requestDTO dtos.DatasetProposalDTO
	json.Unmarshal(bytes, &requestDTO)

	log.Println("handleCreateDatasetProposal() requestDTO: %#v", requestDTO)

	resultDTO, err := service.CreateDatasetProposal(int(claims.UserClaim.Id), requestDTO)
	if err != nil {
		log.Fatalln("handleCreateDatasetProposal() - service.CreateDatasetProposal() failed: ", err)
		return nil, 500
	}
	log.Println("handleCreateDatasetProposal() resultDTO: %#v", resultDTO)

	jsonBody, err := json.Marshal(resultDTO)
	if err != nil {
		log.Fatalln("handleCreateDatasetProposal() - json.Marshal() failed: ", err)
		// TODO: provide a better response than nil on a 500
		return nil, 500
	}

	return jsonBody, 201
}

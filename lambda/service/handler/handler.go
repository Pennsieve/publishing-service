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
	"github.com/valyala/fastjson"
	"os"
	"regexp"
)

func init() {
	log.SetFormatter(&log.JSONFormatter{})
	ll, err := log.ParseLevel(os.Getenv("LOG_LEVEL"))
	if err != nil {
		log.SetLevel(log.DebugLevel)
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
	httpMethod := request.RequestContext.HTTP.Method

	log.WithFields(log.Fields{"method": httpMethod, "route": routeKey}).Info("handleRequest()")

	// TODO: create function for each invocation
	// TODO: each invocation function shall invoke the Service to process the request
	// TODO: each invocation function shall return an APIGatewayV2HTTPResponse, error
	// TODO: each invocation function shall format success responses (in JSON)
	// TODO: each invocation function shall generate error responses
	// TODO: if an invocation function returns an error, the top-level handler will generate a 500 with the error

	// TODO: figure out authorization

	switch routeKey {
	case "/publishing/repositories":
		switch httpMethod {
		case "GET":
			jsonBody, statusCode = handleGetPublishingRepositories(service)
		}
	case "/publishing/questions":
		switch httpMethod {
		case "GET":
			jsonBody, statusCode = handleGetProposalQuestions(service)
		}
	case "/publishing/proposal":
		switch httpMethod {
		case "GET":
			if ok := authorized(); ok {
				jsonBody, statusCode = handleGetUserDatasetProposals(claims, service)
			} else {
				jsonBody = nil
				statusCode = 401
			}
		case "POST":
			jsonBody, statusCode = handleCreateDatasetProposal(request, claims, service)
		case "PUT":
			jsonBody, statusCode = handleUpdateDatasetProposal(request, claims, service)
		case "DELETE":
			jsonBody, statusCode = handleDeleteDatasetProposal(request, claims, service)
		}
	case "/publishing/submission":
		switch httpMethod {
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
	log.Info("handleGetUserDatasetProposals()")
	// get user id from User Claim
	userId := claims.UserClaim.Id
	log.WithFields(log.Fields{"userId": userId}).Debug("handleGetUserDatasetProposals()")

	result, err := service.GetDatasetProposalsForUser(userId)
	if err != nil {
		log.Error("service.GetDatasetProposalsForUser() failed: ", err)
		return nil, 500
	}

	jsonBody, err := json.Marshal(result)
	if err != nil {
		log.Error("json.Marshal() failed: ", err)
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
	log.WithFields(log.Fields{"requestDTO": fmt.Sprintf("%+v", requestDTO)}).Debug("handleCreateDatasetProposal()")

	resultDTO, err := service.CreateDatasetProposal(int(claims.UserClaim.Id), requestDTO)
	if err != nil {
		log.Fatalln("handleCreateDatasetProposal() - service.CreateDatasetProposal() failed: ", err)
		return nil, 500
	}
	log.WithFields(log.Fields{"resultDTO": fmt.Sprintf("%+v", resultDTO)}).Debug("handleCreateDatasetProposal()")

	jsonBody, err := json.Marshal(resultDTO)
	if err != nil {
		log.Fatalln("handleCreateDatasetProposal() - json.Marshal() failed: ", err)
		// TODO: provide a better response than nil on a 500
		return nil, 500
	}

	return jsonBody, 201
}

func handleUpdateDatasetProposal(request events.APIGatewayV2HTTPRequest, claims *authorizer.Claims, service service.PublishingService) ([]byte, int) {
	log.WithFields(log.Fields{"request.body": request.Body}).Debug("handleUpdateDatasetProposal()")

	var err error

	// validate JSON
	err = fastjson.Validate(request.Body)
	if err != nil {
		log.WithFields(log.Fields{"request.Body": request.Body}).Error("request body validation failed: ", err)
		return nil, 400
	}

	// Unmarshal JSON into Dataset Proposal DTO
	bytes := []byte(request.Body)
	var requestDTO dtos.DatasetProposalDTO
	json.Unmarshal(bytes, &requestDTO)
	log.WithFields(log.Fields{"requestDTO": fmt.Sprintf("%+v", requestDTO)}).Debug("handleUpdateDatasetProposal()")

	// check that ProposalNodeId was provided
	if requestDTO.ProposalNodeId == "" {
		log.WithFields(log.Fields{}).Error("missing required field(s): ProposalNodeId")
		return nil, 400
	}

	// get Proposal by UserId and ProposalNodeId
	_, err = service.GetDatasetProposal(requestDTO.UserId, requestDTO.ProposalNodeId)
	if err != nil {
		log.WithFields(log.Fields{"UserId": requestDTO.UserId, "ProposalNodeId": requestDTO.ProposalNodeId}).Error("Dataset Proposal does not exist")
		return nil, 404
	}

	// if it exists, then invoke update
	resultDTO, err := service.UpdateDatasetProposal(int(claims.UserClaim.Id), requestDTO)
	if err != nil {
		log.Error("service.UpdateDatasetProposal() failed: ", err)
		return nil, 500
	}
	log.WithFields(log.Fields{"resultDTO": fmt.Sprintf("%+v", resultDTO)}).Debug("handleCreateDatasetProposal()")

	jsonBody, err := json.Marshal(resultDTO)
	if err != nil {
		log.Error("json.Marshal() failed: ", err)
		return nil, 500
	}

	return jsonBody, 200
}

func handleDeleteDatasetProposal(request events.APIGatewayV2HTTPRequest, claims *authorizer.Claims, service service.PublishingService) ([]byte, int) {
	log.WithFields(log.Fields{}).Debug("handleDeleteDatasetProposal()")

	var err error
	var nodeId string
	var found bool

	// get ProposalNodeId from request query parameters
	queryParams := request.QueryStringParameters
	if nodeId, found = queryParams["proposal_node_id"]; !found {
		return nil, 400
	}

	userId := int(claims.UserClaim.Id)

	proposal, err := service.GetDatasetProposal(userId, nodeId)
	if err != nil {
		// probably not found
		return nil, 404
	}
	log.WithFields(log.Fields{"proposal": fmt.Sprintf("%+v", proposal)}).Debug("handleDeleteDatasetProposal() found proposal")

	_, err = service.DeleteDatasetProposal(proposal)
	if err != nil {
		// TODO: log an error message
		return nil, 500
	}

	return nil, 200
}

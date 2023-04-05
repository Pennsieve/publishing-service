package handler

import (
	"encoding/json"
	"fmt"
	"github.com/aws/aws-lambda-go/events"
	"github.com/pennsieve/pennsieve-go-core/pkg/authorizer"
	"github.com/pennsieve/pennsieve-go-core/pkg/queries/pgdb"
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

	response, err = handleRequest(request)

	return response, err
}

func handleRequest(request events.APIGatewayV2HTTPRequest) (*events.APIGatewayV2HTTPResponse, error) {
	log.Info("handleRequest()")

	var err error
	var statusCode int
	var jsonBody []byte

	claims := authorizer.ParseClaims(request.RequestContext.Authorizer.Lambda)
	orgId := claims.OrgClaim.IntId

	db, err := pgdb.ConnectRDSWithOrg(int(orgId))
	if err != nil {
		panic(fmt.Sprintf("unable to connect to RDS database: %s", err))
	}
	log.Info("connected to RDS database")

	pubStore := store.NewPublishingStore()
	pennsieve := store.NewPennsieveStore(db, orgId)
	service := service.NewPublishingService(pubStore, pennsieve)

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
	case "/publishing/proposal/submit":
		switch httpMethod {
		case "POST":
			jsonBody, statusCode = handleSubmitDatasetProposal(request, claims, service)
		}
	case "/publishing/proposal/withdraw":
		switch httpMethod {
		case "POST":
			jsonBody, statusCode = handleWithdrawDatasetProposal(request, claims, service)
		}
	case "/publishing/submission":
		switch httpMethod {
		case "GET":
			if ok := authorized(); ok {
				jsonBody, statusCode = handleGetWorkspaceDatasetProposals(request, claims, service)
			} else {
				jsonBody = nil
				statusCode = 401
			}
		}
	case "/publishing/submission/accept":
		switch httpMethod {
		case "POST":
			jsonBody, statusCode = handleAcceptDatasetProposal(request, claims, service)
		}
	case "/publishing/submission/reject":
		switch httpMethod {
		case "POST":
			jsonBody, statusCode = handleRejectDatasetProposal(request, claims, service)
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
func handleGetWorkspaceDatasetProposals(request events.APIGatewayV2HTTPRequest, claims *authorizer.Claims, service service.PublishingService) ([]byte, int) {
	// get workspace id from Organization Claim
	id := claims.OrgClaim.IntId

	// get ProposalNodeId from request query parameters
	var status string
	var found bool
	queryParams := request.QueryStringParameters
	if status, found = queryParams["status"]; !found {
		status = "SUBMITTED"
	}

	result, err := service.GetDatasetProposalsForWorkspace(id, status)
	if err != nil {
		// TODO: provide a better response than nil on a 500
		return nil, 500
	}

	response := &dtos.DatasetSubmissionsDTO{
		TotalCount: len(result),
		Proposals:  result,
	}

	jsonBody, err := json.Marshal(response)
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
	if requestDTO.NodeId == "" {
		log.WithFields(log.Fields{}).Error("missing required field(s): ProposalNodeId")
		return nil, 400
	}

	// get Proposal by UserId and ProposalNodeId
	proposal, err := service.GetDatasetProposal(requestDTO.UserId, requestDTO.NodeId)
	if err != nil {
		log.WithFields(log.Fields{"UserId": requestDTO.UserId, "NodeId": requestDTO.NodeId}).Error("Dataset Proposal does not exist")
		return nil, 404
	}

	// if it exists, then invoke update
	resultDTO, err := service.UpdateDatasetProposal(int(claims.UserClaim.Id), proposal, requestDTO)
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

func handleSubmitDatasetProposal(request events.APIGatewayV2HTTPRequest, claims *authorizer.Claims, service service.PublishingService) ([]byte, int) {
	log.WithFields(log.Fields{}).Debug("handleSubmitDatasetProposal()")

	var err error
	var nodeId string
	var found bool

	// get ProposalNodeId from request query parameters
	queryParams := request.QueryStringParameters
	if nodeId, found = queryParams["node_id"]; !found {
		return nil, 400
	}

	userId := int(claims.UserClaim.Id)

	proposalDTO, err := service.SubmitDatasetProposal(userId, nodeId)
	if err != nil {
		return nil, 400
	}
	log.WithFields(log.Fields{"proposalDTO": fmt.Sprintf("%+v", proposalDTO)}).Debug("handleSubmitDatasetProposal() submitted proposal")

	jsonBody, err := json.Marshal(proposalDTO)
	if err != nil {
		log.Error("json.Marshal() failed: ", err)
		return nil, 500
	}

	return jsonBody, 200
}

func handleWithdrawDatasetProposal(request events.APIGatewayV2HTTPRequest, claims *authorizer.Claims, service service.PublishingService) ([]byte, int) {
	log.WithFields(log.Fields{}).Debug("handleWithdrawDatasetProposal()")

	var err error
	var nodeId string
	var found bool

	// get ProposalNodeId from request query parameters
	queryParams := request.QueryStringParameters
	if nodeId, found = queryParams["node_id"]; !found {
		return nil, 400
	}

	userId := int(claims.UserClaim.Id)

	proposalDTO, err := service.WithdrawDatasetProposal(userId, nodeId)
	if err != nil {
		return nil, 400
	}
	log.WithFields(log.Fields{"proposalDTO": fmt.Sprintf("%+v", proposalDTO)}).Debug("handleWithdrawDatasetProposal() withdrew proposal")

	jsonBody, err := json.Marshal(proposalDTO)
	if err != nil {
		log.Error("json.Marshal() failed: ", err)
		return nil, 500
	}

	return jsonBody, 200

}

func handleAcceptDatasetProposal(request events.APIGatewayV2HTTPRequest, claims *authorizer.Claims, service service.PublishingService) ([]byte, int) {
	log.WithFields(log.Fields{}).Debug("handleAcceptDatasetProposal()")

	var err error
	var nodeId string
	var found bool

	// get ProposalNodeId from request query parameters
	queryParams := request.QueryStringParameters
	if nodeId, found = queryParams["node_id"]; !found {
		return nil, 400
	}

	repositoryId := claims.OrgClaim.IntId
	log.WithFields(log.Fields{"repositoryId": repositoryId, "nodeId": nodeId}).Debug("handleAcceptDatasetProposal()")

	proposalDTO, err := service.AcceptDatasetProposal(int(repositoryId), nodeId)
	if err != nil {
		return nil, 400
	}
	log.WithFields(log.Fields{"proposalDTO": fmt.Sprintf("%+v", proposalDTO)}).Debug("handleAcceptDatasetProposal() accepted proposal")

	jsonBody, err := json.Marshal(proposalDTO)
	if err != nil {
		log.Error("json.Marshal() failed: ", err)
		return nil, 500
	}

	return jsonBody, 200
}

func handleRejectDatasetProposal(request events.APIGatewayV2HTTPRequest, claims *authorizer.Claims, service service.PublishingService) ([]byte, int) {
	log.WithFields(log.Fields{}).Debug("handleRejectDatasetProposal()")

	var err error
	var nodeId string
	var found bool

	// get ProposalNodeId from request query parameters
	queryParams := request.QueryStringParameters
	if nodeId, found = queryParams["node_id"]; !found {
		return nil, 400
	}

	repositoryId := claims.OrgClaim.IntId
	log.WithFields(log.Fields{"repositoryId": repositoryId, "nodeId": nodeId}).Debug("handleRejectDatasetProposal()")

	proposalDTO, err := service.RejectDatasetProposal(int(repositoryId), nodeId)
	if err != nil {
		return nil, 400
	}
	log.WithFields(log.Fields{"proposalDTO": fmt.Sprintf("%+v", proposalDTO)}).Debug("handleRejectDatasetProposal() rejected proposal")

	jsonBody, err := json.Marshal(proposalDTO)
	if err != nil {
		log.Error("json.Marshal() failed: ", err)
		return nil, 500
	}

	return jsonBody, 200
}

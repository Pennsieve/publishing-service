package handler

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"regexp"

	"github.com/aws/aws-lambda-go/events"
	"github.com/pennsieve/pennsieve-go-core/pkg/authorizer"
	"github.com/pennsieve/pennsieve-go-core/pkg/queries/pgdb"
	"github.com/pennsieve/publishing-service/api/dtos"
	"github.com/pennsieve/publishing-service/api/logging"
	"github.com/pennsieve/publishing-service/api/notification"
	"github.com/pennsieve/publishing-service/api/service"
	"github.com/pennsieve/publishing-service/api/store"
	"github.com/valyala/fastjson"
)

func init() {
	logging.SetDefaultFromEnv()
}

// PublishingServiceHandler takes the invocation context so that the Lambda
// invocation id the runtime puts there can be logged; lambda.Start supplies it.
func PublishingServiceHandler(ctx context.Context, request events.APIGatewayV2HTTPRequest) (*events.APIGatewayV2HTTPResponse, error) {
	// The request-scoped logger is built exactly once, here, and threaded
	// through every layer below. Nothing downstream reconstructs it or falls
	// back to slog.Default.
	logger := newRequestLogger(ctx, request)

	return handleRequest(logger, request)
}

func handleRequest(logger *slog.Logger, request events.APIGatewayV2HTTPRequest) (*events.APIGatewayV2HTTPResponse, error) {
	var err error
	var statusCode int
	var jsonBody []byte

	r := regexp.MustCompile(`(?P<method>) (?P<pathKey>.*)`)
	routeKeyParts := r.FindStringSubmatch(request.RouteKey)
	routeKey := routeKeyParts[r.SubexpIndex("pathKey")]
	httpMethod := request.RequestContext.HTTP.Method

	logger = logger.With(
		slog.String(logging.KeyMethod, httpMethod),
		slog.String(logging.KeyRoute, routeKey),
	)
	logger.Info("handling request")

	var serviceImpl service.PublishingService
	var claims *authorizer.Claims
	switch routeKey {
	case "/repositories":
		pubStore := store.NewPublishingStore(logger)
		serviceImpl = service.NewPublishingService(logger, pubStore, nil, nil)

	default:
		claims = authorizer.ParseClaims(request.RequestContext.Authorizer.Lambda)
		orgId := claims.OrgClaim.IntId
		logger = logger.With(
			slog.Int64(logging.KeyOrgID, orgId),
			slog.Int64(logging.KeyUserID, claims.UserClaim.Id),
		)

		db, err := pgdb.ConnectRDSWithOrg(int(orgId))
		if err != nil {
			logger.Error("unable to connect to RDS database", slog.Any(logging.KeyError, err))
			return &events.APIGatewayV2HTTPResponse{StatusCode: 500}, nil
		}
		logger.Info("connected to RDS database")
		defer db.Close()

		pubStore := store.NewPublishingStore(logger)
		pennsieve, err := store.NewPennsieveStore(logger, db, orgId)
		if err != nil {
			logger.Error("failed to create pennsieve store", slog.Any(logging.KeyError, err))
			return &events.APIGatewayV2HTTPResponse{StatusCode: 500}, nil
		}
		// Emails are sent via the Pennsieve email-service (enqueue -> consumer
		// renders + delivers), replacing the previous direct-SES EmailNotifier.
		notifier, err := notification.NewQueueNotifier(context.TODO(), logger)
		if err != nil {
			logger.Error("failed to create email notifier", slog.Any(logging.KeyError, err))
			return &events.APIGatewayV2HTTPResponse{StatusCode: 500}, nil
		}
		serviceImpl = service.NewPublishingService(logger, pubStore, pennsieve, notifier)
	}

	switch routeKey {
	case "/info":
		switch httpMethod {
		case "GET":
			jsonBody, statusCode = handleGetPublishingInfo(logger, serviceImpl)
		}
	case "/repositories":
		switch httpMethod {
		case "GET":
			jsonBody, statusCode = handleGetPublishingRepositories(logger, serviceImpl)
		}
	case "/questions":
		switch httpMethod {
		case "GET":
			jsonBody, statusCode = handleGetProposalQuestions(logger, serviceImpl)
		}
	case "/proposal":
		switch httpMethod {
		case "GET":
			if ok := authorizedAuthor(claims); ok {
				jsonBody, statusCode = handleGetUserDatasetProposals(logger, claims, serviceImpl)
			} else {
				jsonBody = nil
				statusCode = 401
			}
		case "POST":
			jsonBody, statusCode = handleCreateDatasetProposal(logger, request, claims, serviceImpl)
		case "PUT":
			jsonBody, statusCode = handleUpdateDatasetProposal(logger, request, claims, serviceImpl)
		case "DELETE":
			jsonBody, statusCode = handleDeleteDatasetProposal(logger, request, claims, serviceImpl)
		}
	case "/proposal/submit":
		switch httpMethod {
		case "POST":
			jsonBody, statusCode = handleSubmitDatasetProposal(logger, request, claims, serviceImpl)
		}
	case "/proposal/withdraw":
		switch httpMethod {
		case "POST":
			jsonBody, statusCode = handleWithdrawDatasetProposal(logger, request, claims, serviceImpl)
		}
	case "/submission":
		switch httpMethod {
		case "GET":
			jsonBody, statusCode = handleGetWorkspaceDatasetProposals(logger, authorizedPublisher, claims, serviceImpl, request)
		}
	case "/submission/accept":
		switch httpMethod {
		case "POST":
			jsonBody, statusCode = handleAcceptDatasetProposal(logger, authorizedPublisher, claims, serviceImpl, request)
		}
	case "/submission/reject":
		switch httpMethod {
		case "POST":
			jsonBody, statusCode = handleRejectDatasetProposal(logger, authorizedPublisher, claims, serviceImpl, request)
		}
	default:
		err = errors.New("unknown route")
		logger.Error("unknown route")
	}

	jsonString := string(jsonBody)

	response := events.APIGatewayV2HTTPResponse{
		Body:       jsonString,
		StatusCode: statusCode,
		Headers: map[string]string{
			"content-type": "application/json",
		},
	}
	logger.Info("request handled", slog.Int(logging.KeyStatus, statusCode))
	// Full response bodies are only ever emitted at DEBUG: this service carries
	// dataset-proposal content, so bodies must not land in logs by default.
	logger.Debug("response body", slog.String("body", jsonString))

	return &response, err
}

type Authorizer func(claims *authorizer.Claims) bool

// TODO: figure out author authorization
func authorizedAuthor(claims *authorizer.Claims) bool {
	return true
}
func authorizedPublisher(claims *authorizer.Claims) bool {
	return authorizer.IsPublisher(claims)
}

func handleGetPublishingInfo(logger *slog.Logger, service service.PublishingService) ([]byte, int) {
	result, err := service.GetPublishingInfo()
	if err != nil {
		// TODO: provide a better response than nil on a 500
		logger.Error("service.GetPublishingInfo() failed", slog.Any(logging.KeyError, err))
		return nil, 500
	}

	jsonBody, err := json.Marshal(result)
	if err != nil {
		// TODO: provide a better response than nil on a 500
		logger.Error("json.Marshal() failed marshalling publishing info", slog.Any(logging.KeyError, err))
		return nil, 500
	}

	return jsonBody, 200
}

func handleGetPublishingRepositories(logger *slog.Logger, service service.PublishingService) ([]byte, int) {
	result, err := service.GetPublishingRepositories()
	if err != nil {
		// TODO: provide a better response than nil on a 500
		logger.Error("service.GetPublishingRepositories() failed", slog.Any(logging.KeyError, err))
		return nil, 500
	}

	jsonBody, err := json.Marshal(result)
	if err != nil {
		// TODO: provide a better response than nil on a 500
		logger.Error("json.Marshal() failed marshalling repositories", slog.Any(logging.KeyError, err))
		return nil, 500
	}

	return jsonBody, 200
}

func handleGetProposalQuestions(logger *slog.Logger, service service.PublishingService) ([]byte, int) {
	result, err := service.GetProposalQuestions()
	if err != nil {
		// TODO: provide a better response than nil on a 500
		logger.Error("service.GetProposalQuestions() failed", slog.Any(logging.KeyError, err))
		return nil, 500
	}

	jsonBody, err := json.Marshal(result)
	if err != nil {
		// TODO: provide a better response than nil on a 500
		logger.Error("json.Marshal() failed marshalling questions", slog.Any(logging.KeyError, err))
		return nil, 500
	}

	return jsonBody, 200
}

func handleGetUserDatasetProposals(logger *slog.Logger, claims *authorizer.Claims, service service.PublishingService) ([]byte, int) {
	// get user id from User Claim
	userId := claims.UserClaim.Id

	result, err := service.GetDatasetProposalsForUser(userId)
	if err != nil {
		logger.Error("service.GetDatasetProposalsForUser() failed", slog.Any(logging.KeyError, err))
		return nil, 500
	}

	jsonBody, err := json.Marshal(result)
	if err != nil {
		logger.Error("json.Marshal() failed marshalling user proposals", slog.Any(logging.KeyError, err))
		return nil, 500
	}

	return jsonBody, 200
}

func handleGetWorkspaceDatasetProposals(logger *slog.Logger, authorized Authorizer, claims *authorizer.Claims, service service.PublishingService, request events.APIGatewayV2HTTPRequest) ([]byte, int) {
	if !authorized(claims) {
		logger.Warn("caller is not authorized to list workspace proposals")
		return nil, 401
	}

	// get workspace NodeId from Organization Claim
	orgNodeId := claims.OrgClaim.NodeId

	// get proposal status from request query parameters (default = 'SUBMITTED')
	var status string
	var found bool
	queryParams := request.QueryStringParameters
	if status, found = queryParams["status"]; !found {
		status = "SUBMITTED"
	}

	// TODO: only permit query where status is SUBMITTED, ACCEPTED or REJECTED; else return a 400?

	result, err := service.GetDatasetProposalsForWorkspace(orgNodeId, status)
	if err != nil {
		// TODO: provide a better response than nil on a 500
		logger.Error("service.GetDatasetProposalsForWorkspace() failed",
			slog.String(logging.KeyOrgNodeID, orgNodeId),
			slog.String(logging.KeyProposalStatus, status),
			slog.Any(logging.KeyError, err))
		return nil, 500
	}

	response := &dtos.DatasetSubmissionsDTO{
		TotalCount: len(result),
		Proposals:  result,
	}

	jsonBody, err := json.Marshal(response)
	if err != nil {
		// TODO: provide a better response than nil on a 500
		logger.Error("json.Marshal() failed marshalling workspace proposals", slog.Any(logging.KeyError, err))
		return nil, 500
	}

	return jsonBody, 200
}

func handleCreateDatasetProposal(logger *slog.Logger, request events.APIGatewayV2HTTPRequest, claims *authorizer.Claims, service service.PublishingService) ([]byte, int) {
	err := fastjson.Validate(request.Body)
	if err != nil {
		logger.Error("request body validation failed", slog.Any(logging.KeyError, err))
		return nil, 400
	}

	// Unmarshal JSON into Dataset Proposal DTO
	bytes := []byte(request.Body)
	var requestDTO dtos.DatasetProposalDTO
	if err := json.Unmarshal(bytes, &requestDTO); err != nil {
		logger.Error("json.Unmarshal() failed unmarshalling proposal", slog.Any(logging.KeyError, err))
		return nil, 400
	}

	resultDTO, err := service.CreateDatasetProposal(claims.UserClaim.Id, requestDTO)
	if err != nil {
		logger.Error("service.CreateDatasetProposal() failed", slog.Any(logging.KeyError, err))
		return nil, 500
	}
	logger.Info("created dataset proposal", slog.String(logging.KeyNodeID, resultDTO.NodeId))

	jsonBody, err := json.Marshal(resultDTO)
	if err != nil {
		logger.Error("json.Marshal() failed marshalling created proposal", slog.Any(logging.KeyError, err))
		// TODO: provide a better response than nil on a 500
		return nil, 500
	}

	return jsonBody, 201
}

func handleUpdateDatasetProposal(logger *slog.Logger, request events.APIGatewayV2HTTPRequest, claims *authorizer.Claims, service service.PublishingService) ([]byte, int) {
	var err error

	// validate JSON
	err = fastjson.Validate(request.Body)
	if err != nil {
		logger.Error("request body validation failed", slog.Any(logging.KeyError, err))
		return nil, 400
	}

	// Unmarshal JSON into Dataset Proposal DTO
	bytes := []byte(request.Body)
	var requestDTO dtos.DatasetProposalDTO
	if err := json.Unmarshal(bytes, &requestDTO); err != nil {
		logger.Error("json.Unmarshal() failed unmarshalling proposal", slog.Any(logging.KeyError, err))
		return nil, 400
	}

	// check that ProposalNodeId was provided
	if requestDTO.NodeId == "" {
		logger.Error("missing required field(s): ProposalNodeId")
		return nil, 400
	}

	// get Proposal by UserId and ProposalNodeId
	proposal, err := service.GetDatasetProposal(requestDTO.UserId, requestDTO.NodeId)
	if err != nil {
		logger.Error("dataset proposal does not exist",
			slog.Int(logging.KeyUserID, requestDTO.UserId),
			slog.String(logging.KeyNodeID, requestDTO.NodeId),
			slog.Any(logging.KeyError, err))
		return nil, 404
	}

	// if it exists, then invoke update
	resultDTO, err := service.UpdateDatasetProposal(claims.UserClaim.Id, proposal, requestDTO)
	if err != nil {
		logger.Error("service.UpdateDatasetProposal() failed",
			slog.String(logging.KeyNodeID, requestDTO.NodeId),
			slog.Any(logging.KeyError, err))
		return nil, 500
	}
	logger.Info("updated dataset proposal", slog.String(logging.KeyNodeID, resultDTO.NodeId))

	jsonBody, err := json.Marshal(resultDTO)
	if err != nil {
		logger.Error("json.Marshal() failed marshalling updated proposal", slog.Any(logging.KeyError, err))
		return nil, 500
	}

	return jsonBody, 200
}

func handleDeleteDatasetProposal(logger *slog.Logger, request events.APIGatewayV2HTTPRequest, claims *authorizer.Claims, service service.PublishingService) ([]byte, int) {
	var err error
	var nodeId string
	var found bool

	// get ProposalNodeId from request query parameters
	queryParams := request.QueryStringParameters
	if nodeId, found = queryParams["proposal_node_id"]; !found {
		logger.Error("missing required query parameter: proposal_node_id")
		return nil, 400
	}

	userId := int(claims.UserClaim.Id)

	proposal, err := service.GetDatasetProposal(userId, nodeId)
	if err != nil {
		// probably not found
		logger.Error("dataset proposal not found",
			slog.String(logging.KeyNodeID, nodeId),
			slog.Any(logging.KeyError, err))
		return nil, 404
	}

	_, err = service.DeleteDatasetProposal(proposal)
	if err != nil {
		logger.Error("service.DeleteDatasetProposal() failed",
			slog.String(logging.KeyNodeID, nodeId),
			slog.Any(logging.KeyError, err))
		return nil, 500
	}
	logger.Info("deleted dataset proposal", slog.String(logging.KeyNodeID, nodeId))

	return nil, 200
}

func handleSubmitDatasetProposal(logger *slog.Logger, request events.APIGatewayV2HTTPRequest, claims *authorizer.Claims, service service.PublishingService) ([]byte, int) {
	var err error
	var nodeId string
	var found bool

	// get ProposalNodeId from request query parameters
	queryParams := request.QueryStringParameters
	if nodeId, found = queryParams["node_id"]; !found {
		logger.Error("missing required query parameter: node_id")
		return nil, 400
	}

	userId := int(claims.UserClaim.Id)

	proposalDTO, err := service.SubmitDatasetProposal(userId, nodeId)
	if err != nil {
		logger.Error("service.SubmitDatasetProposal() failed",
			slog.String(logging.KeyNodeID, nodeId),
			slog.Any(logging.KeyError, err))
		return nil, 400
	}
	logger.Info("submitted dataset proposal", slog.String(logging.KeyNodeID, nodeId))

	jsonBody, err := json.Marshal(proposalDTO)
	if err != nil {
		logger.Error("json.Marshal() failed marshalling submitted proposal", slog.Any(logging.KeyError, err))
		return nil, 500
	}

	return jsonBody, 200
}

func handleWithdrawDatasetProposal(logger *slog.Logger, request events.APIGatewayV2HTTPRequest, claims *authorizer.Claims, service service.PublishingService) ([]byte, int) {
	var err error
	var nodeId string
	var found bool

	// get ProposalNodeId from request query parameters
	queryParams := request.QueryStringParameters
	if nodeId, found = queryParams["node_id"]; !found {
		logger.Error("missing required query parameter: node_id")
		return nil, 400
	}

	userId := int(claims.UserClaim.Id)

	proposalDTO, err := service.WithdrawDatasetProposal(userId, nodeId)
	if err != nil {
		logger.Error("service.WithdrawDatasetProposal() failed",
			slog.String(logging.KeyNodeID, nodeId),
			slog.Any(logging.KeyError, err))
		return nil, 400
	}
	logger.Info("withdrew dataset proposal", slog.String(logging.KeyNodeID, nodeId))

	jsonBody, err := json.Marshal(proposalDTO)
	if err != nil {
		logger.Error("json.Marshal() failed marshalling withdrawn proposal", slog.Any(logging.KeyError, err))
		return nil, 500
	}

	return jsonBody, 200

}

func handleAcceptDatasetProposal(logger *slog.Logger, authorized Authorizer, claims *authorizer.Claims, service service.PublishingService, request events.APIGatewayV2HTTPRequest) ([]byte, int) {
	if !authorized(claims) {
		logger.Warn("caller is not authorized to accept proposals")
		return nil, 401
	}

	var err error
	var nodeId string
	var found bool

	// get ProposalNodeId from request query parameters
	queryParams := request.QueryStringParameters
	if nodeId, found = queryParams["node_id"]; !found {
		logger.Error("missing required query parameter: node_id")
		return nil, 400
	}

	orgNodeId := claims.OrgClaim.NodeId

	proposalDTO, err := service.AcceptDatasetProposal(orgNodeId, nodeId)
	if err != nil {
		logger.Error("service.AcceptDatasetProposal() failed",
			slog.String(logging.KeyOrgNodeID, orgNodeId),
			slog.String(logging.KeyNodeID, nodeId),
			slog.Any(logging.KeyError, err))
		return nil, 400
	}
	logger.Info("accepted dataset proposal",
		slog.String(logging.KeyOrgNodeID, orgNodeId),
		slog.String(logging.KeyNodeID, nodeId))

	jsonBody, err := json.Marshal(proposalDTO)
	if err != nil {
		logger.Error("json.Marshal() failed marshalling accepted proposal", slog.Any(logging.KeyError, err))
		return nil, 500
	}

	return jsonBody, 200
}

func handleRejectDatasetProposal(logger *slog.Logger, authorized Authorizer, claims *authorizer.Claims, service service.PublishingService, request events.APIGatewayV2HTTPRequest) ([]byte, int) {
	if !authorized(claims) {
		logger.Warn("caller is not authorized to reject proposals")
		return nil, 401
	}

	var err error
	var nodeId string
	var found bool

	// get ProposalNodeId from request query parameters
	queryParams := request.QueryStringParameters
	if nodeId, found = queryParams["node_id"]; !found {
		logger.Error("missing required query parameter: node_id")
		return nil, 400
	}

	orgNodeId := claims.OrgClaim.NodeId

	proposalDTO, err := service.RejectDatasetProposal(orgNodeId, nodeId)
	if err != nil {
		logger.Error("service.RejectDatasetProposal() failed",
			slog.String(logging.KeyOrgNodeID, orgNodeId),
			slog.String(logging.KeyNodeID, nodeId),
			slog.Any(logging.KeyError, err))
		return nil, 400
	}
	logger.Info("rejected dataset proposal",
		slog.String(logging.KeyOrgNodeID, orgNodeId),
		slog.String(logging.KeyNodeID, nodeId))

	jsonBody, err := json.Marshal(proposalDTO)
	if err != nil {
		logger.Error("json.Marshal() failed marshalling rejected proposal", slog.Any(logging.KeyError, err))
		return nil, 500
	}

	return jsonBody, 200
}

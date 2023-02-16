package handler

import (
	"encoding/json"
	"github.com/aws/aws-lambda-go/events"
	"github.com/pennsieve/publishing-service/api/service"
	"github.com/pennsieve/publishing-service/api/store"
	log "github.com/sirupsen/logrus"
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
	var jsonBody []byte

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
	}

	jsonString := string(jsonBody)
	log.Println("handleRequest() jsonString: ", jsonString)

	response := events.APIGatewayV2HTTPResponse{Body: jsonString, StatusCode: 200}
	log.Println("handleRequest() response: ", response)

	return &response, err
}

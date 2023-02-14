package handler

import (
	"github.com/aws/aws-lambda-go/events"
	"github.com/pennsieve/publishing-service/api/service"
	log "github.com/sirupsen/logrus"
	"os"
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
	var apiResponse *events.APIGatewayV2HTTPResponse

	log.Println("PublishingServiceHandler() ")

	publishingService := service.NewPublishingService()
	repos, err := publishingService.GetPublishingRepositories()

	if err != nil {
		log.Fatalln("publishingService.GetPublishingRepositories() failed")
	}
	apiResponse, err = handleRequest(repos)

	return apiResponse, err
}

func handleRequest(repos string) (*events.APIGatewayV2HTTPResponse, error) {
	log.Println("handleRequest() repos: ", repos)
	apiResponse := events.APIGatewayV2HTTPResponse{Body: "{'response':'hello'}", StatusCode: 200}

	return &apiResponse, nil
}

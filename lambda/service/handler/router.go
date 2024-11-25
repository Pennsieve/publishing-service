package handler

import (
	"context"
	"errors"
	"github.com/aws/aws-lambda-go/events"
	"github.com/pennsieve/publishing-service/service/container"
	"log"
	"net/http"
	"regexp"
)

var ErrUnsupportedRoute = errors.New("unsupported route")
var ErrUnsupportedPath = errors.New("unsupported path")

type RouterHandlerFunc func(context.Context, events.APIGatewayV2HTTPRequest, *container.Container) (events.APIGatewayV2HTTPResponse, error)

// Router defines the router interface
type Router interface {
	POST(string, RouterHandlerFunc)
	GET(string, RouterHandlerFunc)
	DELETE(string, RouterHandlerFunc)
	PUT(string, RouterHandlerFunc)
	Start(context.Context, events.APIGatewayV2HTTPRequest, *container.Container) (events.APIGatewayV2HTTPResponse, error)
}

type LambdaRouter struct {
	getRoutes    map[string]RouterHandlerFunc
	postRoutes   map[string]RouterHandlerFunc
	deleteRoutes map[string]RouterHandlerFunc
	putRoutes    map[string]RouterHandlerFunc
}

func NewLambdaRouter() Router {
	return &LambdaRouter{
		make(map[string]RouterHandlerFunc),
		make(map[string]RouterHandlerFunc),
		make(map[string]RouterHandlerFunc),
		make(map[string]RouterHandlerFunc),
	}
}

func (r *LambdaRouter) POST(routeKey string, handler RouterHandlerFunc) {
	r.postRoutes[routeKey] = handler
}

func (r *LambdaRouter) GET(routeKey string, handler RouterHandlerFunc) {
	r.getRoutes[routeKey] = handler
}

func (r *LambdaRouter) DELETE(routeKey string, handler RouterHandlerFunc) {
	r.deleteRoutes[routeKey] = handler
}

func (r *LambdaRouter) PUT(routeKey string, handler RouterHandlerFunc) {
	r.putRoutes[routeKey] = handler
}

func extractRoute(requestRouteKey string) string {
	r := regexp.MustCompile(`(?P<method>) (?P<pathKey>.*)`)
	routeKeyParts := r.FindStringSubmatch(requestRouteKey)
	return routeKeyParts[r.SubexpIndex("pathKey")]
}

func (r *LambdaRouter) Start(ctx context.Context, request events.APIGatewayV2HTTPRequest, container *container.Container) (events.APIGatewayV2HTTPResponse, error) {
	log.Println(request)
	routeKey := extractRoute(request.RouteKey)

	switch request.RequestContext.HTTP.Method {
	case http.MethodPost:
		f, ok := r.postRoutes[routeKey]
		if ok {
			return f(ctx, request, container)
		} else {
			return handleError()
		}
	case http.MethodGet:
		f, ok := r.getRoutes[routeKey]
		if ok {
			return f(ctx, request, container)
		} else {
			return handleError()
		}
	case http.MethodDelete:
		f, ok := r.deleteRoutes[routeKey]
		if ok {
			return f(ctx, request, container)
		} else {
			return handleError()
		}
	case http.MethodPut:
		f, ok := r.putRoutes[routeKey]
		if ok {
			return f(ctx, request, container)
		} else {
			return handleError()
		}
	default:
		log.Println(ErrUnsupportedPath.Error())
		return events.APIGatewayV2HTTPResponse{
			StatusCode: http.StatusUnprocessableEntity,
			Body:       ErrUnsupportedPath.Error(),
		}, nil
	}
}

func handleError() (events.APIGatewayV2HTTPResponse, error) {
	log.Println(ErrUnsupportedRoute.Error())
	return events.APIGatewayV2HTTPResponse{
		StatusCode: http.StatusNotFound,
		Body:       ErrUnsupportedRoute.Error(),
	}, nil
}

package cmd

import (
	"net/http"

	"github.com/gorilla/mux"
)

var globalHandlers = []HandlerFunc{
	setBrowserRedirectHandler,
}

func configureServerHandler() (http.Handler, error) {
	router := mux.NewRouter().SkipClean(true).UseEncodedPath()

	if globalBrowserEnabled {
		if err := registerWebRouter(router); err != nil {
			return nil, err
		}
	}

	registerAPIRouter(router, true, false)

	router.NotFoundHandler = http.HandlerFunc(httpTraceAll(errorResponseHandler))
	router.MethodNotAllowedHandler = http.HandlerFunc(httpTraceAll(errorResponseHandler))

	return registerHandlers(router, globalHandlers...), nil
}

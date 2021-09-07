package cmd

import (
	"github.com/storeros/ipos/cmd/ipos/logger"
	"net/http"
	"strings"
)

type HandlerFunc func(http.Handler) http.Handler

func registerHandlers(h http.Handler, handlerFns ...HandlerFunc) http.Handler {
	for _, hFn := range handlerFns {
		h = hFn(h)
	}
	return h
}

const ReservedMetadataPrefix = "X-IPOS-Internal-"

const (
	iposReservedBucket     = "ipos"
	iposReservedBucketPath = SlashSeparator + iposReservedBucket
)

type redirectHandler struct {
	handler http.Handler
}

func setBrowserRedirectHandler(h http.Handler) http.Handler {
	return redirectHandler{handler: h}
}

func getRedirectLocation(urlPath string) (rLocation string) {
	if urlPath == iposReservedBucketPath {
		rLocation = iposReservedBucketPath + SlashSeparator
	}
	if contains([]string{
		SlashSeparator,
		"/webrpc",
		"/login",
		"/favicon-16x16.png",
		"/favicon-32x32.png",
		"/favicon-96x96.png",
	}, urlPath) {
		rLocation = iposReservedBucketPath + urlPath
	}
	return rLocation
}

func guessIsBrowserReq(req *http.Request) bool {
	if req == nil {
		return false
	}
	aType := getRequestAuthType(req)
	return strings.Contains(req.Header.Get("User-Agent"), "Mozilla") && globalBrowserEnabled &&
		(aType == authTypeJWT || aType == authTypeAnonymous)
}

func (h redirectHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if guessIsBrowserReq(r) {
		redirectLocation := getRedirectLocation(r.URL.Path)
		if redirectLocation != "" {
			http.Redirect(w, r, redirectLocation, http.StatusTemporaryRedirect)
			return
		}
	}
	h.handler.ServeHTTP(w, r)
}

type criticalErrorHandler struct{ handler http.Handler }

func (h criticalErrorHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	defer func() {
		if err := recover(); err == logger.ErrCritical {
			writeErrorResponse(r.Context(), w, errorCodes.ToAPIErr(ErrInternalError), r.URL, guessIsBrowserReq(r))
		} else if err != nil {
			panic(err)
		}
	}()
	h.handler.ServeHTTP(w, r)
}

package cmd

import (
	"net/http"

	xhttp "github.com/storeros/ipos/cmd/ipos/http"
	"github.com/gorilla/mux"
)

func newHTTPServerFn() *xhttp.Server {
	globalObjLayerMutex.Lock()
	defer globalObjLayerMutex.Unlock()
	return globalHTTPServer
}

func newObjectLayerWithoutSafeModeFn() ObjectLayer {
	globalObjLayerMutex.Lock()
	defer globalObjLayerMutex.Unlock()
	return globalObjectAPI
}

func newObjectLayerFn() ObjectLayer {
	globalObjLayerMutex.Lock()
	defer globalObjLayerMutex.Unlock()
	return globalObjectAPI
}

type objectAPIHandlers struct {
	ObjectAPI         func() ObjectLayer
	EncryptionEnabled func() bool
	AllowSSEKMS       func() bool
}

func registerAPIRouter(router *mux.Router, encryptionEnabled, allowSSEKMS bool) {
	api := objectAPIHandlers{
		ObjectAPI: newObjectLayerFn,
		EncryptionEnabled: func() bool {
			return encryptionEnabled
		},
		AllowSSEKMS: func() bool {
			return allowSSEKMS
		},
	}

	apiRouter := router.PathPrefix(SlashSeparator).Subrouter()
	var routers []*mux.Router
	routers = append(routers, apiRouter.PathPrefix("/{bucket}").Subrouter())

	for _, bucket := range routers {
		bucket.Methods(http.MethodGet).Path("/{object:.+}").HandlerFunc(
			maxClients(collectAPIStats("getobject", httpTraceHdrs(api.GetObjectHandler))))

		bucket.Methods(http.MethodPut).Path("/{object:.+}").HandlerFunc(
			maxClients(collectAPIStats("putobject", httpTraceHdrs(api.PutObjectHandler))))
		bucket.Methods(http.MethodDelete).Path("/{object:.+}").HandlerFunc(
			maxClients(collectAPIStats("deleteobject", httpTraceAll(api.DeleteObjectHandler))))

		bucket.Methods(http.MethodGet).HandlerFunc(
			maxClients(collectAPIStats("getbucketlocation", httpTraceAll(api.GetBucketLocationHandler)))).Queries("location", "")

		bucket.Methods(http.MethodGet).HandlerFunc(
			maxClients(collectAPIStats("listobjectsv2", httpTraceAll(api.ListObjectsV2Handler)))).Queries("list-type", "2")
		bucket.Methods(http.MethodGet).HandlerFunc(
			maxClients(collectAPIStats("listobjectsv1", httpTraceAll(api.ListObjectsV1Handler))))

		bucket.Methods(http.MethodPut).HandlerFunc(
			maxClients(collectAPIStats("putbucket", httpTraceAll(api.PutBucketHandler))))
		bucket.Methods(http.MethodHead).HandlerFunc(
			maxClients(collectAPIStats("headbucket", httpTraceAll(api.HeadBucketHandler))))
		bucket.Methods(http.MethodPost).HandlerFunc(
			maxClients(collectAPIStats("deletemultipleobjects", httpTraceAll(api.DeleteMultipleObjectsHandler)))).Queries("delete", "")
		bucket.Methods(http.MethodDelete).HandlerFunc(
			maxClients(collectAPIStats("deletebucket", httpTraceAll(api.DeleteBucketHandler))))
	}

	apiRouter.Methods(http.MethodGet).Path(SlashSeparator).HandlerFunc(
		maxClients(collectAPIStats("listbuckets", httpTraceAll(api.ListBucketsHandler))))

	apiRouter.NotFoundHandler = http.HandlerFunc(collectAPIStats("notfound", httpTraceAll(errorResponseHandler)))
	apiRouter.MethodNotAllowedHandler = http.HandlerFunc(collectAPIStats("methodnotallowed", httpTraceAll(errorResponseHandler)))
}

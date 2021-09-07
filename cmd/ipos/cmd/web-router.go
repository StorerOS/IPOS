package cmd

import (
	"fmt"
	"net/http"

	"github.com/storeros/ipos/browser"
	assetfs "github.com/elazarl/go-bindata-assetfs"
	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
	jsonrpc "github.com/gorilla/rpc/v2"
	"github.com/gorilla/rpc/v2/json2"
)

type webAPIHandlers struct {
	ObjectAPI func() ObjectLayer
}

type indexHandler struct {
	handler http.Handler
}

func (h indexHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	r.URL.Path = iposReservedBucketPath + SlashSeparator
	h.handler.ServeHTTP(w, r)
}

const assetPrefix = "production"

func assetFS() *assetfs.AssetFS {
	return &assetfs.AssetFS{
		Asset:     browser.Asset,
		AssetDir:  browser.AssetDir,
		AssetInfo: browser.AssetInfo,
		Prefix:    assetPrefix,
	}
}

const specialAssets = "index_bundle.*.js|loader.css|logo.svg|firefox.png|safari.png|chrome.png|favicon-16x16.png|favicon-32x32.png|favicon-96x96.png"

func registerWebRouter(router *mux.Router) error {
	web := &webAPIHandlers{
		ObjectAPI: newObjectLayerFn,
	}

	codec := json2.NewCodec()

	webBrowserRouter := router.PathPrefix(iposReservedBucketPath).HeadersRegexp("User-Agent", ".*Mozilla.*").Subrouter()

	webRPC := jsonrpc.NewServer()
	webRPC.RegisterCodec(codec, "application/json")
	webRPC.RegisterCodec(codec, "application/json; charset=UTF-8")

	if err := webRPC.RegisterService(web, "Web"); err != nil {
		return err
	}

	webBrowserRouter.Methods("POST").Path("/webrpc").Handler(webRPC)
	webBrowserRouter.Methods("PUT").Path("/upload/{bucket}/{object:.+}").HandlerFunc(httpTraceHdrs(web.Upload))

	webBrowserRouter.Methods("GET").Path("/download/{bucket}/{object:.+}").Queries("token", "{token:.*}").HandlerFunc(httpTraceHdrs(web.Download))
	webBrowserRouter.Methods("POST").Path("/zip").Queries("token", "{token:.*}").HandlerFunc(httpTraceHdrs(web.DownloadZip))

	compressAssets := handlers.CompressHandler(http.StripPrefix(iposReservedBucketPath, http.FileServer(assetFS())))

	webBrowserRouter.Path(fmt.Sprintf("/{assets:%s}", specialAssets)).Handler(compressAssets)

	webBrowserRouter.Path("/{index:.*}").Handler(indexHandler{compressAssets})

	return nil
}

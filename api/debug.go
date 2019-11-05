package api

import (
	"net/http"
	"net/http/pprof"

	"github.com/brunotm/replicant/server"
)

// AddDebugRoutes add routes for serving runtime profiling data
// Add all handlers from net/http/pprof and all profiles from pprof.Profiles().
func AddDebugRoutes(server *server.Server) {
	server.AddHandler(http.MethodGet, "/debug/pprof", wrapHTTPHandlerFunc(pprof.Index))
	server.AddHandler(http.MethodGet, "/debug/pprof/cmdline", wrapHTTPHandlerFunc(pprof.Cmdline))
	server.AddHandler(http.MethodGet, "/debug/pprof/profile", wrapHTTPHandlerFunc(pprof.Profile))
	server.AddHandler(http.MethodGet, "/debug/pprof/symbol", wrapHTTPHandlerFunc(pprof.Symbol))
	server.AddHandler(http.MethodGet, "/debug/pprof/trace", wrapHTTPHandlerFunc(pprof.Trace))
	server.AddHandler(http.MethodGet, "/debug/pprof/heap", wrapHTTPHandler(pprof.Handler("heap")))
	server.AddHandler(http.MethodGet, "/debug/pprof/block", wrapHTTPHandler(pprof.Handler("block")))
	server.AddHandler(http.MethodGet, "/debug/pprof/mutex", wrapHTTPHandler(pprof.Handler("mutex")))
	server.AddHandler(http.MethodGet, "/debug/pprof/allocs", wrapHTTPHandler(pprof.Handler("allocs")))
	server.AddHandler(http.MethodGet, "/debug/pprof/goroutine", wrapHTTPHandler(pprof.Handler("goroutine")))
	server.AddHandler(http.MethodGet, "/debug/pprof/threadcreate", wrapHTTPHandler(pprof.Handler("threadcreate")))
}

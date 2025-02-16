package diagnostics

import (
	"context"
	"fmt"
	"net/http"
	"net/http/pprof"
	"time"

	"github.com/prometheus/client_golang/prometheus/promhttp"
)

const writeTimeout = 10 * time.Second
const readTimeout = 5 * time.Second

type Server struct {
	httpServer *http.Server
}

func NewServer(port int) *Server {
	mux := http.NewServeMux()

	mux.Handle("/metrics", promhttp.Handler())
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	mux.HandleFunc("/debug/pprof/", pprofIndex)
	mux.HandleFunc("/debug/pprof/cmdline", pprofCmdLine)
	mux.HandleFunc("/debug/pprof/profile", pprofProfile)
	mux.HandleFunc("/debug/pprof/symbol", pprofSymbol)
	mux.HandleFunc("/debug/pprof/trace", pprofTrace)
	mux.HandleFunc("/debug/pprof/allocs", pprofAllocs)
	mux.HandleFunc("/debug/pprof/block", pprofBlock)
	mux.HandleFunc("/debug/pprof/goroutine", pprofGoroutine)
	mux.HandleFunc("/debug/pprof/heap", pprofHeap)
	mux.HandleFunc("/debug/pprof/mutex", pprofMutex)
	mux.HandleFunc("/debug/pprof/threadcreate", pprofThreadCreate)

	return &Server{
		httpServer: &http.Server{
			Addr:         fmt.Sprintf(":%d", port),
			Handler:      mux,
			ReadTimeout:  readTimeout,
			WriteTimeout: writeTimeout,
		},
	}
}

func (s *Server) Start() error {
	return s.httpServer.ListenAndServe()
}

func (s *Server) Shutdown(ctx context.Context) error {
	return s.httpServer.Shutdown(ctx)
}

func pprofCmdLine(w http.ResponseWriter, r *http.Request) {
	pprof.Cmdline(w, r)
}

func pprofProfile(w http.ResponseWriter, r *http.Request) {
	pprof.Profile(w, r)
}

func pprofSymbol(w http.ResponseWriter, r *http.Request) {
	pprof.Symbol(w, r)
}

func pprofTrace(w http.ResponseWriter, r *http.Request) {
	pprof.Trace(w, r)
}

func pprofAllocs(w http.ResponseWriter, r *http.Request) {
	pprof.Handler("allocs").ServeHTTP(w, r)
}

func pprofBlock(w http.ResponseWriter, r *http.Request) {
	pprof.Handler("block").ServeHTTP(w, r)
}

func pprofGoroutine(w http.ResponseWriter, r *http.Request) {
	pprof.Handler("goroutine").ServeHTTP(w, r)
}

func pprofHeap(w http.ResponseWriter, r *http.Request) {
	pprof.Handler("heap").ServeHTTP(w, r)
}

func pprofMutex(w http.ResponseWriter, r *http.Request) {
	pprof.Handler("mutex").ServeHTTP(w, r)
}

func pprofThreadCreate(w http.ResponseWriter, r *http.Request) {
	pprof.Handler("threadcreate").ServeHTTP(w, r)
}

func pprofIndex(w http.ResponseWriter, r *http.Request) {
	pprof.Index(w, r)
}

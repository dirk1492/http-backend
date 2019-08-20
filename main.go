package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net/http"
	"net/http/httputil"
	"os"
	"os/signal"
	"runtime"
	"strings"
	"syscall"
	"time"
)

// Version
var Version string

// Tag
var Tag string

var debug *bool
var copyHeader *bool

func main() {
	port := flag.Int("port", 8080, "Port number")
	status := flag.Int("status", 200, "HTTP status")
	timeout := flag.Duration("timeout", 5*time.Second, "Shutdown timeout in seconds")
	debug = flag.Bool("debug", false, "Log all traffic")

	copyHeader = flag.Bool("copy-auth-header", false, "Copy authentication header entries")

	if Version == "" {
		Version = "development"
	}

	fmt.Printf("Go version: %s\n", runtime.Version())
	fmt.Printf("Program version: %s\n", Tag)
	fmt.Printf("Git version: %s\n\n", Version)

	flag.Parse()

	notFound := newHTTPServer(fmt.Sprintf(":%d", *port), handle(*status))

	// start the main http server
	go func() {
		fmt.Fprintf(os.Stdout, "Start http server on port %v\n", *port)
		err := notFound.ListenAndServe()
		if err != http.ErrServerClosed {
			fmt.Fprintf(os.Stderr, "failed to start server: %s\n", err)
			os.Exit(1)
		}
	}()

	waitShutdown(notFound, *timeout)
}

type server struct {
	mux *http.ServeMux
}

func (s *server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.mux.ServeHTTP(w, r)
}

func newHTTPServer(addr string, handler http.Handler) *http.Server {
	return &http.Server{
		Addr:              addr,
		Handler:           handler,
		ReadTimeout:       10 * time.Second,
		ReadHeaderTimeout: 10 * time.Second,
		WriteTimeout:      10 * time.Second,
		IdleTimeout:       10 * time.Second,
	}
}

func handle(status int) *server {
	s := &server{mux: http.NewServeMux()}
	s.mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {

		if *copyHeader {
			for k, v := range r.Header {
				if strings.HasPrefix(k, "X-Auth-Subject") || k == "Authorization" {
					for _, s := range v {
						w.Header().Add(k, s)
					}
				}
			}
		}

		w.WriteHeader(status)
		fmt.Fprint(w, http.StatusText(status))

		if *debug {
			resDump, err := httputil.DumpRequest(r, true)
			if err != nil {
				log.Println(err)
			}
			log.Println(string(resDump))
		}
	})
	return s
}

func waitShutdown(s *http.Server, timeout time.Duration) {
	sigchan := make(chan os.Signal, 1)
	signal.Notify(sigchan, os.Interrupt, syscall.SIGTERM)
	<-sigchan

	fmt.Fprintf(os.Stdout, "Stopping server...\n")
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	if err := s.Shutdown(ctx); err != nil {
		fmt.Fprintf(os.Stderr, "Couldn't gracefully shutdown server: %s\n", err)
	}
}

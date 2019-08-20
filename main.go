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

// Version - the git version
var Version string

// Tag - the git tag
var Tag string

var debug *bool
var copyHeader *bool
var checkAuthSubject *bool

var l = log.New(os.Stdout, "", log.Ldate|log.Ltime|log.Lmicroseconds)

func main() {
	port := flag.Int("port", 8080, "Port number")
	status := flag.Int("status", 200, "HTTP status")
	timeout := flag.Duration("timeout", 5*time.Second, "Shutdown timeout in seconds")
	debug = flag.Bool("debug", false, "Log all traffic")

	copyHeader = flag.Bool("copy-auth-header", false, "Copy authentication header entries")
	checkAuthSubject = flag.Bool("check-auth-subject", true, "Send 403 if X-Auth-Subject is missing")

	if Version == "" {
		Version = "development"
	}

	if Tag == "" {
		Tag = "development"
	}

	l.Printf("Go version: %s\n", runtime.Version())
	l.Printf("Program version: %s\n", Tag)
	l.Printf("Git version: %s\n", Version)

	flag.Parse()

	notFound := newHTTPServer(fmt.Sprintf(":%d", *port), handle(*status))

	// start the main http server
	go func() {
		l.Printf("Start http server on port %v\n", *port)
		err := notFound.ListenAndServe()
		if err != http.ErrServerClosed {
			l.Printf("Failed to start server: %s\n", err)
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

		if *checkAuthSubject {
			_, ok := r.Header["X-Auth-Subject"]
			if !ok {
				w.WriteHeader(http.StatusForbidden)
				fmt.Fprint(w, http.StatusText(http.StatusForbidden))
				return
			}
		}

		if *copyHeader {
			for k, v := range r.Header {
				if strings.HasPrefix(k, "X-Auth-") || k == "Authorization" {
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
				l.Println(err)
			}
			l.Println(string(resDump))
		}
	})
	return s
}

func waitShutdown(s *http.Server, timeout time.Duration) {
	sigchan := make(chan os.Signal, 1)
	signal.Notify(sigchan, os.Interrupt, syscall.SIGTERM)
	<-sigchan

	l.Printf("Stopping server...\n")
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	if err := s.Shutdown(ctx); err != nil {
		l.Printf("Couldn't gracefully shutdown server: %s\n", err)
	}
}

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
	"strings"
	"syscall"
	"time"
)

var headers []string
var debug *bool

func main() {
	port := flag.Int("port", 8080, "Port number")
	status := flag.Int("status", 200, "HTTP status")
	timeout := flag.Duration("timeout", 5*time.Second, "Shutdown timeout in seconds")
	debug = flag.Bool("debug", false, "Log all traffic")

	headerParam := flag.String("copy-header", "", "Comma-seperated list of header keys")

	flag.Parse()

	headers = getHeaderList(*headerParam)

	notFound := newHTTPServer(fmt.Sprintf(":%d", *port), handle(*status))

	// start the main http server
	go func() {
		fmt.Fprintf(os.Stdout, "start http server on port %v\n", *port)
		err := notFound.ListenAndServe()
		if err != http.ErrServerClosed {
			fmt.Fprintf(os.Stderr, "failed to start server: %s\n", err)
			os.Exit(1)
		}
	}()

	waitShutdown(notFound, *timeout)
}

func getHeaderList(txt string) []string {
	tmp := strings.Split(txt, ",")
	rc := make([]string, len(tmp))
	for i, k := range tmp {
		rc[i] = strings.Trim(k, " \t\n\r")
	}
	return rc
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

		for _, k := range headers {
			v := r.Header.Get(k)
			if v != "" {
				w.Header().Add(k, v)
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

	fmt.Fprintf(os.Stdout, "stopping server...\n")
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	if err := s.Shutdown(ctx); err != nil {
		fmt.Fprintf(os.Stderr, "couldn't gracefully shutdown server: %s\n", err)
	}
}

package service

import (
	"fmt"
	"net/http"
	"net/http/httputil"
	"os"

	"github.com/nitrictech/suga/runtime/service"
)

type customService struct{}

func (a *customService) Start(proxy service.Proxy) error {
	p := &httputil.ReverseProxy{
		Director: func(req *http.Request) {
			req.URL.Host = proxy.Host()
			req.URL.Scheme = "http"
		},
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/", p.ServeHTTP)

	port := os.Getenv("PORT")
	if port == "" {
		return fmt.Errorf("PORT environment variable not set")
	}

	fmt.Printf("Starting Service proxy on port %s\n", port)
	return http.ListenAndServe(fmt.Sprintf(":%s", port), mux)
}

func Plugin() (service.Service, error) {
	return &customService{}, nil
}

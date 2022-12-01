package main

import (
	"fmt"
	"net"
	"net/http"

	"github.com/krajorama/weaveworks-common-testserver/server"
	opentracing "github.com/opentracing/opentracing-go"
	jaegercfg "github.com/uber/jaeger-client-go/config"
	"github.com/weaveworks/common/httpgrpc"
	"github.com/weaveworks/common/middleware"
	"github.com/weaveworks/common/user"
	"google.golang.org/grpc"
)

type testServer struct {
	*server.Server
	URL        string
	grpcServer *grpc.Server
}

func newTestServer(handler http.Handler) (*testServer, error) {
	lis, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return nil, err
	}

	server := &testServer{
		Server:     server.NewServer(handler),
		grpcServer: grpc.NewServer(),
		URL:        "direct://" + lis.Addr().String(),
	}

	httpgrpc.RegisterHTTPServer(server.grpcServer, server.Server)
	go server.grpcServer.Serve(lis)

	return server, nil
}

func main() {
    fmt.Println("Starting server")

	done := make(chan struct{})

	jaeger := jaegercfg.Configuration{}
	closer, err := jaeger.InitGlobalTracer("test")
	if err != nil {
		panic("Could not init tracing")
	}
	defer closer.Close()

	server, err := newTestServer(middleware.Tracer{}.Wrap(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodDelete:
			done <- struct{}{}
			return
		default:
		}
		switch r.URL.Path {
		case "/hello":
			fmt.Fprint(w, "world")
			return
		case "/error500":
			http.Error(w, "server error message", http.StatusInternalServerError)
		case "/error400":
			http.Error(w, "request error message", http.StatusForbidden)
		case "/trace":
			// This depends on having the http.HandlerFunc being wrapped in the tracer middleware
			span := opentracing.SpanFromContext(r.Context())
			if span != nil {
				fmt.Fprint(w, span.BaggageItem("name"))
			} else {
				http.Error(w, "could not load span from context", http.StatusBadRequest)
			}
		case "/orgid":
			id, _, err := user.ExtractOrgIDFromHTTPRequest(r)
			if err == nil {
				fmt.Fprint(w, id)
			} else {
				http.Error(w, "could not extract orgid", http.StatusInternalServerError)
			}
		default:
			http.Error(w, "Unrecognized test path", http.StatusBadRequest)
		}
	}),))

	if err != nil {
		panic(fmt.Sprintf("Could not start server %e", err))
	}
	defer server.grpcServer.GracefulStop()

	fmt.Println("Server listening on ", server.URL)
	<- done
}

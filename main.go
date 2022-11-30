package main

import (
	"fmt"
	"net"
	"net/http"

	"github.com/krajorama/weaveworks-common-testserver/server"
	"github.com/weaveworks/common/httpgrpc"
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

	server, err := newTestServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodDelete:
			done <- struct{}{}
		default:
		}
		fmt.Fprint(w, "world")
	}))

	if err != nil {
		panic(fmt.Sprintf("Could not start server %e", err))
	}
	defer server.grpcServer.GracefulStop()

	fmt.Println("Server listening on ", server.URL)
	<- done
}

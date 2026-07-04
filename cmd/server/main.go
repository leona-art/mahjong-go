package main

import (
	"context"
	"fmt"
	"log"
	"net/http"

	"connectrpc.com/connect"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"

	greetv1 "github.com/leona-art/mahjong-go/gen/greet/v1"
	"github.com/leona-art/mahjong-go/gen/greet/v1/greetv1connect"
)

type greetServer struct {
	greetv1connect.UnimplementedGreetServiceHandler
}

func (s *greetServer) Greet(
	ctx context.Context,
	req *connect.Request[greetv1.GreetRequest],
) (*connect.Response[greetv1.GreetResponse], error) {
	res := connect.NewResponse(&greetv1.GreetResponse{
		Greeting: fmt.Sprintf("Hello, %s!", req.Msg.Name),
	})
	return res, nil
}

func main() {
	mux := http.NewServeMux()
	path, handler := greetv1connect.NewGreetServiceHandler(&greetServer{})
	mux.Handle(path, handler)

	addr := "localhost:8080"
	log.Printf("listening on %s", addr)
	err := http.ListenAndServe(
		addr,
		h2c.NewHandler(mux, &http2.Server{}),
	)
	log.Fatal(err)
}

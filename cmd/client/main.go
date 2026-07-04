package main

import (
	"context"
	"log"
	"net/http"

	"connectrpc.com/connect"

	greetv1 "github.com/leona-art/mahjong-go/gen/greet/v1"
	"github.com/leona-art/mahjong-go/gen/greet/v1/greetv1connect"
)

func main() {
	client := greetv1connect.NewGreetServiceClient(
		http.DefaultClient,
		"http://localhost:8080",
	)
	res, err := client.Greet(
		context.Background(),
		connect.NewRequest(&greetv1.GreetRequest{Name: "Mahjong"}),
	)
	if err != nil {
		log.Fatal(err)
	}
	log.Println(res.Msg.Greeting)
}

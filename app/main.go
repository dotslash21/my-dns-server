package main

import (
	"log"
	"my-dns-server/app/server"
)

func main() {
	srv, err := server.NewServer(server.DefaultPort)
	if err != nil {
		log.Fatalf("Failed to create server: %v", err)
	}

	if err := srv.Run(); err != nil {
		log.Fatalf("Failed to run server: %v", err)
	}
}

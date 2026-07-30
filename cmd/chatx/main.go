package main

import (
	"chatflow/internal/server"
	"log"
)

func main() {

	srv := server.NewServer()
	r := srv.GetRouter()
	if err := r.Run("localhost:8082"); err != nil {
		log.Fatal(err)
	}
}

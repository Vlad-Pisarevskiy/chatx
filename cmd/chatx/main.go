package main

import (
	"chatflow/config"
	"chatflow/internal/server"
	"fmt"
	"log"
)

func main() {

	config.InitConfig()

	srv := server.NewServer()
	r := srv.GetRouter()
	if err := r.Run(fmt.Sprintf("%s:%s", config.Host(), config.Port())); err != nil {
		log.Fatal(err)
	}
}

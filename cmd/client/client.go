package main

import (
	"chatflow/internal/client"
	"log"
)

func main() {
	clt := client.NewClient()
	if err := clt.StartChat(); err != nil {
		log.Fatal(err)
	}
}

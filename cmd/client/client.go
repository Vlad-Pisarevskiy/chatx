package main

import (
	"chatflow/internal/client"
)

func main() {

	clt := client.NewClient()
	clt.StartChat()
}

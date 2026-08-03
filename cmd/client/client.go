package main

import (
	"chatflow/config"
	"chatflow/internal/client"
)

func main() {

	config.InitConfig()

	addr := config.Host() + ":" + config.Port()
	clt := client.NewClient()

	clt.StartChat(addr)
}

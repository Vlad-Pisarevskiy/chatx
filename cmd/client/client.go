package main

import (
	"chatflow/client"
	"chatflow/config"
)

func main() {

	config.InitConfig()

	addr := config.Host() + ":" + config.Port()
	clt := client.NewClient()

	clt.StartChat(addr)
}

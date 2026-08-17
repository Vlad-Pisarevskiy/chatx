package test

import (
	"chatflow/config"
	"chatflow/internal/repository/memory"
	server2 "chatflow/internal/server"
	service2 "chatflow/internal/service"
	"fmt"
	"testing"
)

func TestIntegration(t *testing.T) {

	repository := memory.NewRepository()
	service := service2.NewService(repository)
	server := server2.NewServer(service)

	router := server.GetRouter()
	_ = router.Run(fmt.Sprintf("%s:%s", config.Host(), config.Port()))
}

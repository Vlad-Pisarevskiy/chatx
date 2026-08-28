package main

import (
	"chatflow/config"
	hub2 "chatflow/internal/hub"
	"chatflow/internal/repository/postgres"
	server2 "chatflow/internal/server"
	service2 "chatflow/internal/service"
	"context"
	"fmt"
	"log"

	"github.com/jackc/pgx/v5/pgxpool"
)

func main() {

	config.InitConfig()
	pool, err := pgxpool.New(context.Background(), config.DBPath())

	if err != nil {
		log.Fatal(err)
	}
	defer pool.Close()

	hub := hub2.New()
	repository := postgres.New(pool)
	service := service2.NewService(repository)
	server := server2.New(service, hub)

	r := server.GetRouter()
	if err = r.Run(fmt.Sprintf("%s:%s", config.Host(), config.Port())); err != nil {
		log.Fatal(err)
	}
}

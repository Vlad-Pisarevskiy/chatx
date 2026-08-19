package config

import (
	"log"

	"github.com/joho/godotenv"
)

var values map[string]string

func InitConfig() {
	var err error
	values, err = godotenv.Read(".env")

	if err != nil {
		log.Fatalf("error to read config: %v", err)
	}
}

func Port() string {
	return values["PORT"]
}

func Host() string {
	return values["HOST"]
}

func DBPath() string {
	return values["DATABASE_URL"]
}

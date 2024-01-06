package main

import (
	"log/slog"
	"os"
	"racebot-vk/ergast"
	"racebot-vk/service"
	vk_api "racebot-vk/vk"

	"github.com/joho/godotenv"
)

func init() {
	if err := godotenv.Load("../.env"); err != nil {
		slog.Error("No .env file found")
	}
}

func main() {

	log := setupLogger()

	token, ok := os.LookupEnv("RACEVK_BOT")
	if !ok {
		log.Error("Error getting environment with key")
	} else {
		log.Info("OK load key environment")
	}

	ergastAPI := ergast.NewErgastAPI()
	service := service.NewServiceF1(ergastAPI)

	vkAPI, err := vk_api.NewVKAPI(token, service, service)
	if err != nil {
		log.Error("Error vkApi object")
	}

	vkAPI.Run(log)
}

func setupLogger() *slog.Logger {
	return slog.New(slog.NewJSONHandler(os.Stdout, nil))
}

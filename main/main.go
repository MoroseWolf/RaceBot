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

	vkGroupToken, ok := os.LookupEnv("RACEVK_BOT")
	if !ok {
		log.Error("Error getting environment with vkGroupToken")
	} else {
		log.Info("OK load vkToken environment")
	}

	vkUserToken, ok := os.LookupEnv("USERTOKEN_VK")
	if !ok {
		log.Error("Error getting environment with vkGroupToken")
	} else {
		log.Info("OK load vkToken environment")
	}

	/*	tgToken, ok := os.LookupEnv("RACETG_BOT")
		if !ok {
			log.Error("Error getting environment with tgToken")
		} else {
			log.Info("OK load tgToken environment")
		}
	*/
	ergastAPI := ergast.NewErgastAPI()
	service := service.NewServiceF1(ergastAPI)

	vkAPI, err := vk_api.NewVKAPI(vkGroupToken, vkUserToken, service, service)
	if err != nil {
		log.Error("Error vkApi object")
		os.Exit(1)
	}
	vkAPI.Run(log)

	//tgAPI, err := tg_api.NewTGAPI(tgToken, service)

}

func setupLogger() *slog.Logger {
	return slog.New(slog.NewJSONHandler(os.Stdout, nil))
}

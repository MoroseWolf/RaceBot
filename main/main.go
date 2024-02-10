package main

import (
	"log/slog"
	"os"
	"racebot-vk/config"
	"racebot-vk/ergast"
	"racebot-vk/service"
	tg_api "racebot-vk/telegram"
	vk_api "racebot-vk/vk"

	"github.com/joho/godotenv"
)

func init() {
	if err := godotenv.Load("../data.env"); err != nil {
		slog.Error("No .env file found")
	} else {
		slog.Info("Successfull read .env")
	}
}

func main() {

	conf := config.New()
	log := setupLogger()

	ergastAPI := ergast.NewErgastAPI()
	service := service.NewServiceF1(ergastAPI)

	vkAPI, err := vk_api.NewVKAPI(conf.VkGroupToken, conf.VkUserToken, service, service)
	if err != nil {
		log.Error("Error vkApi object")
		os.Exit(1)
	}

	tgAPI, err := tg_api.NewTGAPI(conf.TgChatToken, service)
	if err != nil {
		log.Error("Error tgApi object")
		os.Exit(1)
	}

	go vkAPI.Run(log)
	tgAPI.Run(log)
}

func setupLogger() *slog.Logger {
	return slog.New(slog.NewJSONHandler(os.Stdout, nil))
}

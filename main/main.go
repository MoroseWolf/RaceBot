package main

import (
	"log/slog"
	"os"
	"path/filepath"
	"racebot-vk/config"
	"racebot-vk/service"
	"racebot-vk/storage/ergast"
	predStorage "racebot-vk/storage/prediction"
	tg_api "racebot-vk/telegram"
	vk_api "racebot-vk/vk"

	"github.com/joho/godotenv"
)

func init() {
	// Пытаемся загрузить .env из нескольких мест:
	// 1. Рядом с бинарником (для Docker-контейнера)
	// 2. В корне проекта (../.env относительно main/) — для локального запуска go run ./main
	// 3. Текущая рабочая директория
	paths := []string{
		filepath.Join(filepath.Dir(os.Args[0]), ".env"),
		"../.env",
		".env",
	}

	loaded := false
	for _, p := range paths {
		if err := godotenv.Load(p); err == nil {
			slog.Info("Successfully loaded .env", slog.String("path", p))
			loaded = true
			break
		}
	}
	if !loaded {
		slog.Warn("No .env file found, relying on system environment variables")
	}
}

func main() {

	conf := config.New()
	log := setupLogger()

	vkAPI, tgAPI := setupConnection(conf, log)

	go vkAPI.Run(log)
	tgAPI.Run(log)
}

func setupLogger() *slog.Logger {
	return slog.New(slog.NewJSONHandler(os.Stdout, nil))
}

func setupConnection(conf *config.Config, log *slog.Logger) (*vk_api.VkAPI, *tg_api.TgAPI) {
	ergastAPI := ergast.NewErgastAPI()
	f1Service := service.NewServiceF1(ergastAPI)

	// Инициализация хранилища прогнозов
	predStore, err := predStorage.NewStorage(conf.PredictionDBPath)
	if err != nil {
		log.Error("failed to init prediction storage", slog.Any("error", err))
		os.Exit(1)
	}
	predService := service.NewPredictionService(predStore)

	vkAPI, err := vk_api.NewVKAPI(conf.VkGroupToken, conf.VkUserToken, f1Service, f1Service, predService)
	if err != nil {
		log.Error("Error vkApi object")
		os.Exit(1)
	}

	tgAPI, err := tg_api.NewTGAPI(conf.TgChatToken, f1Service)
	if err != nil {
		log.Error("Error tgApi object")
		os.Exit(1)
	}

	return vkAPI, tgAPI
}

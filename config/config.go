package config

import (
	"fmt"
	"os"
)

type Config struct {
	VkUserToken      string
	VkGroupToken     string
	TgChatToken      string
	PredictionDBPath string
}

func New() *Config {
	dbPath := os.Getenv("PREDICTION_DB_PATH")
	if dbPath == "" {
		dbPath = "/data/predictions.db"
	}

	return &Config{
		VkGroupToken:     getEnv("RACEVK_BOT"),
		VkUserToken:      getEnv("USERTOKEN_VK"),
		TgChatToken:      getEnv("RACETG_BOT"),
		PredictionDBPath: dbPath,
	}
}

func getEnv(key string) string {
	value, ok := os.LookupEnv(key)
	if !ok {
		panic(fmt.Sprintf("error getting environment %s.", key))
	} else {
		return value
	}
}

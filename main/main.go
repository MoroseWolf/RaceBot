package main

import (
	"log"
	"os"
	"racebot-vk/ergast"
	"racebot-vk/service"
	vk_api "racebot-vk/vk"

	"github.com/joho/godotenv"
)

func init() {
	if err := godotenv.Load("../.env"); err != nil {
		log.Print("No .env file found")
	}
}

func main() {

	token, _ := os.LookupEnv("RACEVK_BOT")

	ergastAPI := ergast.NewErgastAPI()

	service := service.NewServiceF1(ergastAPI)

	vkAPI, err := vk_api.NewVKAPI(token, service, service)
	if err != nil {
		log.Fatalln(err)
	}

	vkAPI.Run()
}

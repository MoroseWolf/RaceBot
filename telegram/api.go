package telegram

import (
	"fmt"
	"log/slog"
	"time"

	"github.com/mymmrac/telego"
	th "github.com/mymmrac/telego/telegohandler"
	tu "github.com/mymmrac/telego/telegoutil"
)

type messageService interface {
	GetDriverStandingsMessage(userDate time.Time) (string, error)
	GetConstructorStandingsMessage(uerDate time.Time) (string, error)
	GetCalendarMessage(year int) (string, error)
	GetNextRaceMessage(userDate time.Time, userTimestamp int) (string, error)
	GetRaceResultsMessage(userDate time.Time, raceId string) (string, error)
	GetCountDaysAfterRaceMessage(userDate time.Time, raceId string) (string, error)
}

type TgAPI struct {
	bot            *telego.Bot
	updates        <-chan telego.Update
	messageService messageService
	handler        *th.BotHandler
}

func NewTGAPI(token string, messageService messageService) (*TgAPI, error) {
	bot, err := telego.NewBot(token)
	if err != nil {
		return nil, fmt.Errorf("error create tg bot from token: %w", err)
	}

	updates, err := bot.UpdatesViaLongPolling(nil)
	if err != nil {
		return nil, fmt.Errorf("error taking updates from longpool: %w", err)
	}

	return &TgAPI{bot: bot, updates: updates, messageService: messageService}, nil

}

func (tg *TgAPI) Run(log *slog.Logger) {

	var err error
	tg.handler, err = th.NewBotHandler(tg.bot, tg.updates)
	if err != nil {
		log.Error("%w", err)
	}
	tg.messageHandler(log)
	tg.handler.Start()
	defer tg.handler.Stop()
	defer tg.bot.StopLongPolling()
}

func (tg *TgAPI) messageHandler(log *slog.Logger) {

	tg.handler.Handle(func(bot *telego.Bot, update telego.Update) {

		log.Info(
			"MESSAGE info",
			slog.Int("peer_id", int(update.Message.Chat.ID)),
			slog.String("text", update.Message.Text))

		userDate := getDateFromMessage(update.Message.Date)

		messageToUser, err := tg.messageService.GetDriverStandingsMessage(userDate)
		if err != nil {
			log.Error("Error with driver standings", err)
		}

		_, err = bot.SendMessage(tu.Message(
			tu.ID(update.Message.Chat.ID),
			messageToUser,
		))
		if err != nil {
			log.Error("Error sending message with driver standings", err)
		}
	}, th.CommandEqual("driverstandings"))

	tg.handler.Handle(func(bot *telego.Bot, update telego.Update) {

		log.Info(
			"MESSAGE info",
			slog.Int("peer_id", int(update.Message.Chat.ID)),
			slog.String("text", update.Message.Text))

		userDate := getDateFromMessage(update.Message.Date)

		messageToUser, err := tg.messageService.GetCalendarMessage(userDate.Year())
		if err != nil {
			log.Error("Error with calendar", err)
		}

		_, err = bot.SendMessage(tu.Message(
			tu.ID(update.Message.Chat.ID),
			messageToUser,
		))
		if err != nil {
			log.Error("Error sending message with calendar", err)
		}
	}, th.CommandEqual("calendar"))

	tg.handler.Handle(func(bot *telego.Bot, update telego.Update) {

		log.Info(
			"MESSAGE info",
			slog.Int("peer_id", int(update.Message.Chat.ID)),
			slog.String("text", update.Message.Text))

		userDate := getDateFromMessage(update.Message.Date)

		messageToUser, err := tg.messageService.GetConstructorStandingsMessage(userDate)
		if err != nil {
			log.Error("Error with constructorstandings", err)
		}

		_, err = bot.SendMessage(tu.Message(
			tu.ID(update.Message.Chat.ID),
			messageToUser,
		))
		if err != nil {
			log.Error("Error sending message with constructorstandings", err)
		}
	}, th.CommandEqual("constructorstandings"))

	tg.handler.Handle(func(bot *telego.Bot, update telego.Update) {

		log.Info(
			"MESSAGE info",
			slog.Int("peer_id", int(update.Message.Chat.ID)),
			slog.String("text", update.Message.Text))

		userDate := getDateFromMessage(update.Message.Date)

		messageToUser, err := tg.messageService.GetRaceResultsMessage(userDate, "last")
		if err != nil {
			log.Error("Error with lastrace", err)
		}

		_, err = bot.SendMessage(tu.Message(
			tu.ID(update.Message.Chat.ID),
			messageToUser,
		))
		if err != nil {
			log.Error("Error sending message with lastrace", err)
		}
	}, th.CommandEqual("lastrace"))

	tg.handler.Handle(func(bot *telego.Bot, update telego.Update) {

		log.Info(
			"MESSAGE info",
			slog.Int("peer_id", int(update.Message.Chat.ID)),
			slog.String("text", update.Message.Text))

		userDate := getDateFromMessage(update.Message.Date)

		messageToUser, err := tg.messageService.GetNextRaceMessage(userDate, int(update.Message.Date))
		if err != nil {
			log.Error("Error with nextrace", err)
		}

		_, err = bot.SendMessage(tu.Message(
			tu.ID(update.Message.Chat.ID),
			messageToUser,
		))
		if err != nil {
			log.Error("Error sending message with nextrace", err)
		}
	}, th.CommandEqual("nextrace"))

	tg.handler.Handle(func(bot *telego.Bot, update telego.Update) {

		log.Info(
			"MESSAGE info",
			slog.Int("peer_id", int(update.Message.Chat.ID)),
			slog.String("text", update.Message.Text))

		userDate := getDateFromMessage(update.Message.Date)

		messageToUser, err := tg.messageService.GetCountDaysAfterRaceMessage(userDate, "last")
		if err != nil {
			log.Error("Error with daysafterrace", err)
		}

		_, err = bot.SendMessage(tu.Message(
			tu.ID(update.Message.Chat.ID),
			messageToUser,
		))
		if err != nil {
			log.Error("Error sending message with daysafterrace", err)
		}
	}, th.CommandEqual("daysafterrace"))
}

func getDateFromMessage(userTimestamp int64) time.Time {
	return time.Unix(int64(userTimestamp), 0)
}

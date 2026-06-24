package telegram

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/mymmrac/telego"
	th "github.com/mymmrac/telego/telegohandler"
	tu "github.com/mymmrac/telego/telegoutil"
)

type messageService interface {
	GetDriverStandingsMessage(userDate time.Time) (string, error)
	GetConstructorStandingsMessage(userDate time.Time) (string, error)
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
	cancel         context.CancelFunc
}

func NewTGAPI(token string, messageService messageService) (*TgAPI, error) {
	bot, err := telego.NewBot(token)
	if err != nil {
		return nil, fmt.Errorf("error create tg bot from token: %w", err)
	}

	ctx, cancel := context.WithCancel(context.Background())

	updates, err := bot.UpdatesViaLongPolling(ctx, nil)
	if err != nil {
		cancel()
		return nil, fmt.Errorf("error taking updates from longpool: %w", err)
	}

	return &TgAPI{bot: bot, updates: updates, messageService: messageService, cancel: cancel}, nil

}

func (tg *TgAPI) Run(log *slog.Logger) {

	var err error
	tg.handler, err = th.NewBotHandler(tg.bot, tg.updates)
	if err != nil {
		log.Error("failed to create bot handler", slog.Any("error", err))
	}
	tg.messageHandler(log)
	tg.handler.Start()
	defer tg.handler.Stop()
	defer tg.cancel()
}

// sendReply отправляет ответ пользователю и логирует результат
func (tg *TgAPI) sendReply(ctx *th.Context, log *slog.Logger, chatID int64, message string, commandName string) {
	_, err := ctx.Bot().SendMessage(ctx.Context(), tu.Message(
		tu.ID(chatID),
		message,
	))
	if err != nil {
		log.Error("failed to send message",
			slog.String("command", commandName),
			slog.Int64("chat_id", chatID),
			slog.Any("error", err))
	}
}

func (tg *TgAPI) messageHandler(log *slog.Logger) {

	tg.handler.Handle(func(ctx *th.Context, update telego.Update) error {

		log.Info(
			"MESSAGE info",
			slog.Int64("peer_id", update.Message.Chat.ID),
			slog.String("text", update.Message.Text))

		userDate := getDateFromMessage(update.Message.Date)

		messageToUser, err := tg.messageService.GetDriverStandingsMessage(userDate)
		if err != nil {
			log.Error("failed to get driver standings", slog.Any("error", err))
		}

		tg.sendReply(ctx, log, update.Message.Chat.ID, messageToUser, "driverstandings")
		return nil
	}, th.CommandEqual("driverstandings"))

	tg.handler.Handle(func(ctx *th.Context, update telego.Update) error {

		log.Info(
			"MESSAGE info",
			slog.Int64("peer_id", update.Message.Chat.ID),
			slog.String("text", update.Message.Text))

		userDate := getDateFromMessage(update.Message.Date)

		messageToUser, err := tg.messageService.GetCalendarMessage(userDate.Year())
		if err != nil {
			log.Error("failed to get calendar", slog.Any("error", err))
		}

		tg.sendReply(ctx, log, update.Message.Chat.ID, messageToUser, "calendar")
		return nil
	}, th.CommandEqual("calendar"))

	tg.handler.Handle(func(ctx *th.Context, update telego.Update) error {

		log.Info(
			"MESSAGE info",
			slog.Int64("peer_id", update.Message.Chat.ID),
			slog.String("text", update.Message.Text))

		userDate := getDateFromMessage(update.Message.Date)

		messageToUser, err := tg.messageService.GetConstructorStandingsMessage(userDate)
		if err != nil {
			log.Error("failed to get constructor standings", slog.Any("error", err))
		}

		tg.sendReply(ctx, log, update.Message.Chat.ID, messageToUser, "constructorstandings")
		return nil
	}, th.CommandEqual("constructorstandings"))

	tg.handler.Handle(func(ctx *th.Context, update telego.Update) error {

		log.Info(
			"MESSAGE info",
			slog.Int64("peer_id", update.Message.Chat.ID),
			slog.String("text", update.Message.Text))

		userDate := getDateFromMessage(update.Message.Date)

		messageToUser, err := tg.messageService.GetRaceResultsMessage(userDate, "last")
		if err != nil {
			log.Error("failed to get last race", slog.Any("error", err))
		}

		tg.sendReply(ctx, log, update.Message.Chat.ID, messageToUser, "lastrace")
		return nil
	}, th.CommandEqual("lastrace"))

	tg.handler.Handle(func(ctx *th.Context, update telego.Update) error {

		log.Info(
			"MESSAGE info",
			slog.Int64("peer_id", update.Message.Chat.ID),
			slog.String("text", update.Message.Text))

		userDate := getDateFromMessage(update.Message.Date)

		messageToUser, err := tg.messageService.GetNextRaceMessage(userDate, int(update.Message.Date))
		if err != nil {
			log.Error("failed to get next race", slog.Any("error", err))
		}

		tg.sendReply(ctx, log, update.Message.Chat.ID, messageToUser, "nextrace")
		return nil
	}, th.CommandEqual("nextrace"))

	tg.handler.Handle(func(ctx *th.Context, update telego.Update) error {

		log.Info(
			"MESSAGE info",
			slog.Int64("peer_id", update.Message.Chat.ID),
			slog.String("text", update.Message.Text))

		userDate := getDateFromMessage(update.Message.Date)

		messageToUser, err := tg.messageService.GetCountDaysAfterRaceMessage(userDate, "last")
		if err != nil {
			log.Error("failed to get days after race", slog.Any("error", err))
		}

		tg.sendReply(ctx, log, update.Message.Chat.ID, messageToUser, "daysafterrace")
		return nil
	}, th.CommandEqual("daysafterrace"))
}

func getDateFromMessage(userTimestamp int64) time.Time {
	return time.Unix(int64(userTimestamp), 0)
}

package vk

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"racebot-vk/models"
	"racebot-vk/service"
	"strings"
	"sync"
	"time"

	"github.com/SevereCloud/vksdk/v3/api"
	"github.com/SevereCloud/vksdk/v3/api/params"
	"github.com/SevereCloud/vksdk/v3/events"
	"github.com/SevereCloud/vksdk/v3/longpoll-bot"
)

const (
	f1memesChatId = 2000000003
	//testChatId      = 2000000005
	//alphaTestChatId = 2000000006
	//f1memesStreamer = 152819213
	botAdminId = 147506714
	f1memesId  = -211183989
	//testGroupId     = -210295709
)

var lastStreamId = 0

// Состояние отслеживания новых видео (защита от повторного запуска/остановки)
var (
	streamCheckMu  sync.Mutex
	streamChecking bool
	streamQuit     chan bool
	streamTicker   *time.Ticker
)

type messageService interface {
	GetDriversListMessage(userDate time.Time) (string, error)
	GetDriverStandingsMessage(userDate time.Time) (string, error)
	GetCalendarMessage(year int) (string, error)
	GetNextRaceMessage(userDate time.Time, userTimestamp int) (string, error)
	GetConstructorStandingsMessage(uerDate time.Time) (string, error)
	GetRaceResultsMessage(userDate time.Time, raceId string) (string, error)
	GetGPInfoCarousel(userDate time.Time, raceId string) (string, error)
	GetGPKeyboard() string
	GetCountDaysAfterRaceMessage(userDate time.Time, raceId string) (string, error)
	GetQualifyingResultsMessage(userDate time.Time, raceId string) (string, error)
	GetSprintResultsMessage(userDate time.Time, raceId string) string
	GetCountOfRaces(userDate time.Time) (int, error)
	GetNextRace(userDate time.Time, userTimestamp int) (models.Race, error)
}

type eventService interface {
	GetGPInfoCarousel(userDate time.Time, raceId string) (string, error)
}

type VkAPI struct {
	usrVk             *api.VK
	lp                *longpoll.LongPoll
	messageService    messageService
	eventService      eventService
	predictionService *service.PredictionService
}

func NewVKAPI(groupToken, userToken string, messageService messageService, eventService eventService, predictionService *service.PredictionService) (*VkAPI, error) {
	vk := api.NewVK(groupToken)

	lp, err := longpoll.NewLongPollCommunity(vk)
	if err != nil {
		return nil, fmt.Errorf("error creating new log pool: %w", err)
	}

	return &VkAPI{
		usrVk:             api.NewVK(userToken),
		lp:                lp,
		messageService:    messageService,
		eventService:      eventService,
		predictionService: predictionService,
	}, nil
}

func (vk *VkAPI) Run(log *slog.Logger) {
	vk.messageHandler(log)
	vk.eventHandler(log)

	log.Info("Start longpoll")
	if err := vk.lp.Run(); err != nil {
		log.Error("longpoll run failed", slog.Any("error", err))
	}
}

func (vk *VkAPI) sendAndLog(log *slog.Logger, message string, peerID int, keyboard, template, attachment *string, commandLabel string) (api.MessagesSendUserIDsResponse, error) {
	resp, err := sendMessageToUser(message, peerID, vk.lp.VK, keyboard, template, attachment)
	if err != nil {
		log.Error("failed to send message",
			slog.String("command", commandLabel),
			slog.Int("peer_id", peerID),
			slog.Any("error", err))
		return nil, err
	}
	if len(resp) > 0 {
		log.Info("Message sent",
			slog.String("command", commandLabel),
			slog.Int("peer_id", resp[0].PeerID),
			slog.Int("message_id", resp[0].MessageID),
			slog.Int("cm_id", resp[0].ConversationMessageID))
	}
	return resp, nil
}

func (vk *VkAPI) messageHandler(log *slog.Logger) {
	var myUsrVk MyVk = MyVk{vk.usrVk}

	vk.lp.MessageNew(func(_ context.Context, obj events.MessageNewObject) {
		log.Info(
			"MESSAGE info",
			slog.Int("peer_id", obj.Message.PeerID),
			slog.String("text", obj.Message.Text))

		userTimestamp := obj.Message.Date
		userDate := time.Unix(int64(userTimestamp), 0)
		messageText := strings.ToLower(obj.Message.Text)

		textPayload, err := extractCommand(obj.Message.Payload)
		if err != nil {
			log.Error("Error reading payload: ", slog.Any("error", err))
		}

		ctx := handlerContext{
			log:           log,
			vk:            vk,
			myUsrVk:       &myUsrVk,
			obj:           obj,
			userDate:      userDate,
			userTimestamp: userTimestamp,
			messageText:   messageText,
			raceID:        "last",
		}

		if textPayload != nil {
			cmd := getCommand(*textPayload)
			ctx.raceID = (strings.Split(*textPayload, "_"))[1]

			if handler, ok := payloadHandlers[cmd]; ok {
				handler(ctx)
			}
		} else {
			cmd := getCommand(messageText)

			if handler, ok := messageHandlers[cmd]; ok {
				handler(ctx)
			} else {
				log.Info("Команда в сообщении не распознана", slog.String("text", obj.Message.Text))
				handleUnknownWithPrediction(ctx)
			}
		}
	})
}

func (vk *VkAPI) eventHandler(log *slog.Logger) {
	vk.lp.MessageEvent(func(_ context.Context, obj events.MessageEventObject) {
		log.Info(
			"EVENT info",
			slog.Int("peer_id", obj.PeerID),
			slog.Any("text", obj.Payload))

		payloadCommand, err := extractCommand(string(obj.Payload))
		if err != nil {
			log.Error("Error reading payload", slog.Any("error", err))
			return
		}
		if payloadCommand == nil {
			return
		}

		cmd := getEventCommand(*payloadCommand)

		ctx := eventHandlerContext{
			log:     log,
			vk:      vk,
			obj:     obj,
			payload: *payloadCommand,
		}

		if handler, ok := eventHandlers[cmd]; ok {
			handler(ctx)
		}
	})
}

func sendMessageToUser(messageToUser string, peerID int, vk *api.VK, keyboard, template, attachment *string) (api.MessagesSendUserIDsResponse, error) {
	b := params.NewMessagesSendBuilder()
	b.Message(messageToUser)
	b.RandomID(0)
	b.PeerIDs([]int{peerID})

	if keyboard != nil {
		b.Keyboard(*keyboard)
	}
	if template != nil {
		b.Template(*template)
	}
	if attachment != nil {
		b.Attachment(*attachment)
	}

	msgId, err := vk.MessagesSendPeerIDs(b.Params)
	if err != nil {
		return nil, fmt.Errorf("error sending message to user: %w", err)
	}
	return msgId, nil
}

func sendEventMessageToUser(vk *api.VK, peerID int, eventID string, userID int) (int, error) {
	prms := params.NewMessagesSendMessageEventAnswerBuilder()
	prms.PeerID(peerID)
	prms.EventID(eventID)
	prms.UserID(userID)

	resp, err := vk.MessagesSendMessageEventAnswer(prms.Params)
	if err != nil {
		return resp, fmt.Errorf("error sending message to user: %w", err)
	}
	return resp, nil
}

func deleteMessages(vk *api.VK, messageIds []int, peerID int, deleteForAllFlag bool) error {
	prms := params.NewMessagesDeleteBuilder()
	prms.PeerID(peerID)
	prms.DeleteForAll(deleteForAllFlag)
	prms.ConversationMessageIDs(messageIds)

	resp, err := vk.MessagesDelete(prms.Params)
	if err != nil {
		return fmt.Errorf("error deleting message: %w", err)
	}
	slog.Info("Response deleting message", slog.Any("id", resp))
	return nil
}

func extractCommand(payload string) (*string, error) {
	var pl Payload
	if payload != "" {
		err := json.Unmarshal([]byte(payload), &pl)
		if err != nil {
			return nil, fmt.Errorf("error unmarshal command in payload message: %w", err)
		}
		slog.Debug("Command from paylpad", slog.String("Command", pl.Command))
		return &pl.Command, nil
	}
	return nil, nil
}

func makeKeyboard(row, col, numPage, countEl int, inline bool) (Kb, error) {
	var button Button
	btnsRow := make([]Button, 0, row)
	buttons := [][]Button{}
	sizeKb := row * col

	if countEl == 0 {
		return Kb{Inline: inline, Buttons: buttons}, nil
	}

	visKb := countEl - sizeKb*(numPage-1)
	if visKb > sizeKb {
		visKb = sizeKb
	}
	if visKb <= 0 {
		return Kb{}, fmt.Errorf("с заданными параметрами невозможно отобразить клавиатуру. Для количества элементов %d не существует %d-ой страницы клавиатуры при %d кнопках", countEl, numPage, sizeKb)
	}
	addedNum := sizeKb * (numPage - 1)
	for i := 1; i <= visKb; i++ {
		button = Button{Action: ActionBtn{TypeAction: "callback", Label: fmt.Sprintf("%d", i+addedNum), Payload: fmt.Sprintf(`{"command" : "gpPage_%d"}`, i+addedNum)}}
		btnsRow = append(btnsRow, button)

		if (i%col == 0) || (i == visKb) {
			buttons = append(buttons, btnsRow)
			btnsRow = nil
		}
	}

	switch numPage {
	case 1:
		buttons = append(buttons,
			[]Button{{Action: ActionBtn{TypeAction: "callback", Label: "Далее", Payload: `{"command" : "gpListPage_2", "message_id":""}`}, Color: "primary"}})
	case 2:
		buttons = append(buttons,
			[]Button{{Action: ActionBtn{TypeAction: "callback", Label: "Назад", Payload: `{"command" : "gpListPage_1"}`}, Color: "primary"},
				{Action: ActionBtn{TypeAction: "callback", Label: "Далее", Payload: `{"command" : "gpListPage_3"}`}, Color: "primary"}})
	case 3:
		buttons = append(buttons,
			[]Button{{Action: ActionBtn{TypeAction: "callback", Label: "Назад", Payload: `{"command" : "gpListPage_2"}`}, Color: "primary"},
				{Action: ActionBtn{TypeAction: "callback", Label: "В начало", Payload: `{"command" : "gpListPage_1"}`}, Color: "primary"}})
	}

	return Kb{Inline: inline, Buttons: buttons}, nil
}

func getLastVideos(vk MyVk, count int) ([]MyVideo, error) {
	prms := params.NewVideoGetBuilder()
	prms.OwnerID(f1memesId)
	prms.Count(count)

	resp, err := vk.VideoGet(prms.Params)
	if err != nil {
		return nil, fmt.Errorf("error in video.get: %w", err)
	}

	return resp.Items, nil
}

// checkStreamOnce выполняет одну проверку новых видео и возвращает false,
// если отслеживание нужно прекратить (произошла ошибка получения видео).
func checkStreamOnce(log *slog.Logger, vk *VkAPI, myUsrVk *MyVk, obj events.MessageNewObject) bool {
	lastVideo, err := getLastVideos(*myUsrVk, 2)
	if err != nil {
		log.Error(err.Error())
		_, err := sendMessageToUser("Ошибка получения новых видео. Перезапустите отслеживание.", botAdminId, vk.lp.VK, nil, nil, nil)
		if err != nil {
			log.Error("Error with sending message-answer to command `checkStream` to user", slog.Int("peer_id", obj.Message.PeerID), slog.Any("error", err))
		}
		return false
	}
	log.Info("Video id", slog.Int("ID", lastVideo[0].ID))

	if lastVideo[0].ID != lastStreamId {
		if lastVideo[0].Live && lastVideo[0].LiveStatus == "started" {
			lastStreamId = lastVideo[0].ID
			streamLink := fmt.Sprintf("video%d_%d", f1memesId, lastStreamId)

			messageToUser := fmt.Sprintf("'F1 Memes TV' начали трансляцию '%s'!\n", lastVideo[0].Title)
			resp, err := sendMessageToUser(messageToUser, f1memesChatId, vk.lp.VK, nil, nil, &streamLink)
			if err != nil {
				log.Error("Error with sending message-answer to command `checkStream` to user", slog.Int("peer_id", obj.Message.PeerID), slog.Any("error", err))
			}

			log.Info("Video link", slog.String("video_id", streamLink))
			log.Info("Message sent", slog.Group("response", slog.Int("peer_id", resp[0].PeerID), slog.Int("message_id", resp[0].MessageID), slog.Int("cm_id", resp[0].ConversationMessageID)))
		}
	}
	return true
}

func checkLastStream(quit <-chan bool, ticker *time.Ticker, log *slog.Logger, vk *VkAPI, myUsrVk *MyVk, obj events.MessageNewObject) {
	defer func() {
		ticker.Stop()
		streamCheckMu.Lock()
		streamChecking = false
		streamQuit = nil
		streamTicker = nil
		streamCheckMu.Unlock()
		log.Info("End video check")
	}()

	for {
		select {
		case <-quit:
			return
		case t := <-ticker.C:
			log.Info("Video check", slog.String("time", t.UTC().String()))
			if !checkStreamOnce(log, vk, myUsrVk, obj) {
				return
			}
		}
	}
}

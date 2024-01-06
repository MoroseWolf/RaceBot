package vk

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/SevereCloud/vksdk/v2/api"
	"github.com/SevereCloud/vksdk/v2/api/params"
	"github.com/SevereCloud/vksdk/v2/events"
	"github.com/SevereCloud/vksdk/v2/longpoll-bot"
)

const (
	f1memesId       = 2000000003
	f1memesStreamer = 152819213
)

type messageService interface {
	GetDriverStandingsMessage(userDate time.Time) (string, error)
	GetCalendarMessage(year int) (string, error)
	GetNextRaceMessage(userDate time.Time, userTimestamp int) (string, error)
	GetConstructorStandingsMessage(uerDate time.Time) (string, error)
	GetRaceResultsMessage(userDate time.Time, raceId string) (string, error)
	GetGPInfoCarousel(userDate time.Time, raceId string) string
	GetGPKeyboard() string
	GetCountDaysAfterRaceMessage(userDate time.Time, raceId string) string
	GetQualifyingResultsMessage(userDate time.Time, raceId string) string
	GetSprintResultsMessage(userDate time.Time, raceId string) string
}

type eventService interface {
	GetGPInfoCarousel(userDate time.Time, raceId string) string
}

type VkAPI struct {
	lp             *longpoll.LongPoll
	messageService messageService
	eventService   eventService
}

func NewVKAPI(token string, messageService messageService, eventService eventService) (*VkAPI, error) {
	vk := api.NewVK(token)

	group, err := vk.GroupsGetByID(api.Params{})
	if err != nil {
		return nil, fmt.Errorf("error groups get by id: %w", err)
	}

	lp, err := longpoll.NewLongPoll(vk, group[0].ID)
	if err != nil {
		return nil, fmt.Errorf("error creating new log pool: %w", err)
	}

	return &VkAPI{lp: lp, messageService: messageService, eventService: eventService}, nil
}

func (vk *VkAPI) Run(log *slog.Logger) {
	vk.messageHandler(log)
	vk.eventHandler(log)

	log.Info("Start longpoll")
	if err := vk.lp.Run(); err != nil {
		log.Error("%w", err)
	}
}

func (vk *VkAPI) messageHandler(log *slog.Logger) {
	vk.lp.MessageNew(func(_ context.Context, obj events.MessageNewObject) {
		log.Info(
			"MESSAGE info",
			slog.Int("peer_id", obj.Message.PeerID),
			slog.String("text", obj.Message.Text))

		var messageToUser string
		var command command
		var raceId string

		userTimestamp := obj.Message.Date
		userDate := time.Unix(int64(userTimestamp), 0)

		messageText := strings.ToLower(obj.Message.Text)

		textPayload, err := extractCommand(obj.Message.Payload)
		if err != nil {
			log.Error("Error reading payload: ", slog.Any("", err))
		}

		if textPayload != nil {
			command = getCommand(*textPayload)
			raceId = (strings.Split(*textPayload, "_"))[1]

			switch command {

			case commandRaceRes:
				messageToUser, err = vk.messageService.GetRaceResultsMessage(userDate, raceId)
				if err != nil {
					log.Error("Error with race result", err)
				}

				err := sendMessageToUser(messageToUser, obj.Message.PeerID, vk.lp.VK, nil, nil, nil)
				if err != nil {
					log.Error("Error with sending message-answer to command `commandRaceRes` to user", slog.Int("peer_id", obj.Message.PeerID), slog.Any("error", err))
				}

			case commandQualRes:
				messageToUser := vk.messageService.GetQualifyingResultsMessage(userDate, raceId)
				err := sendMessageToUser(messageToUser, obj.Message.PeerID, vk.lp.VK, nil, nil, nil)
				if err != nil {
					log.Error("Error with sending message-answer to command `commandQualRes` to user", slog.Int("peer_id", obj.Message.PeerID), slog.Any("error", err))
				}

			case commandSprRes:
				messageToUser = vk.messageService.GetSprintResultsMessage(userDate, raceId)
				err := sendMessageToUser(messageToUser, obj.Message.PeerID, vk.lp.VK, nil, nil, nil)
				if err != nil {
					log.Error("Error with sending message-answer to command `commandSprRes` to user", slog.Int("peer_id", obj.Message.PeerID), slog.Any("error", err))
				}
			}
		} else {
			command = getCommand(messageText)
			raceId = "last"

			if checkStream(obj.Message.PeerID, command) {
				streamLink := extractStreamLink(messageText)
				messageToUser = "Трансляция 'F1 Memes TV' началась! Смотри в Telegram t.me/f1memestv и в [vk.com/f1memestv|VK]."
				err := sendMessageToUser(messageToUser, f1memesId, vk.lp.VK, nil, nil, &streamLink)
				if err != nil {
					log.Error("Error with sending message-answer to command `checkStream` to user", slog.Int("peer_id", obj.Message.PeerID), slog.Any("error", err))
				}

			} else {

				switch command {

				case commandHello:
					messageToUser =
						`Привет! Я бот, который делится информацией про F1 :)
					Пока что я могу сказать тебе информацию только о текущем сезоне (но всё ещё впереди).
					Для того чтобы подробнее познакомиться с моими возможностями напиши мне "Что умеешь?". 
					
					Приятного пользования :)`
					err := sendMessageToUser(messageToUser, obj.Message.PeerID, vk.lp.VK, nil, nil, nil)
					if err != nil {
						log.Error("Error with sending message-answer to command `Hello` to user", slog.Int("peer_id", obj.Message.PeerID), slog.Any("error", err))
					}

				case commandHelp:
					messageToUser =
						`Команды которые я понимаю (могу их прочесть в твоём ообщении среди других слов):
						• календарь сезона - список гран-при F1 текущего сезона
						• кубок кострукторов или кк - текущее положение команд в кубке контрукторов
						• личный зачёт - текущее положение гонщиков в личном зачёте
						• следующая гонка - информация о следующем гран-при F1
						• результат гонки - результат последней прошедшей гонки F1
						• дней без формулы/F1 - количество дней с последней гонки F1
				
					!Внимание! Информация, связанная с проведённой гонкой может обновляться не сразу.
					Работаем над этим.`
					err := sendMessageToUser(messageToUser, obj.Message.PeerID, vk.lp.VK, nil, nil, nil)
					if err != nil {
						log.Error("Error with sending message-answer to command `commandHelp` to user", slog.Int("peer_id", obj.Message.PeerID), slog.Any("error", err))
					}

				case commandDrSt:
					messageToUser, err = vk.messageService.GetDriverStandingsMessage(userDate)
					if err != nil {
						log.Error("Error with driver standings", err)
					}

					err := sendMessageToUser(messageToUser, obj.Message.PeerID, vk.lp.VK, nil, nil, nil)
					if err != nil {
						log.Error("Error with sending message-answer to command `commandDrSt` to user", slog.Int("peer_id", obj.Message.PeerID), slog.Any("error", err))
					}

				case commandCld:
					messageToUser, err = vk.messageService.GetCalendarMessage(userDate.Year())
					if err != nil {
						log.Error("Error with calendar", err)
					}

					err := sendMessageToUser(messageToUser, obj.Message.PeerID, vk.lp.VK, nil, nil, nil)
					if err != nil {
						log.Error("Error with sending message-answer to command `commandCld` to user", slog.Int("peer_id", obj.Message.PeerID), slog.Any("error", err))
					}

				case commandNxRc:
					messageToUser, err = vk.messageService.GetNextRaceMessage(userDate, userTimestamp)
					if err != nil {
						log.Error("Error with nextRace", err)
						break
					}
					err := sendMessageToUser(messageToUser, obj.Message.PeerID, vk.lp.VK, nil, nil, nil)
					if err != nil {
						log.Error("Error with sending message-answer to command `commandNxRc` to user", slog.Int("peer_id", obj.Message.PeerID), slog.Any("error", err))
					}

				case commandConsStFull, commandConsSt:
					messageToUser, err = vk.messageService.GetConstructorStandingsMessage(userDate)
					if err != nil {
						log.Error("Error with constructor standings", err)
					}

					err := sendMessageToUser(messageToUser, obj.Message.PeerID, vk.lp.VK, nil, nil, nil)
					if err != nil {
						log.Error("Error with sending message-answer to command `commandConsStFull` to user", slog.Int("peer_id", obj.Message.PeerID), slog.Any("error", err))
					}

				case commandLstRc:
					messageToUser, err = vk.messageService.GetRaceResultsMessage(userDate, raceId)
					if err != nil {
						log.Error("Error with last result", err)
					}
					err := sendMessageToUser(messageToUser, obj.Message.PeerID, vk.lp.VK, nil, nil, nil)
					if err != nil {
						log.Error("Error with sending message-answer to command `commandLstRc` to user", slog.Int("peer_id", obj.Message.PeerID), slog.Any("error", err))
					}

				case commandLstGP:
					crsl := vk.messageService.GetGPInfoCarousel(userDate, raceId)
					err := sendMessageToUser("Информация о гран-при:", obj.Message.PeerID, vk.lp.VK, nil, &crsl, nil)
					if err != nil {
						log.Error("Error with sending message-answer to command `commandLstGP` to user", slog.Int("peer_id", obj.Message.PeerID), slog.Any("error", err))
					}

				case commandGPs:
					//kb := vk.messageService.GetGPKeyboard()

					kb, err := makeKeyboard(2, 4, 1, 24, false)
					if err != nil {
						log.Error("Error creating keyboard", slog.Any("error", err))
					}

					jsKb, err := json.Marshal(kb)
					if err != nil {
						log.Error("Error marshal keyboard", slog.Any("error", err))
					}

					strKb := string(jsKb)
					err = sendMessageToUser("Этапы F1:", obj.Message.PeerID, vk.lp.VK, &strKb, nil, nil)
					if err != nil {
						log.Error("Error with sending message-answer to command `commandGPs` to user", slog.Int("peer_id", obj.Message.PeerID), slog.Any("error", err))
					}

				case commandDaysAfterRace:
					messageToUser := vk.messageService.GetCountDaysAfterRaceMessage(userDate, raceId)
					err := sendMessageToUser(messageToUser, obj.Message.PeerID, vk.lp.VK, nil, nil, nil)
					if err != nil {
						log.Error("Error with sending message-answer to command `commandDaysAfterRace` to user", slog.Int("peer_id", obj.Message.PeerID), slog.Any("error", err))
					}

				case commandLstQual:
					messageToUser := vk.messageService.GetQualifyingResultsMessage(userDate, raceId)
					err := sendMessageToUser(messageToUser, obj.Message.PeerID, vk.lp.VK, nil, nil, nil)
					if err != nil {
						log.Error("Error with sending message-answer to command `commandLstQual` to user", slog.Int("peer_id", obj.Message.PeerID), slog.Any("error", err))
					}
				default:
					log.Info("Команда в сообщении не распознана", slog.String("text", obj.Message.Text))

				}
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
		}
		command := getEventCommand(*payloadCommand)

		switch command {

		case commandGpList1, commandGpList2, commandGpList3:
			var numPage int

			switch command {
			case commandGpList1:
				numPage = 1
			case commandGpList2:
				numPage = 2
			case commandGpList3:
				numPage = 3
			}

			newKeyboard, err := makeKeyboard(2, 4, numPage, 24, false)
			if err != nil {
				log.Error("Error making keyboard", slog.Any("error", err))
				break
			}

			jsKb, err := json.Marshal(newKeyboard)
			if err != nil {
				log.Error("Error marshall keyboard", slog.Any("error", err))
			}
			strKb := string(jsKb)

			messageToUser := "Обновление"

			err = sendMessageToUser(messageToUser, obj.PeerID, vk.lp.VK, &strKb, nil, nil)
			if err != nil {
				log.Error("Error with sending message-answer to command `commandGpList1` to user", slog.Int("peer_id", obj.PeerID), slog.Any("error", err))
			}

			err = sendEventMessageToUser(vk.lp.VK, obj.PeerID, obj.EventID, obj.UserID)
			if err != nil {
				log.Error("Error with sending event-answer to command `commandGpList1` to user", slog.Int("peer_id", obj.PeerID), slog.Any("error", err))
			}

		case commandGpInfo:

			timeNow := time.Now()
			number := strings.Split(*payloadCommand, "_")

			curRace := vk.eventService.GetGPInfoCarousel(timeNow, number[1])

			fmt.Println(curRace)

			messageToUser := "Информация о гран-при:"

			err := sendMessageToUser(messageToUser, obj.PeerID, vk.lp.VK, nil, &curRace, nil)
			if err != nil {
				log.Error("Error with sending message-answer to command `commandGpInfo` to user", slog.Int("peer_id", obj.PeerID), slog.Any("error", err))
			}
			err = sendEventMessageToUser(vk.lp.VK, obj.PeerID, obj.EventID, obj.UserID)
			if err != nil {
				log.Error("Error with sending event-answer to command `commandGpInfo` to user", slog.Int("peer_id", obj.PeerID), slog.Any("error", err))
			}

		}
	})
}

func sendMessageToUser(messageToUser string, peerID int, vk *api.VK, keyboard, template, attachment *string) error {
	b := params.NewMessagesSendBuilder()
	b.Message(messageToUser)
	b.RandomID(0)
	b.PeerID(peerID)

	if keyboard != nil {
		b.Keyboard(*keyboard)
	}
	if template != nil {
		b.Template(*template)
	}

	if attachment != nil {
		b.Attachment(*attachment)
	}

	msgId, err := vk.MessagesSend(b.Params)
	if err != nil {
		return fmt.Errorf("error sending message to user: %w", err)
	}
	slog.Info("Message-answer sended", slog.Int("id", msgId))
	return nil
}

func sendEventMessageToUser(vk *api.VK, peerID int, eventID string, userID int) error {
	prms := params.NewMessagesSendMessageEventAnswerBuilder()
	prms.PeerID(peerID)
	prms.EventID(eventID)
	prms.UserID(userID)

	resp, err := vk.MessagesSendMessageEventAnswer(prms.Params)
	if err != nil {
		return fmt.Errorf("error sending message to user: %w", err)
	}
	slog.Info("Responce sended MessageEvent", slog.Int("id", resp))
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
	} else {
		return nil, nil
	}
}

func makeKeyboard(row, col, numPage, countEl int, inline bool) (Kb, error) {

	var button Button
	btnsRow := make([]Button, 0, row)
	buttons := [][]Button{}
	sizeKb := row * col

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

func checkStream(id int, command command) bool {
	if (id == f1memesStreamer) && (command == commandStream) {
		return true
	}
	return false
}

func extractStreamLink(messageText string) string {
	msgParts := strings.Split(messageText, " ")
	link := strings.TrimPrefix(msgParts[1], "https://vk.com/")
	return link
}

// ----------------------------------
//
//	неиспользуемые на данный момент функции
//
// ----------------------------------
/*
func deleteMention(messageText string) string {
	messageText = strings.Replace(messageText, ", ", "", 1)
	messageText = strings.TrimPrefix(messageText, "[club219009582|@club219009582]")
	messageText = strings.TrimPrefix(messageText, "[club219009582|Race Bot]")
	return messageText
}
*/

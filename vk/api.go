package vk

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"racebot-vk/models"
	"strconv"
	"strings"
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

// обработка команд в payload сообщения
type eventService interface {
	GetGPInfoCarousel(userDate time.Time, raceId string) (string, error)
}

// прогноз пользователя
type Prediction struct {
	first  uint8
	second uint8
	third  uint8
}

type VkAPI struct {
	usrVk                *api.VK
	lp                   *longpoll.LongPoll
	messageService       messageService
	eventService         eventService
	predictionActive     bool
	predictionMessageIDs map[int]int //map[chat_id]message_id
	//predictions          map[int]Prediction
	raceResult Prediction
}

func NewVKAPI(groupToken, userToken string, messageService messageService, eventService eventService) (*VkAPI, error) {
	vk := api.NewVK(groupToken)

	lp, err := longpoll.NewLongPollCommunity(vk)
	if err != nil {
		return nil, fmt.Errorf("error creating new log pool: %w", err)
	}

	return &VkAPI{
		usrVk:                api.NewVK(userToken),
		lp:                   lp,
		messageService:       messageService,
		eventService:         eventService,
		predictionActive:     false,
		predictionMessageIDs: make(map[int]int),
		//predictions:         make(map[int]Prediction),
		raceResult: Prediction{},
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

// sendAndLog отправляет сообщение и логирует результат. Возвращает ответ API.
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
	quit := make(chan bool)
	var myUsrVk MyVk = MyVk{vk.usrVk}
	//var currentRaceID int

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
			log.Error("Error reading payload: ", slog.Any("error", err))
		}

		if textPayload != nil {
			command = getCommand(*textPayload)
			raceId = (strings.Split(*textPayload, "_"))[1]

			switch command {

			case commandRaceRes:
				messageToUser, err = vk.messageService.GetRaceResultsMessage(userDate, raceId)
				if err != nil {
					log.Error("failed to get race result", slog.Any("error", err))
				}
				vk.sendAndLog(log, messageToUser, obj.Message.PeerID, nil, nil, nil, "raceRes")

			case commandQualRes:
				messageToUser, err := vk.messageService.GetQualifyingResultsMessage(userDate, raceId)
				if err != nil {
					log.Error("failed to get qualifying result", slog.Any("error", err))
				}
				vk.sendAndLog(log, messageToUser, obj.Message.PeerID, nil, nil, nil, "qualRes")

			case commandSprRes:
				messageToUser = vk.messageService.GetSprintResultsMessage(userDate, raceId)
				vk.sendAndLog(log, messageToUser, obj.Message.PeerID, nil, nil, nil, "sprRes")
			}
		} else {
			command = getCommand(messageText)
			raceId = "last"
			/*
				if checkStreamCommand(obj.Message.PeerID, command) {

					ticker := time.NewTicker(5 * time.Minute)
					lastVideo, err := getLastVideos(myUsrVk, 1)
					if err != nil {
						log.Error(err.Error())
					}
					lastStreamId = lastVideo[0].ID
					switch command {

					case commandStartCheckStream:

						messageToUser = "Команда принята!"
						resp, err := sendMessageToUser(messageToUser, obj.Message.PeerID, vk.lp.VK, nil, nil, nil)
						if err != nil {
							log.Error("Error with sending message-answer to command `checkStream` to user", slog.Int("peer_id", obj.Message.PeerID), slog.Any("error", err))
						}
						log.Info("Message sent", slog.Group("response", slog.Int("peer_id", resp[0].PeerID), slog.Int("message_id", resp[0].MessageID), slog.Int("cm_id", resp[0].ConversationMessageID)))
						log.Info("Start video check")

						go checkLastStream(quit, ticker, log, vk, &myUsrVk, obj)

					case commandEndCheckStream:
					ticker.Stop()
						quit <- true
						messageToUser = "Команда принята!"
						resp, err := sendMessageToUser(messageToUser, obj.Message.PeerID, vk.lp.VK, nil, nil, nil)
						if err != nil {
							log.Error("Error with sending message-answer to command `checkStream` to user", slog.Int("peer_id", obj.Message.PeerID), slog.Any("error", err))
						}
						log.Info("Message sent", slog.Group("response", slog.Int("peer_id", resp[0].PeerID), slog.Int("message_id", resp[0].MessageID), slog.Int("cm_id", resp[0].ConversationMessageID)))

					}

				} else {
			*/

			switch command {
			case commandHello:
				messageToUser =
					`Привет! Я бот, который делится информацией про F1 :)
					Пока что я могу сказать тебе информацию только о текущем сезоне (но всё ещё впереди).
					Для того чтобы подробнее познакомиться с моими возможностями напиши мне "Что умеешь?".
					
					Приятного пользования :)`
				vk.sendAndLog(log, messageToUser, obj.Message.PeerID, nil, nil, nil, "hello")

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
				vk.sendAndLog(log, messageToUser, obj.Message.PeerID, nil, nil, nil, "help")

			case commandDrSt:
				messageToUser, err = vk.messageService.GetDriverStandingsMessage(userDate)
				if err != nil {
					log.Error("failed to get driver standings", slog.Any("error", err))
				}
				vk.sendAndLog(log, messageToUser, obj.Message.PeerID, nil, nil, nil, "driverStandings")

			case commandCld:
				messageToUser, err = vk.messageService.GetCalendarMessage(userDate.Year())
				if err != nil {
					log.Error("failed to get calendar", slog.Any("error", err))
				}
				vk.sendAndLog(log, messageToUser, obj.Message.PeerID, nil, nil, nil, "calendar")

			case commandNxRc:
				messageToUser, err = vk.messageService.GetNextRaceMessage(userDate, userTimestamp)
				if err != nil {
					log.Error("failed to get next race", slog.Any("error", err))
					break
				}
				vk.sendAndLog(log, messageToUser, obj.Message.PeerID, nil, nil, nil, "nextRace")

			case commandConsStFull, commandConsSt:
				messageToUser, err = vk.messageService.GetConstructorStandingsMessage(userDate)
				if err != nil {
					log.Error("failed to get constructor standings", slog.Any("error", err))
				}
				vk.sendAndLog(log, messageToUser, obj.Message.PeerID, nil, nil, nil, "constructorStandings")

			case commandLstRc:
				messageToUser, err = vk.messageService.GetRaceResultsMessage(userDate, raceId)
				if err != nil {
					log.Error("failed to get last race result", slog.Any("error", err))
				}
				vk.sendAndLog(log, messageToUser, obj.Message.PeerID, nil, nil, nil, "lastRace")
			case commandLstGP:
				crsl, err := vk.messageService.GetGPInfoCarousel(userDate, raceId)
				if err != nil {
					log.Error("failed to get GP info carousel", slog.Any("error", err))
				}
				vk.sendAndLog(log, "Информация о гран-при:", obj.Message.PeerID, nil, &crsl, nil, "lastGP")

			case commandGPs:
				count, err := vk.messageService.GetCountOfRaces(userDate)
				if err != nil {
					log.Error("failed to get count of races", slog.Any("error", err))
				}

				kb, err := makeKeyboard(2, 4, 1, count, false)
				if err != nil {
					log.Error("failed to create keyboard", slog.Any("error", err))
				}

				jsKb, err := json.Marshal(kb)
				if err != nil {
					log.Error("failed to marshal keyboard", slog.Any("error", err))
				}

				strKb := string(jsKb)
				vk.sendAndLog(log, "Этапы F1:", obj.Message.PeerID, &strKb, nil, nil, "GPs")

			case commandDaysAfterRace, commandDaysAfterRaceСut:
				messageToUser, err := vk.messageService.GetCountDaysAfterRaceMessage(userDate, raceId)
				if err != nil {
					log.Error("failed to get days after race", slog.Int("peer_id", obj.Message.PeerID), slog.Any("error", err))
				}
				vk.sendAndLog(log, messageToUser, obj.Message.PeerID, nil, nil, nil, "daysAfterRace")

			case commandLstQual:
				messageToUser, err := vk.messageService.GetQualifyingResultsMessage(userDate, raceId)
				if err != nil {
					log.Error("failed to get qualifying result", slog.Any("error", err))
				}
				vk.sendAndLog(log, messageToUser, obj.Message.PeerID, nil, nil, nil, "lastQual")

			case commandClsKb:
				kb, err := makeKeyboard(0, 0, 0, 0, false)
				if err != nil {
					log.Error("failed to create keyboard", slog.Any("error", err))
				}

				jsKb, err := json.Marshal(kb)
				if err != nil {
					log.Error("failed to marshal keyboard", slog.Any("error", err))
				}

				strKb := string(jsKb)

				msgResp, err := vk.sendAndLog(log, "Закрываю", obj.Message.PeerID, &strKb, nil, nil, "closeKeyboard")
				if err != nil {
					return
				}

				err = deleteMessages(vk.lp.VK, []int{msgResp[0].ConversationMessageID}, obj.Message.PeerID, true)
				if err != nil {
					log.Error("failed to delete messages", slog.Any("error", err))
				}

			case commandLvrsList:
				photo := "photo-219009582_457239026"
				vk.sendAndLog(log, "Ливреи машин 2024 года", obj.Message.PeerID, nil, nil, &photo, "liveries")

			case commandPredictionAdmin:
				if obj.Message.FromID == botAdminId {
					if vk.predictionActive {
						vk.sendAndLog(log, "Прогноз активен.", obj.Message.PeerID, nil, nil, nil, "predictionAdmin")
					} else {
						nxtRc, err := vk.messageService.GetNextRace(userDate, userTimestamp)
						if err != nil {
							log.Error("failed to get next race for prediction", slog.Any("error", err))
							return
						}
						raceId = nxtRc.Season + "_" + nxtRc.Round
						vk.predictionActive = true

						driversMessage, err := vk.messageService.GetDriversListMessage(userDate)
						if err != nil {
							log.Error("failed to get drivers list", slog.Any("error", err))
						}

						//chats := []int{alphaTestChatId, botAdminId}
						chats := []int{botAdminId}
						for _, chat := range chats {
							messageToUser = "Открываем конкурс прогнозов! Укажите номера гонщиков, которые по вашему мнению займут первые 3 места по итогам будущей гонки. \n\nОтветьте на ЭТО сообщение в формате: №_топ1 №_топ2 №_топ3\nили напишите\n'/мойпрогноз №_топ1 №_топ2 №_топ3'\n\n\n Пример ответа: '23 17 29'"
							msgResp, err := vk.sendAndLog(log, messageToUser, chat, nil, nil, nil, "predictionStart")
							if err != nil {
								continue
							}

							vk.predictionMessageIDs[chat] = msgResp[0].ConversationMessageID
							log.Info("Prediction started in chat", slog.Int("chat_id", chat), slog.Int("message_id", msgResp[0].ConversationMessageID))

							_, err = vk.sendAndLog(log, driversMessage, chat, nil, nil, nil, "predictionDriversList")
							if err != nil {
								continue
							}
						}
						log.Info("Prediction started globally", slog.String("race_id", raceId))
					}
				}

			case commandPredictionUser:
				if vk.predictionActive {

					pred, err := savePrediction(strings.TrimPrefix(messageText, "/мойпрогноз "))
					if err != nil {
						messageToUser = "Неверный формат сообщения! Повторите попытку и укажите номера гонщиков, которые на ваш взгляд займут первые 3 места, в формате:\n\n/мойпрогноз №_топ1 №_топ2 №_топ3"
						vk.sendAndLog(log, messageToUser, obj.Message.PeerID, nil, nil, nil, "predictionError")
						return
					}
					vk.predictionMessageIDs[obj.Message.FromID] = obj.Message.ConversationMessageID

					messageToUser = fmt.Sprintf("Ваш прогноз принят: 1. №%d, 2. №%d, 3. №%d", pred.first, pred.second, pred.third)
					resp, err := vk.sendAndLog(log, messageToUser, obj.Message.PeerID, nil, nil, nil, "predictionConfirm")
					if err == nil && len(resp) > 0 {
						log.Info("Prediction recorded", slog.Int("user_id", obj.Message.FromID), slog.Any("prediction", pred), slog.Int("message_id", resp[0].MessageID))
					}
				}

			case commandStartCheckStream, commandEndCheckStream:
				ticker := time.NewTicker(5 * time.Minute)
				lastVideo, err := getLastVideos(myUsrVk, 1)
				if err != nil {
					log.Error("failed to get last videos", slog.Any("error", err))
				}
				lastStreamId = lastVideo[0].ID
				if command == commandStartCheckStream {
					vk.sendAndLog(log, "Команда принята!", obj.Message.PeerID, nil, nil, nil, "checkStreamStart")
					log.Info("Start video check")
					go checkLastStream(quit, ticker, log, vk, &myUsrVk, obj)
				}

				if command == commandEndCheckStream {
					ticker.Stop()
					quit <- true
					vk.sendAndLog(log, "Команда принята!", obj.Message.PeerID, nil, nil, nil, "checkStreamEnd")
				}

			default:
				log.Info("Команда в сообщении не распознана", slog.String("text", obj.Message.Text))

				if vk.predictionActive && obj.Message.ReplyMessage != nil && obj.Message.ReplyMessage.ConversationMessageID == vk.predictionMessageIDs[obj.Message.PeerID] {

					pred, err := parsePrediction(messageText)
					if err != nil {
						messageToUser = "Неверный формат сообщения! Повторите попытку и укажите только номера гонщиков, которые на ваш взгляд займут первые 3 места."
						vk.sendAndLog(log, messageToUser, obj.Message.PeerID, nil, nil, nil, "predictionParseError")
						return
					}
					vk.predictionMessageIDs[obj.Message.FromID] = obj.Message.ConversationMessageID
					messageToUser = fmt.Sprintf("Ваш прогноз принят: 1. №%d, 2. №%d, 3. №%d", pred.first, pred.second, pred.third)

					resp, err := vk.sendAndLog(log, messageToUser, obj.Message.PeerID, nil, nil, nil, "predictionReplyConfirm")
					if err == nil && len(resp) > 0 {
						log.Info("Prediction recorded", slog.Int("user_id", obj.Message.FromID), slog.Any("prediction", pred), slog.Int("message_id", resp[0].MessageID))
					}
					return
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
				log.Error("failed to make keyboard", slog.Any("error", err))
				break
			}

			jsKb, err := json.Marshal(newKeyboard)
			if err != nil {
				log.Error("failed to marshal keyboard", slog.Any("error", err))
			}
			strKb := string(jsKb)

			msgResp, err := vk.sendAndLog(log, "Обновление", obj.PeerID, &strKb, nil, nil, fmt.Sprintf("gpListPage_%d", numPage))
			if err != nil {
				break
			}

			evResp, err := sendEventMessageToUser(vk.lp.VK, obj.PeerID, obj.EventID, obj.UserID)
			if err != nil {
				log.Error("failed to send event answer", slog.Int("peer_id", obj.PeerID), slog.Any("error", err))
			}
			log.Info("Event sent", slog.Int("response", evResp))

			err = deleteMessages(vk.lp.VK, []int{msgResp[0].ConversationMessageID}, obj.PeerID, true)
			if err != nil {
				log.Error("failed to delete messages", slog.Any("error", err))
			}

		case commandGpInfo:

			timeNow := time.Now()
			number := strings.Split(*payloadCommand, "_")

			curRace, err := vk.eventService.GetGPInfoCarousel(timeNow, number[1])
			if err != nil {
				log.Error("failed to get GP info carousel", slog.Any("error", err))
			}

			vk.sendAndLog(log, "Информация о гран-при:", obj.PeerID, nil, &curRace, nil, "gpInfo")

			evResp, err := sendEventMessageToUser(vk.lp.VK, obj.PeerID, obj.EventID, obj.UserID)
			if err != nil {
				log.Error("failed to send event answer", slog.Int("peer_id", obj.PeerID), slog.Any("error", err))
			}
			log.Info("Event sent", slog.Int("response", evResp))

		}
	})
}

func sendMessageToUser(messageToUser string, peerID int, vk *api.VK, keyboard, template, attachment *string) (api.MessagesSendUserIDsResponse, error) {
	b := params.NewMessagesSendBuilder()
	b.Message(messageToUser)
	b.RandomID(0)
	//b.PeerID(peerID)
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
	//slog.Info("Message-answer sended", slog.Any("id", msgId))
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
	//slog.Info("Event sent", slog.Int("response", resp))
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
	} else {
		return nil, nil
	}
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

func checkLastStream(quit <-chan bool, ticker *time.Ticker, log *slog.Logger, vk *VkAPI, myUsrVk *MyVk, obj events.MessageNewObject) {
	for {
		select {
		case <-quit:
			ticker.Stop()
			log.Info("End video check")
			return
		case t := <-ticker.C:
			log.Info("Video check", slog.String("time", t.UTC().String()))
			lastVideo, err := getLastVideos(*myUsrVk, 2)
			if err != nil {
				log.Error(err.Error())
				ticker.Stop()
				log.Info("End video check")
				_, err := sendMessageToUser("Ошибка получения новых видео. Перезапустите отслеживание.", botAdminId, vk.lp.VK, nil, nil, nil)
				if err != nil {
					log.Error("Error with sending message-answer to command `checkStream` to user", slog.Int("peer_id", obj.Message.PeerID), slog.Any("error", err))
				}
				return
			}
			log.Info("Video id", slog.Int("ID", lastVideo[0].ID))

			if lastVideo[0].ID != lastStreamId {
				if lastVideo[0].Live && lastVideo[0].LiveStatus == "started" {
					lastStreamId = lastVideo[0].ID
					streamLink := fmt.Sprintf("video%d_%d", f1memesId, lastStreamId)
					/*if err != nil {
						log.Error(err.Error())
					}*/

					messageToUser := fmt.Sprintf("'F1 Memes TV' начали трансляцию '%s'!\n", lastVideo[0].Title)
					resp, err := sendMessageToUser(messageToUser, f1memesChatId, vk.lp.VK, nil, nil, &streamLink)
					if err != nil {
						log.Error("Error with sending message-answer to command `checkStream` to user", slog.Int("peer_id", obj.Message.PeerID), slog.Any("error", err))
					}

					log.Info("Video link", slog.String("video_id", streamLink))
					log.Info("Message sent", slog.Group("response", slog.Int("peer_id", resp[0].PeerID), slog.Int("message_id", resp[0].MessageID), slog.Int("cm_id", resp[0].ConversationMessageID)))

				}

			}
		}
	}
}

func parsePrediction(text string) (Prediction, error) {
	parts := strings.Fields(text)
	if len(parts) != 3 {
		return Prediction{}, fmt.Errorf("error in parsing prediction")
	}
	top1, err := strconv.ParseUint(parts[0], 10, 8)
	if err != nil {
		return Prediction{}, fmt.Errorf("Top_1 position is not number")
	}

	top2, err := strconv.ParseUint(parts[1], 10, 8)
	if err != nil {
		return Prediction{}, fmt.Errorf("Top_2 position is not number")
	}

	top3, err := strconv.ParseUint(parts[2], 10, 8)
	if err != nil {
		return Prediction{}, fmt.Errorf("Top_3 position is not number")
	}

	return Prediction{
		first:  uint8(top1),
		second: uint8(top2),
		third:  uint8(top3),
	}, nil
}

func savePrediction(messageText string) (*Prediction, error) {
	pred, err := parsePrediction(messageText)
	if err != nil {
		return &Prediction{}, err
	}

	//здесь будет проходить сохранение в БД данных о прогнозе юзера
	return &pred, nil
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
/*
func extractStreamLink(messageText string) string {
	msgParts := strings.Split(messageText, " ")
	link := strings.TrimPrefix(msgParts[1], "https://vk.com/")
	return link
}
*/

/*
func checkStreamCommand(id int, command command) bool {
	if ((id == f1memesStreamer) || (id == botAdminId)) && ((command == commandStartCheckStream) || (command == commandEndCheckStream)) {
		return true
	}
	return false
}

*/

package vk

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
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
	GetDriverStandingsMessage(userDate time.Time) string
	GetCalendarMessage(year int) string
	GetNextRaceMessage(userDate time.Time, userTimestamp int) string
	GetConstructorStandingsMessage(uerDate time.Time) string
	GetRaceResultsMessage(userDate time.Time, raceId string) string
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

func (vk *VkAPI) Run() {
	vk.messageHandler()
	vk.eventHandler()

	log.Println("Start longpoll")
	if err := vk.lp.Run(); err != nil {
		log.Fatal(err)
	}
}

func (vk *VkAPI) messageHandler() {
	vk.lp.MessageNew(func(_ context.Context, obj events.MessageNewObject) {
		log.Printf("From id %d: %s", obj.Message.PeerID, obj.Message.Text)

		var messageToUser string
		var command command
		var raceId string

		userTimestamp := obj.Message.Date
		userDate := time.Unix(int64(userTimestamp), 0)

		messageText := strings.ToLower(obj.Message.Text)

		textPayload, err := extractCommand(obj.Message.Payload)
		if err != nil {
			log.Printf("Error reading payload: %v", err)
		}

		if textPayload != nil {
			command = getCommand(*textPayload)
			raceId = (strings.Split(*textPayload, "_"))[1]

			switch command {

			case commandRaceRes:
				messageToUser = vk.messageService.GetRaceResultsMessage(userDate, raceId)
				err := sendMessageToUser(messageToUser, obj.Message.PeerID, vk.lp.VK, nil, nil, nil)
				if err != nil {
					log.Printf("Error with sending message-answer to command `commandRaceRes` to user %d: %s", obj.Message.PeerID, err)
				}

			case commandQualRes:
				messageToUser := vk.messageService.GetQualifyingResultsMessage(userDate, raceId)
				err := sendMessageToUser(messageToUser, obj.Message.PeerID, vk.lp.VK, nil, nil, nil)
				if err != nil {
					log.Printf("Error with sending message-answer to command `commandQualRes` to user %d: %s", obj.Message.PeerID, err)
				}

			case commandSprRes:
				messageToUser = vk.messageService.GetSprintResultsMessage(userDate, raceId)
				err := sendMessageToUser(messageToUser, obj.Message.PeerID, vk.lp.VK, nil, nil, nil)
				if err != nil {
					log.Printf("Error with sending message-answer to command `commandSprRes` to user %d: %v", obj.Message.PeerID, err)
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
					log.Printf("Error with sending message-answer to command `checkStream` to user %d: %v", obj.Message.PeerID, err)
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
						log.Printf("Error with sending message-answer to command `Hello` to user %d: %v", obj.Message.PeerID, err)
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
						log.Printf("Error with sending message-answer to command `commandHelp` to user %d: %v", obj.Message.PeerID, err)
					}

				case commandDrSt:
					messageToUser = vk.messageService.GetDriverStandingsMessage(userDate)
					err := sendMessageToUser(messageToUser, obj.Message.PeerID, vk.lp.VK, nil, nil, nil)
					if err != nil {
						log.Printf("Error with sending message-answer to command `commandDrSt` to user %d: %v", obj.Message.PeerID, err)
					}

				case commandCld:
					messageToUser = vk.messageService.GetCalendarMessage(userDate.Year())
					err := sendMessageToUser(messageToUser, obj.Message.PeerID, vk.lp.VK, nil, nil, nil)
					if err != nil {
						log.Printf("Error with sending message-answer to command `commandCld` to user %d: %v", obj.Message.PeerID, err)
					}

				case commandNxRc:
					messageToUser = vk.messageService.GetNextRaceMessage(userDate, userTimestamp)
					err := sendMessageToUser(messageToUser, obj.Message.PeerID, vk.lp.VK, nil, nil, nil)
					if err != nil {
						log.Printf("Error with sending message-answer to command `commandNxRc` to user %d: %v", obj.Message.PeerID, err)
					}

				case commandConsStFull:
					messageToUser = vk.messageService.GetConstructorStandingsMessage(userDate)
					err := sendMessageToUser(messageToUser, obj.Message.PeerID, vk.lp.VK, nil, nil, nil)
					if err != nil {
						log.Printf("Error with sending message-answer to command `commandConsStFull` to user %d: %v", obj.Message.PeerID, err)
					}

				case commandConsSt:
					messageToUser = vk.messageService.GetConstructorStandingsMessage(userDate)
					err := sendMessageToUser(messageToUser, obj.Message.PeerID, vk.lp.VK, nil, nil, nil)
					if err != nil {
						log.Printf("Error with sending message-answer to command `commandConsSt` to user %d: %v", obj.Message.PeerID, err)
					}

				case commandLstRc:
					messageToUser = vk.messageService.GetRaceResultsMessage(userDate, raceId)
					err := sendMessageToUser(messageToUser, obj.Message.PeerID, vk.lp.VK, nil, nil, nil)
					if err != nil {
						log.Printf("Error with sending message-answer to command `commandLstRc` to user %d: %v", obj.Message.PeerID, err)
					}

				case commandLstGP:
					crsl := vk.messageService.GetGPInfoCarousel(userDate, raceId)
					err := sendMessageToUser("Информация о гран-при:", obj.Message.PeerID, vk.lp.VK, nil, &crsl, nil)
					if err != nil {
						log.Printf("Error with sending message-answer to command `commandLstGP` to user %d: %v", obj.Message.PeerID, err)
					}

				case commandGPs:
					kb := vk.messageService.GetGPKeyboard()
					err := sendMessageToUser("Этапы F1:", obj.Message.PeerID, vk.lp.VK, &kb, nil, nil)
					if err != nil {
						log.Printf("Error with sending message-answer to command `commandGPs` to user %d: %v", obj.Message.PeerID, err)
					}

				case commandDaysAfterRace:
					messageToUser := vk.messageService.GetCountDaysAfterRaceMessage(userDate, raceId)
					err := sendMessageToUser(messageToUser, obj.Message.PeerID, vk.lp.VK, nil, nil, nil)
					if err != nil {
						log.Printf("Error with sending message-answer to command `commandDaysAfterRace` to user %d: %v", obj.Message.PeerID, err)
					}

				case commandLstQual:
					messageToUser := vk.messageService.GetQualifyingResultsMessage(userDate, raceId)
					err := sendMessageToUser(messageToUser, obj.Message.PeerID, vk.lp.VK, nil, nil, nil)
					if err != nil {
						log.Printf("Error with sending message-answer to command `commandLstQual` to user %d: %v", obj.Message.PeerID, err)
					}
				default:
					log.Printf("Команда в сообщении `%s` не распознана", obj.Message.Text)

				}
			}

		}
	})
}

func (vk *VkAPI) eventHandler() {
	vk.lp.MessageEvent(func(_ context.Context, obj events.MessageEventObject) {

		log.Printf("From id %d: %s", obj.PeerID, obj.Payload)

		payloadCommand, err := extractCommand(string(obj.Payload))
		if err != nil {
			log.Printf("Error reading payload: %v", err)
		}
		command := getEventCommand(*payloadCommand)

		switch command {

		case commandGpList1:
			newKeyboard, err := makeKeyboard(2, 4, 1, 22, false)
			if err != nil {
				log.Printf("Error making keyboard: %v", err)
			}

			jsKb, err := json.Marshal(newKeyboard)
			if err != nil {
				log.Printf("Error marshall keyboard: %v", err)
			}
			strKb := string(jsKb)

			messageToUser := "Обновление"

			err = sendMessageToUser(messageToUser, obj.PeerID, vk.lp.VK, &strKb, nil, nil)
			if err != nil {
				log.Printf("Error with sending message-answer to command `commandGpList1` to user %d: %v", obj.PeerID, err)
			}

			err = sendEventMessageToUser(vk.lp.VK, obj.PeerID, obj.EventID, obj.UserID)
			if err != nil {
				log.Printf("Error with sending event-answer to command `commandGpList1` to user %d: %v", obj.PeerID, err)
			}

		case commandGpList2:
			newKeyboard, err := makeKeyboard(2, 4, 2, 22, false)
			if err != nil {
				log.Printf("Error making keyboard: %v", err)
			}

			jsKb, err := json.Marshal(newKeyboard)
			if err != nil {
				log.Printf("Error marshall keyboard: %v", err)
			}
			strKb := string(jsKb)

			messageToUser := "Обновление"

			err = sendMessageToUser(messageToUser, obj.PeerID, vk.lp.VK, &strKb, nil, nil)
			if err != nil {
				log.Printf("Error with sending message-answer to command `commandGpList2` to user %d: %v", obj.PeerID, err)
			}

			err = sendEventMessageToUser(vk.lp.VK, obj.PeerID, obj.EventID, obj.UserID)
			if err != nil {
				log.Printf("Error with sending event-answer to command `commandGpList2` to user %d: %v", obj.PeerID, err)
			}

		case commandGpList3:
			newKeyboard, err := makeKeyboard(2, 4, 3, 22, false)
			if err != nil {
				log.Printf("Error making keyboard: %v", err)
			}

			jsKb, err := json.Marshal(newKeyboard)
			if err != nil {
				log.Printf("Error marshall keyboard: %v", err)
			}
			strKb := string(jsKb)

			messageToUser := "Обновление"

			err = sendMessageToUser(messageToUser, obj.PeerID, vk.lp.VK, &strKb, nil, nil)
			if err != nil {
				log.Printf("Error with sending message-answer to command `commandGpList3` to user %d: %v", obj.PeerID, err)
			}

			err = sendEventMessageToUser(vk.lp.VK, obj.PeerID, obj.EventID, obj.UserID)
			if err != nil {
				log.Printf("Error with sending event-answer to command `commandGpList3` to user %d: %v", obj.PeerID, err)
			}

		case commandGpInfo:

			timeNow := time.Now()
			number := strings.Split(*payloadCommand, "_")

			curRace := vk.eventService.GetGPInfoCarousel(timeNow, number[1])

			fmt.Println(curRace)

			messageToUser := "Информация о гран-при:"

			err := sendMessageToUser(messageToUser, obj.PeerID, vk.lp.VK, nil, &curRace, nil)
			if err != nil {
				log.Printf("Error with sending message-answer to command `commandGpInfo` to user %d: %v", obj.PeerID, err)
			}
			err = sendEventMessageToUser(vk.lp.VK, obj.PeerID, obj.EventID, obj.UserID)
			if err != nil {
				log.Printf("Error with sending event-answer to command `commandGpInfo` to user %d: %v", obj.PeerID, err)
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
	fmt.Printf("Message-answer ID: %d\n", msgId)
	return nil
}

func sendEventMessageToUser(vk *api.VK, peerID int, eventID string, userID int) error {
	prms := params.NewMessagesSendMessageEventAnswerBuilder()
	prms.PeerID(peerID)
	prms.EventID(eventID)
	prms.UserID(userID)

	evId, err := vk.MessagesSendMessageEventAnswer(prms.Params)
	fmt.Println(evId)
	if err != nil {
		return fmt.Errorf("error sending message to user: %w", err)
	}

	return nil
}

func extractCommand(payload string) (*string, error) {
	var pl Payload
	if payload != "" {
		err := json.Unmarshal([]byte(payload), &pl)
		if err != nil {
			return nil, fmt.Errorf("error unmarshal command in payload message: %w", err)
		}
		log.Printf("Command from paylpad: %s\n", pl.Command)
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
	//link := fmt.Sprintf("[%s|VK]", strings.TrimPrefix(msgParts[1], "https://vk.com/"))
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

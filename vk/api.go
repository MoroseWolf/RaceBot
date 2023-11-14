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
	f1memesId       = 2000000005
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
		return nil, fmt.Errorf("Error groups get by id: %w", err)
	}

	lp, err := longpoll.NewLongPoll(vk, group[0].ID)
	if err != nil {
		return nil, fmt.Errorf("Error creating new log pool: %w", err)
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

		textPayload := extractCommand(obj.Message.Payload)
		if textPayload != nil {
			command = getCommand(*textPayload)
			raceId = (strings.Split(*textPayload, "_"))[1]

			switch command {

			case commandRaceRes:
				messageToUser = vk.messageService.GetRaceResultsMessage(userDate, raceId)
				sendMessageToUser(messageToUser, obj.Message.PeerID, vk.lp.VK, nil, nil)

			case commandQualRes:
				messageToUser := vk.messageService.GetQualifyingResultsMessage(userDate, raceId)
				sendMessageToUser(messageToUser, obj.Message.PeerID, vk.lp.VK, nil, nil)

			case commandSprRes:
				messageToUser = vk.messageService.GetSprintResultsMessage(userDate, raceId)
				sendMessageToUser(messageToUser, obj.Message.PeerID, vk.lp.VK, nil, nil)
			}
		} else {
			command = getCommand(messageText)
			raceId = "last"

			if obj.Message.PeerID == f1memesStreamer && command == commandStream {
				messageToUser = "Трансляция 'F1 Memes TV' началась! Смотри в Telegram t.me/f1memestv и в VK vk.com/f1memestv."
				sendMessageToUser(messageToUser, f1memesId, vk.lp.VK, nil, nil)

			} else {

				switch command {

				case commandHello:
					messageToUser =
						`Привет! Я бот, который делится информацией про F1 :)
					Пока что я могу сказать тебе информацию только о текущем сезоне (но всё ещё впереди).
					Для того чтобы подробнее познакомиться с моими возможностями напиши мне "Что умеешь?". 
					
					Приятного пользования :)`
					sendMessageToUser(messageToUser, obj.Message.PeerID, vk.lp.VK, nil, nil)

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
					sendMessageToUser(messageToUser, obj.Message.PeerID, vk.lp.VK, nil, nil)

				case commandDrSt:
					messageToUser = vk.messageService.GetDriverStandingsMessage(userDate)
					sendMessageToUser(messageToUser, obj.Message.PeerID, vk.lp.VK, nil, nil)

				case commandCld:
					messageToUser = vk.messageService.GetCalendarMessage(userDate.Year())
					sendMessageToUser(messageToUser, obj.Message.PeerID, vk.lp.VK, nil, nil)

				case commandNxRc:
					messageToUser = vk.messageService.GetNextRaceMessage(userDate, userTimestamp)
					sendMessageToUser(messageToUser, obj.Message.PeerID, vk.lp.VK, nil, nil)

				case commandConsStFull:
					messageToUser = vk.messageService.GetConstructorStandingsMessage(userDate)
					sendMessageToUser(messageToUser, obj.Message.PeerID, vk.lp.VK, nil, nil)

				case commandConsSt:
					messageToUser = vk.messageService.GetConstructorStandingsMessage(userDate)
					sendMessageToUser(messageToUser, obj.Message.PeerID, vk.lp.VK, nil, nil)

				case commandLstRc:
					messageToUser = vk.messageService.GetRaceResultsMessage(userDate, raceId)
					sendMessageToUser(messageToUser, obj.Message.PeerID, vk.lp.VK, nil, nil)

				case commandLstGP:
					crsl := vk.messageService.GetGPInfoCarousel(userDate, raceId)
					sendMessageToUser("Информация о гран-при:", obj.Message.PeerID, vk.lp.VK, nil, &crsl)

				case commandGPs:
					kb := vk.messageService.GetGPKeyboard()
					sendMessageToUser("Этапы F1:", obj.Message.PeerID, vk.lp.VK, &kb, nil)

				case commandDaysAfterRace:
					messageToUser := vk.messageService.GetCountDaysAfterRaceMessage(userDate, raceId)
					sendMessageToUser(messageToUser, obj.Message.PeerID, vk.lp.VK, nil, nil)

				case commandLstQual:
					messageToUser := vk.messageService.GetQualifyingResultsMessage(userDate, raceId)
					sendMessageToUser(messageToUser, obj.Message.PeerID, vk.lp.VK, nil, nil)
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

		payloadCommand := extractCommand(string(obj.Payload))
		command := getEventCommand(*payloadCommand)

		switch command {

		case commandGpList1:
			newKeyboard, err := makeKeyboard(2, 4, 1, 22, false)
			if err != nil {
				fmt.Errorf("Error making keyboard: %w", err)
			}

			jsKb, err := json.Marshal(newKeyboard)
			if err != nil {
				fmt.Errorf("Error marshall keyboard: %w", err)
			}
			strKb := string(jsKb)

			messageToUser := "Обновление"

			sendMessageToUser(messageToUser, obj.PeerID, vk.lp.VK, &strKb, nil)
			sendEventMessageToUser(vk.lp.VK, obj.PeerID, obj.EventID, obj.UserID)

		case commandGpList2:
			newKeyboard, err := makeKeyboard(2, 4, 2, 22, false)
			if err != nil {
				fmt.Errorf("Error making keyboard: %w", err)
			}

			jsKb, err := json.Marshal(newKeyboard)
			if err != nil {
				fmt.Errorf("Error marshall keyboard: %w", err)
			}
			strKb := string(jsKb)

			messageToUser := "Обновление"

			sendMessageToUser(messageToUser, obj.PeerID, vk.lp.VK, &strKb, nil)
			sendEventMessageToUser(vk.lp.VK, obj.PeerID, obj.EventID, obj.UserID)

		case commandGpList3:
			newKeyboard, err := makeKeyboard(2, 4, 3, 22, false)
			if err != nil {
				fmt.Errorf("Error making keyboard: %w", err)
			}

			jsKb, err := json.Marshal(newKeyboard)
			if err != nil {
				fmt.Errorf("Error marshall keyboard: %w", err)
			}
			strKb := string(jsKb)

			messageToUser := "Обновление"

			sendMessageToUser(messageToUser, obj.PeerID, vk.lp.VK, &strKb, nil)
			sendEventMessageToUser(vk.lp.VK, obj.PeerID, obj.EventID, obj.UserID)

		case commandGpInfo:

			timeNow := time.Now()
			number := strings.Split(*payloadCommand, "_")

			curRace := vk.eventService.GetGPInfoCarousel(timeNow, number[1])

			fmt.Println(curRace)

			messageToUser := "Информация о гран-при:"

			sendMessageToUser(messageToUser, obj.PeerID, vk.lp.VK, nil, &curRace)
			sendEventMessageToUser(vk.lp.VK, obj.PeerID, obj.EventID, obj.UserID)

		}
	})
}

func sendMessageToUser(messageToUser string, peerID int, vk *api.VK, keyboard, template *string) {
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

	msgId, err := vk.MessagesSend(b.Params)
	if err != nil {
		fmt.Errorf("Error sending message to user: %w", err)
	}

	fmt.Println(msgId)
}

func sendEventMessageToUser(vk *api.VK, peerID int, eventID string, userID int) {
	prms := params.NewMessagesSendMessageEventAnswerBuilder()
	prms.PeerID(peerID)
	prms.EventID(eventID)
	prms.UserID(userID)

	evId, err := vk.MessagesSendMessageEventAnswer(prms.Params)
	fmt.Println(evId)
	if err != nil {
		fmt.Errorf("Error sending message to user: %w", err)
	}
}

func extractCommand(payload string) *string {
	var pl Payload
	if payload != "" {
		err := json.Unmarshal([]byte(payload), &pl)
		if err != nil {
			fmt.Errorf("Error reading command in payload message: %w", err)
		}
		log.Printf("Command from paylpad: %s \n", pl.Command)
		return &pl.Command
	} else {
		return nil
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
		return Kb{}, fmt.Errorf("С заданными параметрами невозможно отобразить клавиатуру. Для количества элементов %d не существует %d-ой страницы клавиатуры при %d кнопках", countEl, numPage, sizeKb)
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

package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"regexp"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/SevereCloud/vksdk/v2/api"
	"github.com/SevereCloud/vksdk/v2/api/params"
	"github.com/SevereCloud/vksdk/v2/events"
	"github.com/SevereCloud/vksdk/v2/longpoll-bot"
	"github.com/joho/godotenv"
)

var months = map[string]string{
	"01": "января",
	"02": "февраля",
	"03": "марта",
	"04": "апреля",
	"05": "мая",
	"06": "июня",
	"07": "июля",
	"08": "августа",
	"09": "сентября",
	"10": "октября",
	"11": "ноября",
	"12": "декабря",
}

func init() {
	if err := godotenv.Load("../.env"); err != nil {
		log.Print("No .env file found")
	}
}

func main() {

	token, _ := os.LookupEnv("RACEVK_BOT")
	vk := api.NewVK(token)

	group, err := vk.GroupsGetByID(api.Params{})
	if err != nil {
		log.Fatal(err)
	}

	lp, err := longpoll.NewLongPoll(vk, group[0].ID)
	if err != nil {
		log.Fatal(err)
	}

	lp.MessageNew(func(_ context.Context, obj events.MessageNewObject) {
		log.Printf("From id %d: %s", obj.Message.PeerID, obj.Message.Text)

		var messageToUser string
		var adminStream bool
		var playload Playload

		userTimestamp := obj.Message.Date
		userDate := time.Unix(int64(userTimestamp), 0)

		messageText := strings.ToLower(obj.Message.Text)

		if obj.Message.Payload != "" {
			err := json.Unmarshal([]byte(obj.Message.Payload), &playload)
			if err != nil {
				log.Fatalln(err)
			}
		}
		log.Printf("Command from message: %s \n", playload.Command)

		b := params.NewMessagesSendBuilder()

		matchedDrSt, _ := regexp.MatchString(`личн.*зач[её]т`, messageText)
		matchedCld, _ := regexp.MatchString(`календар.*сезона`, messageText)
		matchedNxRc, _ := regexp.MatchString(`следующ.*гонк`, messageText)
		matchedConsStFull, _ := regexp.MatchString(`куб.*конструктор`, messageText)
		matchedConsSt, _ := regexp.MatchString(`кк`, messageText)
		matchedLstRc, _ := regexp.MatchString(`результат.?\sгонк`, messageText)
		matchedLstQual, _ := regexp.MatchString(`результат.?\sквалы`, messageText)
		matchedLstSpr, _ := regexp.MatchString(`результат.?\sспринта`, messageText)
		matchedHelp, _ := regexp.MatchString(`что умеешь`, messageText)
		matchedHello, _ := regexp.MatchString(`начать`, messageText)
		matchedDaysAfterRace, _ := regexp.MatchString(`дней без формулы|F1`, messageText)
		matchedStream, _ := regexp.MatchString(`старт стрим`, messageText)
		matchedLstGP, _ := regexp.MatchString(`ласт гп`, messageText)
		matchedRaceRes, _ := regexp.MatchString(`raceRes_\d{1,2}`, playload.Command)
		matchedQualRes, _ := regexp.MatchString(`qualRes_\d{1,2}`, playload.Command)
		matchedSprRes, _ := regexp.MatchString(`sprRes_\d{1,2}`, playload.Command)

		if obj.Message.PeerID == 152819213 && matchedStream {
			adminStream = true
			messageToUser = "Трансляция 'F1 Memes TV' началась! Смотри в Telegram t.me/f1memestv и в VK vk.com/f1memestv."

		} else {

			switch {

			case matchedHello:
				messageToUser =
					`Привет! Я бот, который делится информацией про F1 :)
				Пока что я могу сказать тебе информацию только о текущем сезоне (но всё ещё впереди).
				Для того чтобы подробнее познакомиться с моими возможностями напиши мне "Что умеешь?". 
				
				Приятного пользования :)`

			case matchedHelp:
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

			case matchedDrSt:
				resp, err := http.Get(fmt.Sprintf("http://ergast.com/api/f1/%d/driverStandings.json", userDate.Year()))
				if err != nil {
					log.Fatalln(err)
				}
				defer resp.Body.Close()

				body, err := io.ReadAll(resp.Body)
				if err != nil {
					log.Fatalln(err)
				}

				var object Object
				json.Unmarshal([]byte(body), &object)
				driversTable := object.MRData.StandingsTable.StandingsLists[0].DriverStandings

				messageToUser = fmt.Sprintf("Личный зачёт F1, сезон %d: \n%s", userDate.Year(), driversToString(driversTable))

			case matchedCld:
				resp, err := http.Get(fmt.Sprintf("http://ergast.com/api/f1/%d.json", userDate.Year()))
				if err != nil {
					log.Fatalln(err)
				}
				defer resp.Body.Close()

				body, err := io.ReadAll(resp.Body)
				if err != nil {
					log.Fatalln(err)
				}

				var races Object
				json.Unmarshal([]byte(body), &races)

				messageToUser = fmt.Sprintf("Календарь F1, сезон %d:\n%s", userDate.Year(), racesToString(races.MRData.RaceTable.Races))

			case matchedNxRc:
				resp, err := http.Get(fmt.Sprintf("http://ergast.com/api/f1/%d.json", userDate.Year()))
				if err != nil {
					log.Fatalln(err)
				}
				defer resp.Body.Close()

				body, err := io.ReadAll(resp.Body)
				if err != nil {
					log.Fatalln(err)
				}

				var races Object
				json.Unmarshal([]byte(body), &races)

				isAfter := checkCurrToLastTime(int64(userTimestamp), races.MRData.RaceTable.Races[len(races.MRData.RaceTable.Races)-1])

				if isAfter {
					messageToUser = "Сезон закончился!"
				} else {
					nextRace := findNextRace(int64(userTimestamp), races.MRData.RaceTable.Races)
					messageToUser = fmt.Sprintf("Cледующий гран-при :\n%s", raceFullInfoToString(formatDateTime(nextRace)))
				}

			case matchedConsStFull || matchedConsSt:
				resp, err := http.Get(fmt.Sprintf("http://ergast.com/api/f1/%d/constructorStandings.json", userDate.Year()))
				if err != nil {
					log.Fatalln(err)
				}
				defer resp.Body.Close()

				body, err := io.ReadAll(resp.Body)
				if err != nil {
					log.Fatalln(err)
				}

				var constructors Object
				json.Unmarshal([]byte(body), &constructors)

				messageToUser = fmt.Sprintf("Кубок конструкторов F1, сезон %d:\n%s", userDate.Year(), constructorsToString(constructors.MRData.StandingsTable.StandingsLists[0].ConstructorStandings))

			case matchedLstRc || matchedRaceRes:
				var reqUrl string
				if matchedRaceRes {
					partsMessage := strings.Split(playload.Command, "_")
					reqUrl = fmt.Sprintf("http://ergast.com/api/f1/%d/%s/results.json", userDate.Year(), partsMessage[1])
				} else {
					reqUrl = fmt.Sprintf("http://ergast.com/api/f1/%d/last/results.json", userDate.Year())
				}
				resp, err := http.Get(reqUrl)
				if err != nil {
					log.Fatalln(err)
				}
				defer resp.Body.Close()

				body, err := io.ReadAll(resp.Body)
				if err != nil {
					log.Fatalln(err)
				}

				var lastRace Object
				json.Unmarshal([]byte(body), &lastRace)

				if matchedRaceRes {
					messageToUser = fmt.Sprintf("Результаты %s:\n%s", lastRace.MRData.RaceTable.Races[0].RaceName, raceResultsToString(lastRace.MRData.RaceTable.Races[0]))
				} else {
					messageToUser = fmt.Sprintf("Последняя гонка F1 %s:\n%s", lastRace.MRData.RaceTable.Races[0].RaceName, raceResultsToString(lastRace.MRData.RaceTable.Races[0]))
				}

			case matchedDaysAfterRace:
				resp, err := http.Get(fmt.Sprintf("http://ergast.com/api/f1/%d/last.json", userDate.Year()))
				if err != nil {
					log.Fatalln(err)
				}
				defer resp.Body.Close()

				body, err := io.ReadAll(resp.Body)
				if err != nil {
					log.Fatalln(err)
				}

				var lastRace Object
				json.Unmarshal([]byte(body), &lastRace)

				lastRaceDate, err := time.Parse("2006-01-02 15:04:05Z", fmt.Sprintf("%s %s", lastRace.MRData.RaceTable.Races[0].Date, lastRace.MRData.RaceTable.Races[0].Time))
				if err != nil {
					log.Fatalln(err)
				}
				difference := userDate.Sub(lastRaceDate)

				messageToUser = fmt.Sprintf("Дней без F1 - %d :(\n", int64(difference.Hours()/24))

			case matchedLstGP:
				resp, err := http.Get(fmt.Sprintf("http://ergast.com/api/f1/%d/last.json", userDate.Year()))
				if err != nil {
					log.Fatalln(err)
				}
				defer resp.Body.Close()

				body, err := io.ReadAll(resp.Body)
				if err != nil {
					log.Fatalln(err)
				}

				var temp Object
				json.Unmarshal([]byte(body), &temp)
				curRace := formatDateTime(temp.MRData.RaceTable.Races[0])
				strCrslItem := makeCarouselGPItem(curRace)

				messageToUser = "Информация о гран-при:"
				b.Template(strCrslItem)

			case matchedLstQual || matchedQualRes:
				var reqUrl string
				if matchedQualRes {
					partsMessage := strings.Split(playload.Command, "_")
					reqUrl = fmt.Sprintf("http://ergast.com/api/f1/%d/%s/qualifying.json", userDate.Year(), partsMessage[1])
				} else {
					reqUrl = fmt.Sprintf("http://ergast.com/api/f1/%d/last/qualifying.json", userDate.Year())
				}
				resp, err := http.Get(reqUrl)
				if err != nil {
					log.Fatalln(err)
				}
				defer resp.Body.Close()

				body, err := io.ReadAll(resp.Body)
				if err != nil {
					log.Fatalln(err)
				}

				var temp Object
				json.Unmarshal([]byte(body), &temp)
				qualRace := temp.MRData.RaceTable.Races[0]

				if matchedQualRes {
					messageToUser = fmt.Sprintf("Результаты квалификации %s:\n%s", qualRace.RaceName, qualifyingResultsToString(qualRace))
				} else {
					messageToUser = fmt.Sprintf("Последняя квалификация %s:\n%s", qualRace.RaceName, qualifyingResultsToString(qualRace))
				}

			case matchedLstSpr || matchedSprRes:
				var reqUrl string
				if matchedSprRes {
					partsMessage := strings.Split(playload.Command, "_")
					reqUrl = fmt.Sprintf("http://ergast.com/api/f1/%d/%s/sprint.json", userDate.Year(), partsMessage[1])
				} else {
					reqUrl = fmt.Sprintf("http://ergast.com/api/f1/%d/sprint.json", userDate.Year())
				}
				resp, err := http.Get(reqUrl)
				if err != nil {
					log.Fatalln(err)
				}
				defer resp.Body.Close()

				body, err := io.ReadAll(resp.Body)
				if err != nil {
					log.Fatalln(err)
				}

				var temp Object
				json.Unmarshal([]byte(body), &temp)

				if matchedSprRes {
					sprRace := temp.MRData.RaceTable.Races[0]
					messageToUser = fmt.Sprintf("Результаты спринт-гонки %s:\n%s", sprRace.RaceName, sprintResultsToString(sprRace))
				} else {
					sprRace := temp.MRData.RaceTable.Races[len(temp.MRData.RaceTable.Races)-1]
					messageToUser = fmt.Sprintf("Последняя спринт-гонка %s:\n%s", sprRace.RaceName, sprintResultsToString(sprRace))
				}

			default:
				messageToUser = "Прости, пока что не понимаю тебя. Но я умный и скоро научусь этому!"

			}
		}

		b.Message(messageToUser)
		b.RandomID(0)

		if adminStream {
			b.PeerID(2000000005)
		} else {
			b.PeerID(obj.Message.PeerID)
		}

		respCode, err := vk.MessagesSend(b.Params)
		fmt.Println(respCode)
		if err != nil {
			log.Fatal(err)
		}
	})

	log.Println("Start longpoll")
	if err := lp.Run(); err != nil {
		log.Fatal(err)
	}
}

func racesToString(races []Race) string {

	var countRaces int = len(races)
	racesList := make([]string, countRaces)

	for num, race := range races {
		races[num] = formatDateTime(race)
	}

	for _, race := range races {
		racesList = append(racesList, raceToString(race))
	}

	return strings.Join(racesList, "")
}

func raceToString(race Race) string {
	return fmt.Sprintf("Номер этапа: %s,\nНазвание этапа: %s,\nДата этапа: %s,\nВремя этапа: %s.\n\n",
		race.Round, race.RaceName, race.Date, race.Time)
}

func driversToString(drivers []DriverStandingsItem) string {

	var countDrivers int = len(drivers)
	driversList := make([]string, countDrivers)

	for _, driver := range drivers {
		driversList = append(driversList, driverToString(driver))
	}

	return strings.Join(driversList, "")
}

func driverToString(driver DriverStandingsItem) string {
	return fmt.Sprintf("%2s | %-3s - %-3s \n", driver.PositionText, driver.Driver.Code, driver.Points)
}

func constructorsToString(constructors []ConstructorStandingsItem) string {
	var countConstructors int = len(constructors)
	constructorsList := make([]string, countConstructors)

	for _, constructor := range constructors {
		constructorsList = append(constructorsList, constructorToString(constructor))
	}

	return strings.Join(constructorsList, "")
}

func constructorToString(constructor ConstructorStandingsItem) string {
	return fmt.Sprintf("%2s | %s - %-3s \n", constructor.Position, constructor.Constructor.Name, constructor.Points)
}

func raceResultsToString(race Race) string {
	/*
		var driverInRaceCount = len(race.Results)
		driversList := make([]string, 2)
		driversList := make([]string, driverInRaceCount+1)

		driversList = append(driversList, race.RaceName+":\n")
	*/
	message := new(strings.Builder)

	w := tabwriter.NewWriter(message, 2, 5, 1, ' ', tabwriter.AlignRight)
	for _, position := range race.Results {
		if position.Status == "Finished" {
			if position.Points != "0" {
				fmt.Fprintf(w, "%s |\t%s |\t %s - %s\n", position.Position, position.Driver.Code, position.Time.Time, position.Points)
			} else {
				fmt.Fprintf(w, "%s |\t%s |\t %s\n", position.Position, position.Driver.Code, position.Time.Time)
			}
		} else {
			fmt.Fprintf(w, "%s |\t%s |\t - %s\n", position.Position, position.Driver.Code, position.Status)
		}
	}

	w.Flush()
	/*
		driversList = append(driversList, message.String())

			for _, position := range race.Results {
				driversList = append(driversList, raceResultToString(position))
			}
	*/
	//return strings.Join(driversList, "")
	return message.String()
}

func raceResultToString(position Result) string {
	return fmt.Sprintf("%2s | %-3s | %-11s - %-3s \n", position.Position, position.Driver.Code, position.Time.Time, position.Points)
}

func formatDateTime(race Race) Race {

	tzone, err := time.LoadLocation("Europe/Moscow")
	if err != nil {
		log.Fatalln(err)
	}

	raceDate := parseStringToTime(race.Date, race.Time)
	fPracticeDate := parseStringToTime(race.FirstPractice.Date, race.FirstPractice.Time)
	sPracticeDate := parseStringToTime(race.SecondPractice.Date, race.SecondPractice.Time)
	qualDate := parseStringToTime(race.Qualifying.Date, race.Qualifying.Time)

	race.Date = ruMonth(raceDate.Format("2006-01-02"))
	race.Time = raceDate.In(tzone).Format("15:04")

	race.FirstPractice.Date = ruMonth(fPracticeDate.Format("2006-01-02"))
	race.FirstPractice.Time = fPracticeDate.In(tzone).Format("15:04")

	race.SecondPractice.Date = ruMonth(sPracticeDate.Format("2006-01-02"))
	race.SecondPractice.Time = sPracticeDate.In(tzone).Format("15:04")

	race.Qualifying.Date = ruMonth(qualDate.Format("2006-01-02"))
	race.Qualifying.Time = qualDate.In(tzone).Format("15:04")

	if len(race.Sprint.Date) > 0 {
		sprDate := parseStringToTime(race.Sprint.Date, race.Sprint.Time)
		race.Sprint.Date = ruMonth(sprDate.Format("2006-01-02"))
		race.Sprint.Time = sprDate.In(tzone).Format("15:04")
	} else {
		tPracticeDate := parseStringToTime(race.ThirdPractice.Date, race.ThirdPractice.Time)
		race.ThirdPractice.Date = ruMonth(tPracticeDate.Format("2006-01-02"))
		race.ThirdPractice.Time = tPracticeDate.In(tzone).Format("15:04")
	}

	return race
}

func parseStringToTime(dateRace string, timeRace string) time.Time {
	tempDateTime, err := time.Parse("2006-01-02 15:04:05Z", fmt.Sprintf("%s %s", dateRace, timeRace))
	if err != nil {
		log.Fatalln(err)
	}
	return tempDateTime
}

func ruMonth(date string) string {

	partsDate := strings.Split(date, "-")
	for key, value := range months {
		if key == partsDate[1] {
			partsDate[1] = value
		}
	}

	return strings.Join([]string{partsDate[2], partsDate[1], partsDate[0]}, " ")
}

func checkCurrToLastTime(messageDate int64, race Race) bool {
	lastRace, err := time.Parse("2006-01-02 15:04:05Z", fmt.Sprintf("%s %s", race.Date, race.Time))
	if err != nil {
		log.Fatalln(err)
	}

	if messageDate >= int64(lastRace.Unix()) {
		return true
	} else {
		return false
	}
}

func findNextRace(messageDate int64, races []Race) Race {

	userDate := time.Unix(messageDate, 0)
	var numRace int

	for num, race := range races {

		tempDateTime, err := time.Parse("2006-01-02 15:04:05Z", fmt.Sprintf("%s %s", race.Date, race.Time))
		if err != nil {
			log.Fatalln(err)
		}

		if tempDateTime.After(userDate) {
			numRace = num
			break
		}
	}

	return races[numRace]
}

func raceFullInfoToString(race Race) string {
	if len(race.Sprint.Date) > 0 {
		return fmt.Sprintf("Номер этапа: %s,\nНазвание этапа: %s,\nВремя гонки: %s,\n\nПрактика: %s,\nКвалификация: %s,\n\nКвалификация спринта: %s, \nСпринт: %s.\n\n",
			race.Round, race.RaceName, race.Date+" "+race.Time, race.FirstPractice.Date+" "+race.FirstPractice.Time, race.Qualifying.Date+" "+race.Qualifying.Time, race.SecondPractice.Date+" "+race.SecondPractice.Time, race.Sprint.Date+" "+race.Sprint.Time)
	} else {
		return fmt.Sprintf("Номер этапа: %s,\nНазвание этапа: %s,\nВремя гонки: %s,\n\nПервая практика: %s,\nВторая практика: %s, \nТретья практика: %s,\nКвалификация: %s.\n",
			race.Round, race.RaceName, race.Date+" "+race.Time, race.FirstPractice.Date+" "+race.FirstPractice.Time, race.SecondPractice.Date+" "+race.SecondPractice.Time, race.ThirdPractice.Date+" "+race.ThirdPractice.Time, race.Qualifying.Date+" "+race.Qualifying.Time)
	}
}

func qualifyingResultsToString(race Race) string {

	message := new(strings.Builder)

	w := tabwriter.NewWriter(message, 2, 5, 1, ' ', tabwriter.AlignRight)
	for _, qualPosition := range race.QualifyingResults {
		if qualPosition.Q3 != "" {
			//fmt.Fprintf(w, "%s |\t%s |\t\n Q1: %s -- Q2: %s -- Q3: %s \n", qualPosition.Position, qualPosition.Driver.Code, qualPosition.Q1, qualPosition.Q2, qualPosition.Q3)
			fmt.Fprintf(w, "%s |\t%s |\t\n Q1: %s\n Q2: %s\n Q3: %s\n\n", qualPosition.Position, qualPosition.Driver.Code, qualPosition.Q1, qualPosition.Q2, qualPosition.Q3)

		} else {
			if qualPosition.Q2 != "" {
				fmt.Fprintf(w, "%s |\t%s |\t\n Q1: %s\n Q2: %s\n\n", qualPosition.Position, qualPosition.Driver.Code, qualPosition.Q1, qualPosition.Q2)
			} else {
				fmt.Fprintf(w, "%s |\t%s |\t\n Q1: %s \n\n", qualPosition.Position, qualPosition.Driver.Code, qualPosition.Q1)
			}
		}
	}

	w.Flush()
	return message.String()
}

func qualifyingResultToString(qualPosition Result) string {
	return fmt.Sprintf("%2s | %-3s | Q1: %-11s -- Q2: %-11s -- Q3: %-11s \n", qualPosition.Position, qualPosition.Driver.Code, qualPosition.Q1, qualPosition.Q2, qualPosition.Q3)
}

func sprintResultsToString(race Race) string {

	message := new(strings.Builder)

	w := tabwriter.NewWriter(message, 2, 5, 1, ' ', tabwriter.AlignRight)
	for _, position := range race.SprintResults {
		if position.Status == "Finished" {
			if position.Points != "0" {
				fmt.Fprintf(w, "%s |\t%s |\t %s - %s\n", position.Position, position.Driver.Code, position.Time.Time, position.Points)
			} else {
				fmt.Fprintf(w, "%s |\t%s |\t %s\n", position.Position, position.Driver.Code, position.Time.Time)
			}
		} else {
			fmt.Fprintf(w, "%s |\t%s |\t - %s\n", position.Position, position.Driver.Code, position.Status)
		}
	}

	w.Flush()
	return message.String()
}

func deleteMention(messageText string) string {
	messageText = strings.Replace(messageText, ", ", "", 1)
	messageText = strings.TrimPrefix(messageText, "[club219009582|@club219009582]")
	messageText = strings.TrimPrefix(messageText, "[club219009582|Race Bot]")
	return messageText
}

func makeCarouselGPItem(curRace Race) string {
	var buttonsArray = make([]Button, 0, 3)

	actionBtn1 := ActionBtn{TypeAction: "text", Label: "Результат гонки", Payload: fmt.Sprintf(`{"command" : "raceRes_%s"}`, curRace.Round)}
	actionBtn2 := ActionBtn{TypeAction: "text", Label: "Результат квалификации", Payload: fmt.Sprintf(`{"command" : "qualRes_%s"}`, curRace.Round)}

	btn1 := Button{actionBtn1}
	btn2 := Button{actionBtn2}
	var btn3 Button
	if curRace.Sprint.Date != "" {
		actionBtn3 := ActionBtn{TypeAction: "text", Label: "Результат спринта", Payload: fmt.Sprintf(`{"command" : "sprRes_%s"}`, curRace.Round)}
		btn3.Action = actionBtn3
	}

	buttonsArray = append(buttonsArray, btn1, btn2)
	if curRace.Sprint.Date != "" {
		buttonsArray = append(buttonsArray, btn3)
	}

	crslItem := CarouselItem{
		Title:       curRace.RaceName,
		Description: fmt.Sprintf("%s\n%s", curRace.Circuit.CircuitName, curRace.Date+", "+curRace.Time),
		PhotoID:     "-219009582_457239024",
		Action:      ActionBtn{TypeAction: "open_link", Link: curRace.Url},
		Buttons:     buttonsArray}
	crsl := Carousel{Type: "carousel", Elements: []CarouselItem{crslItem}}

	jsCrl, err := json.Marshal(crsl)
	fmt.Println(string(jsCrl))
	if err != nil {
		log.Fatal(err)
	}

	return string(jsCrl)
}

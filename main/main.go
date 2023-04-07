package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/SevereCloud/vksdk/v2/api"
	"github.com/SevereCloud/vksdk/v2/api/params"
	"github.com/SevereCloud/vksdk/v2/events"
	"github.com/SevereCloud/vksdk/v2/longpoll-bot"
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

func main() {

	token := os.Getenv("RACEVK_BOT")
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
		log.Printf("%d: %s", obj.Message.PeerID, obj.Message.Text)

		var messageToUser string
		userTimestamp := obj.Message.Date
		userDate := time.Unix(int64(userTimestamp), 0)

		messageText := deleteMention(obj.Message.Text)

		switch {
		case messageText == "личный зачёт" || messageText == "личный зачет" || messageText == "Личный зачёт" || messageText == "Личный зачет":
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

		case messageText == "календарь сезона" || messageText == "Календарь сезона":
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

		case messageText == "следующая гонка" || messageText == "Следующая гонка" || messageText == "некст гонка":
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

		case messageText == "кубок конструкторов" || messageText == "кк" || messageText == "Кубок конструкторов":
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

		default:
			messageToUser = "Прости, пока что не понимаю тебя. Но я умный и скоро научусь этому!"

		}

		b := params.NewMessagesSendBuilder()
		b.Message(messageToUser)
		b.RandomID(0)
		b.PeerID(obj.Message.PeerID)

		respCode, err := vk.MessagesSend(b.Params)
		fmt.Println(respCode)
		if err != nil {
			log.Fatal(err)
		}
	})

	// Запускаем Bots Longpoll
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
		return fmt.Sprintf("Номер этапа: %s,\nНазвание этапа: %s,\nВремя гонки: %s,\n\nПервая практика: %s,\nВторая практика: %s, \nКвалификация: %s,\nСпринт: %s.\n\n",
			race.Round, race.RaceName, race.Date+" "+race.Time, race.FirstPractice.Date+" "+race.FirstPractice.Time, race.SecondPractice.Date+" "+race.SecondPractice.Time, race.Qualifying.Date+" "+race.Qualifying.Time, race.Sprint.Date+" "+race.Sprint.Time)
	} else {
		return fmt.Sprintf("Номер этапа: %s,\nНазвание этапа: %s,\nВремя гонки: %s,\n\nПервая практика: %s,\nВторая практика: %s, \nТретья практика: %s,\nКвалификация: %s.\n",
			race.Round, race.RaceName, race.Date+" "+race.Time, race.FirstPractice.Date+" "+race.FirstPractice.Time, race.SecondPractice.Date+" "+race.SecondPractice.Time, race.ThirdPractice.Date+" "+race.ThirdPractice.Time, race.Qualifying.Date+" "+race.Qualifying.Time)
	}
}

func deleteMention(messageText string) string {
	messageText = strings.Replace(messageText, ", ", "", 1)
	messageText = strings.TrimPrefix(messageText, "[club219009582|@club219009582]")
	messageText = strings.TrimPrefix(messageText, "[club219009582|Race Bot]")
	return messageText
}

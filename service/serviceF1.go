package service

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"log/slog"
	"racebot-vk/models"
	"racebot-vk/temperrors"
	vk_api "racebot-vk/vk"
	"strconv"
	"strings"
	"text/tabwriter"
	"time"
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

type f1Storage interface {
	GetDriverStandings(userDate time.Time) ([]models.DriverStandingsItem, error)
	GetCalendar(year int) ([]models.Race, error)
	GetConstructorStandings(userDate time.Time) ([]models.ConstructorStandingsItem, error)
	GetRaceResults(userDate time.Time, raceId string) ([]models.Race, error)
	GetGPInfo(userDate time.Time, raceId string) ([]models.Race, error)
	GetQualifyingResults(userDate time.Time, raceId string) ([]models.Race, error)
	GetSprintResults(userDate time.Time, raceId string) []models.Race
}

type ServiceF1 struct {
	storage f1Storage
}

func NewServiceF1(storage f1Storage) *ServiceF1 {
	return &ServiceF1{storage}
}

func (s *ServiceF1) GetDriverStandingsMessage(userDate time.Time) (string, error) {
	driversTable, err := s.storage.GetDriverStandings(userDate)
	if err != nil {

		if errors.Is(err, temperrors.ErrEmptyList) {
			return "Личный зачёт еще не сформирован.", nil
		} else {
			slog.Error("%w", err)
			return "", err
		}
	}
	return fmt.Sprintf("Личный зачёт F1, сезон %d: \n%s", userDate.Year(), driversToString(driversTable)), nil
}

func (s *ServiceF1) GetCalendarMessage(year int) (string, error) {
	calendar, err := s.storage.GetCalendar(year)
	if err != nil {

		if errors.Is(err, temperrors.ErrEmptyList) {
			return "Календарь еще не сформирован.", nil
		} else {
			slog.Error("%w", err)
			return "", err
		}
	}
	return fmt.Sprintf("Календарь F1, сезон %d:\n%s", year, racesToString(calendar)), nil
}

func (s *ServiceF1) GetNextRaceMessage(userDate time.Time, userTimestamp int) (string, error) {
	year := userDate.Year()
	calendar, err := s.storage.GetCalendar(year)
	if err != nil {

		if errors.Is(err, temperrors.ErrEmptyList) {
			return "Календарь еще не сформирован.", nil
		} else {
			slog.Error("%w", err)
			return "", err
		}
	}

	isAfter, err := checkCurrToLastTime(int64(userTimestamp), calendar[len(calendar)-1])
	if err != nil {
		return "", err
	}
	if isAfter {
		return "Сезон закончился!", nil
	} else {
		nextRace := findNextRace(int64(userTimestamp), calendar)
		return fmt.Sprintf("Cледующий гран-при :\n%s", raceFullInfoToString(formatDateTime(nextRace))), nil
	}
}

func (s *ServiceF1) GetConstructorStandingsMessage(userDate time.Time) (string, error) {
	constStr, err := s.storage.GetConstructorStandings(userDate)
	if err != nil {

		if errors.Is(err, temperrors.ErrEmptyList) {
			return "Кубок конструктора еще не сформирован.", nil
		} else {
			slog.Error("%w", err)
			return "", err
		}
	}
	return fmt.Sprintf("Кубок конструкторов F1, сезон %d:\n%s", userDate.Year(), constructorsToString(constStr)), nil
}

func (s *ServiceF1) GetRaceResultsMessage(userDate time.Time, raceId string) (string, error) {
	results, err := s.storage.GetRaceResults(userDate, raceId)
	if err != nil {

		if errors.Is(err, temperrors.ErrEmptyList) {

			if raceId != "last" {
				tmpId, tmpErr := strconv.Atoi(raceId)
				if tmpErr != nil {
					return "", err
				}

				tmpId -= 1
				str, tmpErr := s.GetRaceResultsMessage(userDate, strconv.Itoa(tmpId))
				return str, tmpErr
			}
			results, _ = s.storage.GetRaceResults(userDate.AddDate(-1, 0, 0), raceId)

		} else {
			return "Информации о результатах данной гонки нет. Возможно она появится в будущем :)", err
		}
	}
	if raceId == "last" {
		return fmt.Sprintf("Последняя гонка F1 %s:\n%s", results[0].RaceName, raceResultsToString(results[0])), nil
	} else {
		return fmt.Sprintf("Результаты гонки %s:\n%s", results[0].RaceName, raceResultsToString(results[0])), nil
	}
}

func (s *ServiceF1) GetGPInfoCarousel(userDate time.Time, raceId string) (string, error) {
	races, err := s.storage.GetGPInfo(userDate, raceId)

	if err != nil {

		if errors.Is(err, temperrors.ErrEmptyList) {
			races, _ = s.storage.GetGPInfo(userDate.AddDate(-1, 0, 0), raceId)

		} else {
			return "", err
		}

	}

	lastGP := formatDateTime(races[0])

	strCrslItem := makeCarouselGPItem(lastGP)
	crsl := vk_api.Carousel{Type: "carousel", Elements: []vk_api.CarouselItem{strCrslItem}}
	jsCrsl, err := json.Marshal(crsl)
	if err != nil {
		return "", fmt.Errorf("error marshal carousel: %w", err)
	}

	return string(jsCrsl), nil
}

func (s *ServiceF1) GetGPKeyboard() string {
	var button vk_api.Button
	btnsRow := make([]vk_api.Button, 0, 4)
	buttons := [][]vk_api.Button{}

	for i := 0; i < 9; i++ {

		if i%4 == 0 && i != 0 {
			buttons = append(buttons, btnsRow)
			btnsRow = nil
		}
		if i == 8 {
			button = vk_api.Button{Action: vk_api.ActionBtn{TypeAction: "callback", Label: "Далее", Payload: `{"command" : "gpListPage_2", "year":""}`}, Color: "primary"}
			btnsRow = append(btnsRow, button)
			buttons = append(buttons, btnsRow)
		} else {
			button = vk_api.Button{Action: vk_api.ActionBtn{TypeAction: "callback", Label: fmt.Sprintf("%d", i+1), Payload: fmt.Sprintf(`{"command" : "gpPage_%d"}`, i+1)}}
			btnsRow = append(btnsRow, button)
		}

	}

	newKeyboard := vk_api.Kb{Inline: false, Buttons: buttons}

	jsKb, err := json.Marshal(newKeyboard)
	if err != nil {

	}

	return string(jsKb)
}

func (s *ServiceF1) GetCountDaysAfterRaceMessage(userDate time.Time, raceId string) (string, error) {

	races, err := s.storage.GetGPInfo(userDate, raceId)

	if err != nil {

		if errors.Is(err, temperrors.ErrEmptyList) {
			races, _ = s.storage.GetGPInfo(userDate.AddDate(-1, 0, 0), raceId)
		} else {
			return "", err
		}

	}
	lastRace := races[0]

	lastRaceDate := parseStringToTime(lastRace.Date, lastRace.Time)
	difference := userDate.Sub(lastRaceDate)

	return fmt.Sprintf("Дней без F1 - %d :(\n", int64(difference.Hours()/24)), nil
}

func (s *ServiceF1) GetQualifyingResultsMessage(userDate time.Time, raceId string) (string, error) {
	qualRes, err := s.storage.GetQualifyingResults(userDate, raceId)

	if err != nil {

		if errors.Is(err, temperrors.ErrEmptyList) {
			qualRes, _ = s.storage.GetQualifyingResults(userDate.AddDate(-1, 0, 0), raceId)
		} else {
			return "", err
		}
	}

	if raceId == "last" {
		return fmt.Sprintf("Последняя квалификация %s:\n%s", qualRes[0].RaceName, qualifyingResultsToString(qualRes[0])), nil
	} else {
		return fmt.Sprintf("Результаты квалификации %s:\n%s", qualRes[0].RaceName, qualifyingResultsToString(qualRes[0])), nil
	}
}

func (s *ServiceF1) GetSprintResultsMessage(userDate time.Time, raceId string) string {

	if raceId == "last" {
		return "Информации о результатах данной спринт-гонки нет. Возможно она появится в будущем :)"
	}
	sprRace := s.storage.GetSprintResults(userDate, raceId)
	if len(sprRace) > 0 {
		return fmt.Sprintf("Результаты спринт-гонки %s:\n%s", sprRace[0].RaceName, sprintResultsToString(sprRace[0]))
	} else {
		return "Информации о результатах данной спринт-гонки нет. Возможно она появится в будущем :)"
	}
}

func (s *ServiceF1) GetCountOfRaces(userDate time.Time) (int, error) {
	calendar, err := s.storage.GetCalendar(userDate.Year())
	if err != nil {
		slog.Error("%w", err)
		return 0, err
	}
	return len(calendar), nil
}

// ----------------------------------
//
//	вспомогательные функции
//
// ----------------------------------

func driversToString(drivers []models.DriverStandingsItem) string {

	var countDrivers int = len(drivers)
	driversList := make([]string, countDrivers)

	for _, driver := range drivers {
		driversList = append(driversList, driverToString(driver))
	}

	return strings.Join(driversList, "")
}

func driverToString(driver models.DriverStandingsItem) string {
	return fmt.Sprintf("%2s | %-3s - %-3s \n", driver.PositionText, driver.Driver.Code, driver.Points)
}

func constructorsToString(constructors []models.ConstructorStandingsItem) string {
	var countConstructors int = len(constructors)
	constructorsList := make([]string, countConstructors)

	for _, constructor := range constructors {
		constructorsList = append(constructorsList, constructorToString(constructor))
	}

	return strings.Join(constructorsList, "")
}

func constructorToString(constructor models.ConstructorStandingsItem) string {
	return fmt.Sprintf("%2s | %s - %-3s \n", constructor.Position, constructor.Constructor.Name, constructor.Points)
}

func racesToString(races []models.Race) string {

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

func raceToString(race models.Race) string {
	return fmt.Sprintf("Номер этапа: %s,\nНазвание этапа: %s,\nДата этапа: %s,\nВремя этапа: %s.\n\n",
		race.Round, race.RaceName, race.Date, race.Time)
}

func raceResultsToString(race models.Race) string {

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
	return message.String()
}

func formatDateTime(race models.Race) models.Race {

	tzone, err := time.LoadLocation("Europe/Moscow")
	if err != nil {
		log.Fatalln(err)
	}

	var raceDate time.Time

	if race.Time != "" {
		raceDate = parseStringToTime(race.Date, race.Time)
		race.Date = ruMonth(raceDate.Format("2006-01-02"))
		race.Time = raceDate.In(tzone).Format("15:04")
	} else {
		race.Date = ruMonth(race.Date)
		race.Time = "неизвестно"
	}

	if race.FirstPractice.Date != "" {
		fPracticeDate := parseStringToTime(race.FirstPractice.Date, race.FirstPractice.Time)
		race.FirstPractice.Date = ruMonth(fPracticeDate.Format("2006-01-02"))
		race.FirstPractice.Time = fPracticeDate.In(tzone).Format("15:04")

	}
	if race.SecondPractice.Date != "" {
		sPracticeDate := parseStringToTime(race.SecondPractice.Date, race.SecondPractice.Time)
		race.SecondPractice.Date = ruMonth(sPracticeDate.Format("2006-01-02"))
		race.SecondPractice.Time = sPracticeDate.In(tzone).Format("15:04")

	}
	if race.Qualifying.Date != "" {
		qualDate := parseStringToTime(race.Qualifying.Date, race.Qualifying.Time)
		race.Qualifying.Date = ruMonth(qualDate.Format("2006-01-02"))
		race.Qualifying.Time = qualDate.In(tzone).Format("15:04")

	}

	if len(race.Sprint.Date) > 0 {
		sprDate := parseStringToTime(race.Sprint.Date, race.Sprint.Time)
		race.Sprint.Date = ruMonth(sprDate.Format("2006-01-02"))
		race.Sprint.Time = sprDate.In(tzone).Format("15:04")
	}
	if race.ThirdPractice.Date != "" {
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

func findNextRace(messageDate int64, races []models.Race) models.Race {

	userDate := time.Unix(messageDate, 0)
	var numRace int

	for num, race := range races {

		tempDateTime, err := time.Parse("2006-01-02 15:04:05Z", fmt.Sprintf("%s %s", race.Date, race.Time))
		if err != nil {
			fmt.Errorf("Error creating date and time: %w", err)
		}

		if tempDateTime.After(userDate) {
			numRace = num
			break
		}
	}

	return races[numRace]
}

func checkCurrToLastTime(messageDate int64, race models.Race) (bool, error) {

	var lastRace time.Time
	var err error

	if race.Date == "" {
		return false, errors.New("empty date in race")
	}

	if race.Time == "" {
		lastRace, err = time.Parse("2006-01-02", fmt.Sprint(race.Date))
	} else {
		lastRace, err = time.Parse("2006-01-02 15:04:05Z", fmt.Sprintf("%s %s", race.Date, race.Time))
	}
	if err != nil {
		return false, fmt.Errorf("Error creating date and time: %w", err)
	}

	if messageDate >= int64(lastRace.Unix()) {
		return true, nil
	} else {
		return false, nil
	}
}

func raceFullInfoToString(race models.Race) string {
	if len(race.Sprint.Date) > 0 {
		return fmt.Sprintf("Номер этапа: %s,\nНазвание этапа: %s,\nВремя гонки: %s,\n\nПрактика: %s,\nКвалификация: %s,\n\nКвалификация спринта: %s, \nСпринт: %s.\n\n",
			race.Round, race.RaceName, race.Date+" "+race.Time, race.FirstPractice.Date+" "+race.FirstPractice.Time, race.Qualifying.Date+" "+race.Qualifying.Time, race.SecondPractice.Date+" "+race.SecondPractice.Time, race.Sprint.Date+" "+race.Sprint.Time)
	} else {
		return fmt.Sprintf("Номер этапа: %s,\nНазвание этапа: %s,\nВремя гонки: %s,\n\nПервая практика: %s,\nВторая практика: %s, \nТретья практика: %s,\nКвалификация: %s.\n",
			race.Round, race.RaceName, race.Date+" "+race.Time, race.FirstPractice.Date+" "+race.FirstPractice.Time, race.SecondPractice.Date+" "+race.SecondPractice.Time, race.ThirdPractice.Date+" "+race.ThirdPractice.Time, race.Qualifying.Date+" "+race.Qualifying.Time)
	}
}

func makeCarouselGPItem(curRace models.Race) vk_api.CarouselItem {
	var buttonsArray = make([]vk_api.Button, 0, 3)

	actionBtn1 := vk_api.ActionBtn{TypeAction: "text", Label: "Результат гонки", Payload: fmt.Sprintf(`{"command" : "raceRes_%s"}`, curRace.Round)}
	actionBtn2 := vk_api.ActionBtn{TypeAction: "text", Label: "Результат квалификации", Payload: fmt.Sprintf(`{"command" : "qualRes_%s"}`, curRace.Round)}

	btn1 := vk_api.Button{Action: actionBtn1}
	btn2 := vk_api.Button{Action: actionBtn2}

	buttonsArray = append(buttonsArray, btn1, btn2)

	if curRace.Sprint.Date != "" {
		btn3 := vk_api.Button{Action: vk_api.ActionBtn{TypeAction: "text", Label: "Результат спринта", Payload: fmt.Sprintf(`{"command" : "sprRes_%s"}`, curRace.Round)}}
		buttonsArray = append(buttonsArray, btn3)
	}

	crslItem := vk_api.CarouselItem{
		Title:       curRace.RaceName,
		Description: fmt.Sprintf("%s\n%s", curRace.Circuit.CircuitName, curRace.Date+", "+curRace.Time),
		PhotoID:     "-219009582_457239025",
		Action:      vk_api.ActionBtn{TypeAction: "open_link", Link: curRace.Url},
		Buttons:     buttonsArray}

	return crslItem
}

func qualifyingResultsToString(race models.Race) string {

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

/*
func qualifyingResultToString(qualPosition models.Result) string {
	return fmt.Sprintf("%2s | %-3s | Q1: %-11s -- Q2: %-11s -- Q3: %-11s \n", qualPosition.Position, qualPosition.Driver.Code, qualPosition.Q1, qualPosition.Q2, qualPosition.Q3)
}
*/

func sprintResultsToString(race models.Race) string {

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

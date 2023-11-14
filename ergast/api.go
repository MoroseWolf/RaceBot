package ergast

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"racebot-vk/models"
	"time"
)

type ErgastAPI struct {
	url string
}

func NewErgastAPI() *ErgastAPI {
	return &ErgastAPI{url: "http://ergast.com/api/f1"}
}

func (erg *ErgastAPI) GetDriverStandings(userDate time.Time) []models.DriverStandingsItem {
	resp := getRequest(fmt.Sprintf("%s/%d/driverStandings.json", erg.url, userDate.Year()))
	return resp.MRData.StandingsTable.StandingsLists[0].DriverStandings
}

func (erg *ErgastAPI) GetCalendar(year int) []models.Race {
	resp := getRequest(fmt.Sprintf("%s/%d.json", erg.url, year))
	return resp.MRData.RaceTable.Races
}

func (erg *ErgastAPI) GetConstructorStandings(userDate time.Time) []models.ConstructorStandingsItem {
	resp := getRequest(fmt.Sprintf("%s/%d/constructorStandings.json", erg.url, userDate.Year()))
	return resp.MRData.StandingsTable.StandingsLists[0].ConstructorStandings
}

func (erg *ErgastAPI) GetRaceResults(userDate time.Time, raceId string) []models.Race {
	resp := getRequest(fmt.Sprintf("%s/%d/%s/results.json", erg.url, userDate.Year(), raceId))
	return resp.MRData.RaceTable.Races
}

func (erg *ErgastAPI) GetGPInfo(userDate time.Time, raceId string) []models.Race {
	resp := getRequest(fmt.Sprintf("%s/%d/%s.json", erg.url, userDate.Year(), raceId))
	return resp.MRData.RaceTable.Races
}

func (erg *ErgastAPI) GetQualifyingResults(userDate time.Time, raceID string) []models.Race {
	resp := getRequest(fmt.Sprintf("%s/%d/%s/qualifying.json", erg.url, userDate.Year(), raceID))
	return resp.MRData.RaceTable.Races
}

func (erg *ErgastAPI) GetSprintResults(userDate time.Time, raceId string) []models.Race {
	resp := getRequest(fmt.Sprintf("%s/%d/%s/sprint.json", erg.url, userDate.Year(), raceId))
	return resp.MRData.RaceTable.Races
}

func getRequest(url string) models.Object {
	resp, err := http.Get(url)
	if err != nil {
		log.Fatalln(err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Fatalln(err)
	}

	var temp models.Object
	json.Unmarshal([]byte(body), &temp)
	return temp
}

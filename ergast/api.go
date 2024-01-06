package ergast

import (
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"racebot-vk/models"
	"racebot-vk/temperrors"
	"time"
)

type ErgastAPI struct {
	url string
}

func NewErgastAPI() *ErgastAPI {
	return &ErgastAPI{url: "http://ergast.com/api/f1"}
}

func (erg *ErgastAPI) GetDriverStandings(userDate time.Time) ([]models.DriverStandingsItem, error) {
	resp, err := getRequest(fmt.Sprintf("%s/%d/driverStandings.json", erg.url, userDate.Year()))
	if err != nil {
		return nil, fmt.Errorf("in driverStanding %w", err)
	}
	if len(resp.MRData.StandingsTable.StandingsLists) > 0 {
		return resp.MRData.StandingsTable.StandingsLists[0].DriverStandings, nil
	} else {
		return nil, temperrors.ErrEmptyList
	}

}

func (erg *ErgastAPI) GetCalendar(year int) ([]models.Race, error) {
	resp, err := getRequest(fmt.Sprintf("%s/%d.json", erg.url, year))
	if err != nil {
		slog.Info(err.Error())
		return nil, fmt.Errorf("in calendar %w", err)
	}
	if len(resp.MRData.RaceTable.Races) > 0 {
		return resp.MRData.RaceTable.Races, nil
	} else {
		return nil, temperrors.ErrEmptyList
	}

}

func (erg *ErgastAPI) GetConstructorStandings(userDate time.Time) ([]models.ConstructorStandingsItem, error) {
	resp, err := getRequest(fmt.Sprintf("%s/%d/constructorStandings.json", erg.url, userDate.Year()))
	if err != nil {
		return nil, fmt.Errorf("in constructorStanding %w", err)
	}
	if len(resp.MRData.StandingsTable.StandingsLists) > 0 {
		return resp.MRData.StandingsTable.StandingsLists[0].ConstructorStandings, nil
	} else {
		return nil, temperrors.ErrEmptyList
	}

}

func (erg *ErgastAPI) GetRaceResults(userDate time.Time, raceId string) ([]models.Race, error) {
	resp, err := getRequest(fmt.Sprintf("%s/%d/%s/results.json", erg.url, userDate.Year(), raceId))
	if err != nil {
		return nil, fmt.Errorf("in raceResults %w", err)
	}
	if len(resp.MRData.RaceTable.Races) > 0 {
		return resp.MRData.RaceTable.Races, nil
	}
	return nil, temperrors.ErrEmptyList
}

func (erg *ErgastAPI) GetGPInfo(userDate time.Time, raceId string) []models.Race {
	resp, _ := getRequest(fmt.Sprintf("%s/%d/%s.json", erg.url, userDate.Year(), raceId))
	return resp.MRData.RaceTable.Races
}

func (erg *ErgastAPI) GetQualifyingResults(userDate time.Time, raceID string) []models.Race {
	resp, _ := getRequest(fmt.Sprintf("%s/%d/%s/qualifying.json", erg.url, userDate.Year(), raceID))
	return resp.MRData.RaceTable.Races
}

func (erg *ErgastAPI) GetSprintResults(userDate time.Time, raceId string) []models.Race {
	resp, _ := getRequest(fmt.Sprintf("%s/%d/%s/sprint.json", erg.url, userDate.Year(), raceId))
	return resp.MRData.RaceTable.Races
}

func getRequest(url string) (models.Object, error) {
	var temp models.Object

	resp, err := http.Get(url)
	if err != nil {
		return temp, fmt.Errorf("error in getRequest", err)
	} else {
		slog.Info("OK get request")
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return temp, fmt.Errorf("error reading responce", err)
	}

	json.Unmarshal([]byte(body), &temp)
	return temp, nil
}

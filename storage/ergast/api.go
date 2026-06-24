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
	url    string
	client *http.Client
	cache  *cache
}

func NewErgastAPI() *ErgastAPI {
	return &ErgastAPI{
		url: "http://api.jolpi.ca/ergast/f1",
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
		cache: newCache(15 * time.Minute),
	}
}

func (erg *ErgastAPI) GetDriversList(userDate time.Time) ([]models.Driver, error) {
	resp, err := erg.getRequest(fmt.Sprintf("%s/%d/drivers.json", erg.url, userDate.Year()))
	if err != nil {
		return nil, fmt.Errorf("in driversList %w", err)
	}
	if len(resp.MRData.DriverTable.Drivers) > 0 {
		return resp.MRData.DriverTable.Drivers, nil
	}
	return nil, temperrors.ErrEmptyList
}

func (erg *ErgastAPI) GetDriverStandings(userDate time.Time) ([]models.DriverStandingsItem, error) {
	resp, err := erg.getRequest(fmt.Sprintf("%s/%d/driverStandings.json", erg.url, userDate.Year()))
	if err != nil {
		return nil, fmt.Errorf("in driverStanding %w", err)
	}
	if len(resp.MRData.StandingsTable.StandingsLists) > 0 {
		return resp.MRData.StandingsTable.StandingsLists[0].DriverStandings, nil
	}
	return nil, temperrors.ErrEmptyList

}

func (erg *ErgastAPI) GetCalendar(year int) ([]models.Race, error) {
	resp, err := erg.getRequest(fmt.Sprintf("%s/%d.json", erg.url, year))
	if err != nil {
		slog.Error("failed to get calendar", slog.Any("error", err))
		return nil, fmt.Errorf("in calendar %w", err)
	}
	if len(resp.MRData.RaceTable.Races) > 0 {
		return resp.MRData.RaceTable.Races, nil
	}
	return nil, temperrors.ErrEmptyList

}

func (erg *ErgastAPI) GetConstructorStandings(userDate time.Time) ([]models.ConstructorStandingsItem, error) {
	resp, err := erg.getRequest(fmt.Sprintf("%s/%d/constructorStandings.json", erg.url, userDate.Year()))
	if err != nil {
		return nil, fmt.Errorf("in constructorStanding %w", err)
	}
	if len(resp.MRData.StandingsTable.StandingsLists) > 0 {
		return resp.MRData.StandingsTable.StandingsLists[0].ConstructorStandings, nil
	}
	return nil, temperrors.ErrEmptyList

}

func (erg *ErgastAPI) GetRaceResults(userDate time.Time, raceId string) ([]models.Race, error) {
	resp, err := erg.getRequest(fmt.Sprintf("%s/%d/%s/results.json", erg.url, userDate.Year(), raceId))
	if err != nil {
		return nil, fmt.Errorf("in raceResults %w", err)
	}
	if len(resp.MRData.RaceTable.Races) > 0 {
		return resp.MRData.RaceTable.Races, nil
	}
	return nil, temperrors.ErrEmptyList
}

func (erg *ErgastAPI) GetGPInfo(userDate time.Time, raceId string) ([]models.Race, error) {
	resp, err := erg.getRequest(fmt.Sprintf("%s/%d/%s.json", erg.url, userDate.Year(), raceId))
	if err != nil {
		return nil, fmt.Errorf("in getGPInfo %w", err)
	}

	if len(resp.MRData.RaceTable.Races) > 0 {
		return resp.MRData.RaceTable.Races, nil
	}
	return nil, temperrors.ErrEmptyList
}

func (erg *ErgastAPI) GetQualifyingResults(userDate time.Time, raceID string) ([]models.Race, error) {
	resp, err := erg.getRequest(fmt.Sprintf("%s/%d/%s/qualifying.json", erg.url, userDate.Year(), raceID))
	if err != nil {
		return nil, fmt.Errorf("in getQualifyingResults %w", err)
	}

	if len(resp.MRData.RaceTable.Races) > 0 {
		return resp.MRData.RaceTable.Races, nil
	}
	return nil, temperrors.ErrEmptyList
}

func (erg *ErgastAPI) GetSprintResults(userDate time.Time, raceId string) []models.Race {
	resp, err := erg.getRequest(fmt.Sprintf("%s/%d/%s/sprint.json", erg.url, userDate.Year(), raceId))
	if err != nil {
		slog.Error("failed to get sprint results", slog.Any("error", err))
		return nil
	}
	return resp.MRData.RaceTable.Races
}

func (erg *ErgastAPI) getRequest(url string) (models.Object, error) {

	if data, ok := erg.cache.get(url); ok {
		slog.Debug("cache hit", slog.String("url", url))
		return data, nil
	}

	var temp models.Object

	resp, err := erg.client.Get(url)
	if err != nil {
		return temp, fmt.Errorf("error in getRequest %w", err)
	}
	defer resp.Body.Close()

	slog.Info("OK get request", slog.String("url", url))

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return temp, fmt.Errorf("error reading response %w", err)
	}

	if err := json.Unmarshal(body, &temp); err != nil {
		return temp, fmt.Errorf("error unmarshalling response: %w", err)
	}

	erg.cache.set(url, temp)
	return temp, nil
}

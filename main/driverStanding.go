package main

type ConstructorStandingsItem struct {
	Position    string
	Points      string
	Wins        string
	Constructor Constructors
}

type DriverStandingsItem struct {
	Position     int
	PositionText string
	Points       string
	Wins         string
	Driver       Driver
	Constructors []Constructors
}

type Constructors struct {
	ConstructorId string
	Url           string
	Name          string
	Nationality   string
}

type StandingsListItem struct {
	Season               string
	Round                string
	DriverStandings      []DriverStandingsItem
	ConstructorStandings []ConstructorStandingsItem
}

type StandingsTable struct {
	Season         string
	Round          string
	StandingsLists []StandingsListItem
}

package main

type Location struct {
	Lat      string
	Long     string
	Locality string
	Country  string
}

type FirstPractice struct {
	Date string
	Time string
}

type SecondPractice struct {
	Date string
	Time string
}

type ThirdPractice struct {
	Date string
	Time string
}

type Qualifying struct {
	Date string
	Time string
}

type Sprint struct {
	Date string
	Time string
}

type Circuit struct {
	CircuitId   string
	Url         string
	CircuitName string
	Location    Location
}

type Result struct {
	Number      string
	Position    string
	Points      string
	Driver      Driver
	Constructor Constructors
	Grid        string
	Laps        string
	Status      string
	Time        Time
	FastestLap  FastestLap
	Q1          string
	Q2          string
	Q3          string
}

type Time struct {
	Millis string
	Time   string
}
type AverageSpeed struct {
	Units string
	Speed string
}

type FastestLap struct {
	Rank         string
	Lap          string
	Time         Time
	AverageSpeed AverageSpeed
}

type Race struct {
	Season            string
	Round             string
	Url               string
	RaceName          string
	Circuit           Circuit
	Date              string
	Time              string
	FirstPractice     FirstPractice
	SecondPractice    SecondPractice
	ThirdPractice     ThirdPractice
	Qualifying        Qualifying
	Sprint            Sprint
	Results           []Result
	QualifyingResults []Result
	SprintResults     []Result
}

type RaceTable struct {
	Season string
	Races  []Race
}

type MRData struct {
	Series         string
	RaceTable      RaceTable
	StandingsTable StandingsTable
}

type Object struct {
	MRData MRData
}

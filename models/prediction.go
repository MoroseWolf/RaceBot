package models

import "time"

// Прогноз пользователя на гонку
type Prediction struct {
	ID        int       `json:"id"`
	UserID    int       `json:"user_id"`
	RaceID    string    `json:"race_id"` // "2025_1" (season_round)
	Driver1   uint8     `json:"driver_1"`
	Driver2   uint8     `json:"driver_2"`
	Driver3   uint8     `json:"driver_3"`
	Points    int       `json:"points"`
	CreatedAt time.Time `json:"created_at"`
}

// Раунд прогнозов на конкретную гонку
type PredictionRace struct {
	ID        int        `json:"id"`
	RaceID    string     `json:"race_id"` // "2025_1" (season_round)
	RaceName  string     `json:"race_name"`
	IsActive  bool       `json:"is_active"`          // true - приём прогнозов открыт
	Driver1   *uint8     `json:"driver_1,omitempty"` // реальный результат (номер гонщика)
	Driver2   *uint8     `json:"driver_2,omitempty"`
	Driver3   *uint8     `json:"driver_3,omitempty"`
	CreatedAt time.Time  `json:"created_at"`
	ClosedAt  *time.Time `json:"closed_at,omitempty"`
}

// Статичтика пользователя по прогнозам
type UserStats struct {
	UserID      int     `json:"user_id"`
	TotalPoints int     `json:"total_points"`
	TotalRaces  int     `json:"total_races"`
	AvgPoints   float64 `json:"avg_points"`
	BestRaceID  string  `json:"best_race_id,omitempty"`
	BestPoints  int     `json:"best_points,omitempty"`
}

// Результат одного прогноза после подсчёта
type PredictionResult struct {
	UserID     int        `json:"user_id"`
	Prediction Prediction `json:"prediction"`
	Points     int        `json:"points"`
	Details    string     `json:"details"`
}

package prediction

import (
	"database/sql"
	"fmt"
	"racebot-vk/models"
	"time"

	_ "modernc.org/sqlite"
)

// Storage реализует хранение прогнозов в SQLite
type Storage struct {
	db *sql.DB
}

// NewStorage создаёт новый экземпляр Storage и инициализирует БД
func NewStorage(dbPath string) (*Storage, error) {
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Настройки подключения
	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)
	db.SetConnMaxLifetime(5 * time.Minute)

	storage := &Storage{db: db}
	if err := storage.initDB(); err != nil {
		return nil, fmt.Errorf("failed to init database: %w", err)
	}

	return storage, nil
}

// Close закрывает соединение с БД
func (s *Storage) Close() error {
	return s.db.Close()
}

// initDB создаёт таблицы, если их нет
func (s *Storage) initDB() error {
	queries := []string{
		`CREATE TABLE IF NOT EXISTS prediction_races (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			race_id TEXT NOT NULL UNIQUE,
			race_name TEXT NOT NULL,
			is_active INTEGER NOT NULL DEFAULT 1,
			driver_1 INTEGER,
			driver_2 INTEGER,
			driver_3 INTEGER,
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			closed_at DATETIME
		)`,
		`CREATE TABLE IF NOT EXISTS predictions (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			user_id INTEGER NOT NULL,
			race_id TEXT NOT NULL,
			driver_1 INTEGER NOT NULL,
			driver_2 INTEGER NOT NULL,
			driver_3 INTEGER NOT NULL,
			points INTEGER NOT NULL DEFAULT 0,
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			UNIQUE(user_id, race_id)
		)`,
		`CREATE INDEX IF NOT EXISTS idx_predictions_user_id ON predictions(user_id)`,
		`CREATE INDEX IF NOT EXISTS idx_predictions_race_id ON predictions(race_id)`,
		`CREATE INDEX IF NOT EXISTS idx_prediction_races_active ON prediction_races(is_active)`,
	}

	for _, q := range queries {
		if _, err := s.db.Exec(q); err != nil {
			return fmt.Errorf("failed to execute query: %w", err)
		}
	}

	return nil
}

// CreateRace создаёт новый раунд прогнозов
func (s *Storage) CreateRace(race *models.PredictionRace) error {
	query := `INSERT INTO prediction_races (race_id, race_name, is_active) VALUES (?, ?, ?)`
	_, err := s.db.Exec(query, race.RaceID, race.RaceName, boolToInt(race.IsActive))
	if err != nil {
		return fmt.Errorf("failed to create race: %w", err)
	}
	return nil
}

// CloseRace закрывает приём прогнозов для указанной гонки
func (s *Storage) CloseRace(raceID string) error {
	query := `UPDATE prediction_races SET is_active = 0, closed_at = CURRENT_TIMESTAMP WHERE race_id = ?`
	result, err := s.db.Exec(query, raceID)
	if err != nil {
		return fmt.Errorf("failed to close race: %w", err)
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("race %s not found", raceID)
	}
	return nil
}

// SetRaceResults сохраняет реальные результаты гонки
func (s *Storage) SetRaceResults(raceID string, d1, d2, d3 uint8) error {
	query := `UPDATE prediction_races SET driver_1 = ?, driver_2 = ?, driver_3 = ? WHERE race_id = ?`
	result, err := s.db.Exec(query, d1, d2, d3, raceID)
	if err != nil {
		return fmt.Errorf("failed to set race results: %w", err)
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("race %s not found", raceID)
	}
	return nil
}

// GetActiveRace возвращает активный раунд прогнозов (если есть)
func (s *Storage) GetActiveRace() (*models.PredictionRace, error) {
	query := `SELECT id, race_id, race_name, is_active, driver_1, driver_2, driver_3, created_at, closed_at 
			  FROM prediction_races WHERE is_active = 1 LIMIT 1`

	race := &models.PredictionRace{}
	var isActive int
	var driver1, driver2, driver3 sql.NullInt64
	var closedAt sql.NullTime

	err := s.db.QueryRow(query).Scan(
		&race.ID, &race.RaceID, &race.RaceName, &isActive,
		&driver1, &driver2, &driver3,
		&race.CreatedAt, &closedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get active race: %w", err)
	}

	race.IsActive = intToBool(isActive)
	if driver1.Valid {
		v := uint8(driver1.Int64)
		race.Driver1 = &v
	}
	if driver2.Valid {
		v := uint8(driver2.Int64)
		race.Driver2 = &v
	}
	if driver3.Valid {
		v := uint8(driver3.Int64)
		race.Driver3 = &v
	}
	if closedAt.Valid {
		race.ClosedAt = &closedAt.Time
	}

	return race, nil
}

// GetRaceByID возвращает раунд прогнозов по race_id
func (s *Storage) GetRaceByID(raceID string) (*models.PredictionRace, error) {
	query := `SELECT id, race_id, race_name, is_active, driver_1, driver_2, driver_3, created_at, closed_at 
			  FROM prediction_races WHERE race_id = ?`

	race := &models.PredictionRace{}
	var isActive int
	var driver1, driver2, driver3 sql.NullInt64
	var closedAt sql.NullTime

	err := s.db.QueryRow(query, raceID).Scan(
		&race.ID, &race.RaceID, &race.RaceName, &isActive,
		&driver1, &driver2, &driver3,
		&race.CreatedAt, &closedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get race by id: %w", err)
	}

	race.IsActive = intToBool(isActive)
	if driver1.Valid {
		v := uint8(driver1.Int64)
		race.Driver1 = &v
	}
	if driver2.Valid {
		v := uint8(driver2.Int64)
		race.Driver2 = &v
	}
	if driver3.Valid {
		v := uint8(driver3.Int64)
		race.Driver3 = &v
	}
	if closedAt.Valid {
		race.ClosedAt = &closedAt.Time
	}

	return race, nil
}

// SavePrediction сохраняет прогноз пользователя
func (s *Storage) SavePrediction(pred *models.Prediction) error {
	query := `INSERT INTO predictions (user_id, race_id, driver_1, driver_2, driver_3) 
			  VALUES (?, ?, ?, ?, ?)
			  ON CONFLICT(user_id, race_id) DO UPDATE SET 
			  driver_1 = excluded.driver_1, driver_2 = excluded.driver_2, driver_3 = excluded.driver_3`
	_, err := s.db.Exec(query, pred.UserID, pred.RaceID, pred.Driver1, pred.Driver2, pred.Driver3)
	if err != nil {
		return fmt.Errorf("failed to save prediction: %w", err)
	}
	return nil
}

// GetUserPredictions возвращает все прогнозы пользователя
func (s *Storage) GetUserPredictions(userID int) ([]models.Prediction, error) {
	query := `SELECT id, user_id, race_id, driver_1, driver_2, driver_3, points, created_at 
			  FROM predictions WHERE user_id = ? ORDER BY created_at DESC`
	rows, err := s.db.Query(query, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user predictions: %w", err)
	}
	defer rows.Close()

	return scanPredictions(rows)
}

// GetRacePredictions возвращает все прогнозы на указанную гонку
func (s *Storage) GetRacePredictions(raceID string) ([]models.Prediction, error) {
	query := `SELECT id, user_id, race_id, driver_1, driver_2, driver_3, points, created_at 
			  FROM predictions WHERE race_id = ? ORDER BY points DESC, created_at ASC`
	rows, err := s.db.Query(query, raceID)
	if err != nil {
		return nil, fmt.Errorf("failed to get race predictions: %w", err)
	}
	defer rows.Close()

	return scanPredictions(rows)
}

// UpdatePredictionPoints обновляет очки для прогноза
func (s *Storage) UpdatePredictionPoints(id, points int) error {
	query := `UPDATE predictions SET points = ? WHERE id = ?`
	_, err := s.db.Exec(query, points, id)
	if err != nil {
		return fmt.Errorf("failed to update prediction points: %w", err)
	}
	return nil
}

// GetLeaderboard возвращает общую таблицу лидеров
func (s *Storage) GetLeaderboard() ([]models.UserStats, error) {
	query := `SELECT user_id, SUM(points) as total_points, COUNT(*) as total_races
			  FROM predictions
			  GROUP BY user_id
			  ORDER BY total_points DESC, total_races ASC`

	rows, err := s.db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to get leaderboard: %w", err)
	}
	defer rows.Close()

	var stats []models.UserStats
	for rows.Next() {
		var st models.UserStats
		if err := rows.Scan(&st.UserID, &st.TotalPoints, &st.TotalRaces); err != nil {
			return nil, fmt.Errorf("failed to scan leaderboard row: %w", err)
		}
		if st.TotalRaces > 0 {
			st.AvgPoints = float64(st.TotalPoints) / float64(st.TotalRaces)
		}
		stats = append(stats, st)
	}

	return stats, nil
}

// GetUserStats возвращает статистику пользователя
func (s *Storage) GetUserStats(userID int) (*models.UserStats, error) {
	query := `SELECT user_id, SUM(points) as total_points, COUNT(*) as total_races,
			  COALESCE(MAX(points), 0) as best_points
			  FROM predictions
			  WHERE user_id = ?`

	st := &models.UserStats{}
	var bestPoints int
	err := s.db.QueryRow(query, userID).Scan(&st.UserID, &st.TotalPoints, &st.TotalRaces, &bestPoints)
	if err != nil {
		if err == sql.ErrNoRows {
			return &models.UserStats{UserID: userID}, nil
		}
		return nil, fmt.Errorf("failed to get user stats: %w", err)
	}

	if st.TotalRaces > 0 {
		st.AvgPoints = float64(st.TotalPoints) / float64(st.TotalRaces)
	}

	// Находим лучшую гонку
	bestQuery := `SELECT race_id FROM predictions WHERE user_id = ? AND points = ? LIMIT 1`
	err = s.db.QueryRow(bestQuery, userID, bestPoints).Scan(&st.BestRaceID)
	if err != nil && err != sql.ErrNoRows {
		return nil, fmt.Errorf("failed to get best race: %w", err)
	}
	st.BestPoints = bestPoints

	return st, nil
}

// GetAllRaces возвращает все раунды прогнозов
func (s *Storage) GetAllRaces() ([]models.PredictionRace, error) {
	query := `SELECT id, race_id, race_name, is_active, driver_1, driver_2, driver_3, created_at, closed_at 
			  FROM prediction_races ORDER BY created_at DESC`
	rows, err := s.db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to get all races: %w", err)
	}
	defer rows.Close()

	var races []models.PredictionRace
	for rows.Next() {
		var race models.PredictionRace
		var isActive int
		var driver1, driver2, driver3 sql.NullInt64
		var closedAt sql.NullTime

		if err := rows.Scan(
			&race.ID, &race.RaceID, &race.RaceName, &isActive,
			&driver1, &driver2, &driver3,
			&race.CreatedAt, &closedAt,
		); err != nil {
			return nil, fmt.Errorf("failed to scan race row: %w", err)
		}

		race.IsActive = intToBool(isActive)
		if driver1.Valid {
			v := uint8(driver1.Int64)
			race.Driver1 = &v
		}
		if driver2.Valid {
			v := uint8(driver2.Int64)
			race.Driver2 = &v
		}
		if driver3.Valid {
			v := uint8(driver3.Int64)
			race.Driver3 = &v
		}
		if closedAt.Valid {
			race.ClosedAt = &closedAt.Time
		}

		races = append(races, race)
	}

	return races, nil
}

// --- Вспомогательные функции ---

func scanPredictions(rows *sql.Rows) ([]models.Prediction, error) {
	var predictions []models.Prediction
	for rows.Next() {
		var p models.Prediction
		if err := rows.Scan(&p.ID, &p.UserID, &p.RaceID, &p.Driver1, &p.Driver2, &p.Driver3, &p.Points, &p.CreatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan prediction row: %w", err)
		}
		predictions = append(predictions, p)
	}
	return predictions, nil
}

func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}

func intToBool(i int) bool {
	return i == 1
}

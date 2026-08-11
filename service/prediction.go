package service

import (
	"fmt"
	"racebot-vk/models"
	predStorage "racebot-vk/storage/prediction"
)

type PredictionService struct {
	storage *predStorage.Storage
}

func NewPredictionService(storage *predStorage.Storage) *PredictionService {
	return &PredictionService{storage: storage}
}

// StartPrediction открывает конкурс прогнозов на указанную гонку
func (s *PredictionService) StartPrediction(raceID, raceName string) error {
	// Проверяем, нет ли уже активного раунда
	activeRace, err := s.storage.GetActiveRace()
	if err != nil {
		return fmt.Errorf("failed to check active race: %w", err)
	}
	if activeRace != nil {
		return fmt.Errorf("прогноз уже активен для гонки: %s", activeRace.RaceName)
	}

	race := &models.PredictionRace{
		RaceID:   raceID,
		RaceName: raceName,
		IsActive: true,
	}

	return s.storage.CreateRace(race)
}

// ClosePrediction закрывает приём прогнозов
func (s *PredictionService) ClosePrediction(raceID string) error {
	return s.storage.CloseRace(raceID)
}

// SetRaceResult сохраняет реальные результаты гонки
func (s *PredictionService) SetRaceResult(raceID string, d1, d2, d3 uint8) error {
	return s.storage.SetRaceResults(raceID, d1, d2, d3)
}

// SubmitPrediction сохраняет прогноз пользователя
func (s *PredictionService) SubmitPrediction(userID int, raceID string, d1, d2, d3 uint8) error {
	pred := &models.Prediction{
		UserID:  userID,
		RaceID:  raceID,
		Driver1: d1,
		Driver2: d2,
		Driver3: d3,
	}
	return s.storage.SavePrediction(pred)
}

// CalculateResults подсчитывает очки для всех прогнозов на указанную гонку
func (s *PredictionService) CalculateResults(raceID string) ([]models.PredictionResult, error) {
	// Получаем результаты гонки
	race, err := s.storage.GetRaceByID(raceID)
	if err != nil {
		return nil, fmt.Errorf("failed to get race: %w", err)
	}
	if race == nil {
		return nil, fmt.Errorf("гонка %s не найдена", raceID)
	}
	if race.Driver1 == nil || race.Driver2 == nil || race.Driver3 == nil {
		return nil, fmt.Errorf("результаты гонки %s ещё не установлены", raceID)
	}

	// Получаем все прогнозы на эту гонку
	predictions, err := s.storage.GetRacePredictions(raceID)
	if err != nil {
		return nil, fmt.Errorf("failed to get predictions: %w", err)
	}

	realResults := []uint8{*race.Driver1, *race.Driver2, *race.Driver3}

	var results []models.PredictionResult
	for _, pred := range predictions {
		predPoints := calculatePoints(pred, realResults)
		pred.Points = predPoints

		// Обновляем очки в БД
		if err := s.storage.UpdatePredictionPoints(pred.ID, predPoints); err != nil {
			return nil, fmt.Errorf("failed to update points: %w", err)
		}

		details := formatPredictionDetails(pred, realResults)

		results = append(results, models.PredictionResult{
			UserID:     pred.UserID,
			Prediction: pred,
			Points:     predPoints,
			Details:    details,
		})
	}

	return results, nil
}

// GetActiveRace возвращает активный раунд прогнозов
func (s *PredictionService) GetActiveRace() (*models.PredictionRace, error) {
	return s.storage.GetActiveRace()
}

// GetRaceByID возвращает раунд прогнозов по race_id
func (s *PredictionService) GetRaceByID(raceID string) (*models.PredictionRace, error) {
	return s.storage.GetRaceByID(raceID)
}

// GetLeaderboard возвращает таблицу лидеров
func (s *PredictionService) GetLeaderboard() ([]models.UserStats, error) {
	return s.storage.GetLeaderboard()
}

// GetUserStats возвращает статистику пользователя
func (s *PredictionService) GetUserStats(userID int) (*models.UserStats, error) {
	return s.storage.GetUserStats(userID)
}

// GetRacePredictions возвращает все прогнозы на гонку
func (s *PredictionService) GetRacePredictions(raceID string) ([]models.Prediction, error) {
	return s.storage.GetRacePredictions(raceID)
}

// GetAllRaces возвращает все раунды прогнозов
func (s *PredictionService) GetAllRaces() ([]models.PredictionRace, error) {
	return s.storage.GetAllRaces()
}

// --- Вспомогательные функции ---

// calculatePoints рассчитывает очки за прогноз
// Правила:
// - Точное попадание на позицию: 5 очков
// - Гонщик из прогноза попал в топ-3, но не на своей позиции: 3 очка
// - Гонщик из прогноза не попал в топ-3: 0 очков
func calculatePoints(pred models.Prediction, realResults []uint8) int {
	points := 0
	predDrivers := []uint8{pred.Driver1, pred.Driver2, pred.Driver3}

	for i, predicted := range predDrivers {
		if predicted == realResults[i] {
			// Точное попадание
			points += 5
		} else if contains(realResults, predicted) {
			// Гонщик в топ-3, но не на своей позиции
			points += 3
		}
	}

	return points
}

// formatPredictionDetails формирует текстовое описание результатов прогноза
func formatPredictionDetails(pred models.Prediction, realResults []uint8) string {
	predDrivers := []uint8{pred.Driver1, pred.Driver2, pred.Driver3}
	labels := []string{"1", "2", "3"}
	result := ""

	for i, predicted := range predDrivers {
		if i > 0 {
			result += ", "
		}
		if predicted == realResults[i] {
			result += fmt.Sprintf("%s. №%d ✓", labels[i], predicted)
		} else if contains(realResults, predicted) {
			result += fmt.Sprintf("%s. №%d △", labels[i], predicted)
		} else {
			result += fmt.Sprintf("%s. №%d ✗", labels[i], predicted)
		}
	}

	return result
}

func contains(slice []uint8, val uint8) bool {
	for _, item := range slice {
		if item == val {
			return true
		}
	}
	return false
}

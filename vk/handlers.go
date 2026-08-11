package vk

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"racebot-vk/models"
	"strconv"
	"strings"
	"time"

	"github.com/SevereCloud/vksdk/v3/events"
)

// messageHandlerFunc обрабатывает текстовую команду и возвращает результат отправки
type messageHandlerFunc func(ctx handlerContext) error

// eventHandlerFunc обрабатывает event-команду
type eventHandlerFunc func(ctx eventHandlerContext) error

// handlerContext — контекст для обработчика текстовой команды
type handlerContext struct {
	log           *slog.Logger
	vk            *VkAPI
	myUsrVk       *MyVk
	obj           events.MessageNewObject
	userDate      time.Time
	userTimestamp int
	messageText   string
	raceID        string
}

// eventHandlerContext — контекст для обработчика event-команды
type eventHandlerContext struct {
	log     *slog.Logger
	vk      *VkAPI
	obj     events.MessageEventObject
	payload string
}

// messageHandlers — карта текстовых команд
var messageHandlers = map[command]messageHandlerFunc{
	commandHello:              handleHello,
	commandHelp:               handleHelp,
	commandDrSt:               handleDriverStandings,
	commandCld:                handleCalendar,
	commandNxRc:               handleNextRace,
	commandConsStFull:         handleConstructorStandings,
	commandConsSt:             handleConstructorStandings,
	commandLstRc:              handleLastRace,
	commandLstGP:              handleLastGP,
	commandGPs:                handleGPs,
	commandDaysAfterRace:      handleDaysAfterRace,
	commandDaysAfterRaceСut:   handleDaysAfterRace,
	commandLstQual:            handleLastQual,
	commandClsKb:              handleCloseKeyboard,
	commandLvrsList:           handleLiveries,
	commandPredictionAdmin:    handlePredictionAdmin,
	commandPredictionUser:     handlePredictionUser,
	commandClosePrediction:    handleClosePrediction,
	commandPredictionResult:   handlePredictionResult,
	commandPredictionSummary:  handlePredictionSummary,
	commandPredictionRating:   handlePredictionRating,
	commandMyPredictionRating: handleMyPredictionRating,
	commandStartCheckStream:   handleCheckStream,
	commandEndCheckStream:     handleCheckStream,
}

// payloadHandlers — карта команд из payload (кнопки)
var payloadHandlers = map[command]messageHandlerFunc{
	commandRaceRes: handleRaceRes,
	commandQualRes: handleQualRes,
	commandSprRes:  handleSprRes,
}

// eventHandlers — карта event-команд
var eventHandlers = map[eventCommand]eventHandlerFunc{
	commandGpList1: handleGpListPage,
	commandGpList2: handleGpListPage,
	commandGpList3: handleGpListPage,
	commandGpInfo:  handleGpInfo,
}

// ---------- Вспомогательные функции ----------

func marshalKeyboard(kb Kb) (*string, error) {
	jsKb, err := json.Marshal(kb)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal keyboard: %w", err)
	}
	strKb := string(jsKb)
	return &strKb, nil
}

func sendEventAnswerAndDelete(log *slog.Logger, vk *VkAPI, peerID int, eventID string, userID int, msgConvID int, label string) {
	evResp, err := sendEventMessageToUser(vk.lp.VK, peerID, eventID, userID)
	if err != nil {
		log.Error("failed to send event answer", slog.Int("peer_id", peerID), slog.Any("error", err))
	} else {
		log.Info("Event sent", slog.Int("response", evResp))
	}

	err = deleteMessages(vk.lp.VK, []int{msgConvID}, peerID, true)
	if err != nil {
		log.Error("failed to delete messages", slog.Any("error", err))
	}
}

// parsePredictionNumbers парсит строку "N1 N2 N3" в три числа
func parsePredictionNumbers(text string) (uint8, uint8, uint8, error) {
	parts := strings.Fields(text)
	if len(parts) != 3 {
		return 0, 0, 0, fmt.Errorf("error in parsing prediction")
	}
	top1, err := strconv.ParseUint(parts[0], 10, 8)
	if err != nil {
		return 0, 0, 0, fmt.Errorf("Top_1 position is not number")
	}

	top2, err := strconv.ParseUint(parts[1], 10, 8)
	if err != nil {
		return 0, 0, 0, fmt.Errorf("Top_2 position is not number")
	}

	top3, err := strconv.ParseUint(parts[2], 10, 8)
	if err != nil {
		return 0, 0, 0, fmt.Errorf("Top_3 position is not number")
	}

	return uint8(top1), uint8(top2), uint8(top3), nil
}

// ---------- Обработчики текстовых команд ----------

func handleHello(ctx handlerContext) error {
	msg := `Привет! Я бот, который делится информацией про F1 :)
Пока что я могу сказать тебе информацию только о текущем сезоне (но всё ещё впереди).
Для того чтобы подробнее познакомиться с моими возможностями напиши мне "Что умеешь?".

Приятного пользования :)`
	_, err := ctx.vk.sendAndLog(ctx.log, msg, ctx.obj.Message.PeerID, nil, nil, nil, "hello")
	return err
}

func handleHelp(ctx handlerContext) error {
	msg := `Команды которые я понимаю (могу их прочесть в твоём сообщении среди других слов):
• календарь сезона - список гран-при F1 текущего сезона
• кубок конструкторов или кк - текущее положение команд в кубке конструкторов
• личный зачёт - текущее положение гонщиков в личном зачёте
• следующая гонка - информация о следующем гран-при F1
• результат гонки - результат последней прошедшей гонки F1
• дней без формулы/F1 - количество дней с последней гонки F1

!Внимание! Информация, связанная с проведённой гонкой может обновляться не сразу.
Работаем над этим.`
	_, err := ctx.vk.sendAndLog(ctx.log, msg, ctx.obj.Message.PeerID, nil, nil, nil, "help")
	return err
}

func handleDriverStandings(ctx handlerContext) error {
	msg, err := ctx.vk.messageService.GetDriverStandingsMessage(ctx.userDate)
	if err != nil {
		ctx.log.Error("failed to get driver standings", slog.Any("error", err))
		return err
	}
	_, err = ctx.vk.sendAndLog(ctx.log, msg, ctx.obj.Message.PeerID, nil, nil, nil, "driverStandings")
	return err
}

func handleCalendar(ctx handlerContext) error {
	msg, err := ctx.vk.messageService.GetCalendarMessage(ctx.userDate.Year())
	if err != nil {
		ctx.log.Error("failed to get calendar", slog.Any("error", err))
		return err
	}
	_, err = ctx.vk.sendAndLog(ctx.log, msg, ctx.obj.Message.PeerID, nil, nil, nil, "calendar")
	return err
}

func handleNextRace(ctx handlerContext) error {
	msg, err := ctx.vk.messageService.GetNextRaceMessage(ctx.userDate, ctx.userTimestamp)
	if err != nil {
		ctx.log.Error("failed to get next race", slog.Any("error", err))
		return err
	}
	_, err = ctx.vk.sendAndLog(ctx.log, msg, ctx.obj.Message.PeerID, nil, nil, nil, "nextRace")
	return err
}

func handleConstructorStandings(ctx handlerContext) error {
	msg, err := ctx.vk.messageService.GetConstructorStandingsMessage(ctx.userDate)
	if err != nil {
		ctx.log.Error("failed to get constructor standings", slog.Any("error", err))
		return err
	}
	_, err = ctx.vk.sendAndLog(ctx.log, msg, ctx.obj.Message.PeerID, nil, nil, nil, "constructorStandings")
	return err
}

func handleLastRace(ctx handlerContext) error {
	msg, err := ctx.vk.messageService.GetRaceResultsMessage(ctx.userDate, ctx.raceID)
	if err != nil {
		ctx.log.Error("failed to get last race result", slog.Any("error", err))
		return err
	}
	_, err = ctx.vk.sendAndLog(ctx.log, msg, ctx.obj.Message.PeerID, nil, nil, nil, "lastRace")
	return err
}

func handleLastGP(ctx handlerContext) error {
	crsl, err := ctx.vk.messageService.GetGPInfoCarousel(ctx.userDate, ctx.raceID)
	if err != nil {
		ctx.log.Error("failed to get GP info carousel", slog.Any("error", err))
		return err
	}
	_, err = ctx.vk.sendAndLog(ctx.log, "Информация о гран-при:", ctx.obj.Message.PeerID, nil, &crsl, nil, "lastGP")
	return err
}

func handleGPs(ctx handlerContext) error {
	count, err := ctx.vk.messageService.GetCountOfRaces(ctx.userDate)
	if err != nil {
		ctx.log.Error("failed to get count of races", slog.Any("error", err))
		return err
	}

	kb, err := makeKeyboard(2, 4, 1, count, false)
	if err != nil {
		ctx.log.Error("failed to create keyboard", slog.Any("error", err))
		return err
	}

	strKb, err := marshalKeyboard(kb)
	if err != nil {
		ctx.log.Error("failed to marshal keyboard", slog.Any("error", err))
		return err
	}

	_, err = ctx.vk.sendAndLog(ctx.log, "Этапы F1:", ctx.obj.Message.PeerID, strKb, nil, nil, "GPs")
	return err
}

func handleDaysAfterRace(ctx handlerContext) error {
	msg, err := ctx.vk.messageService.GetCountDaysAfterRaceMessage(ctx.userDate, ctx.raceID)
	if err != nil {
		ctx.log.Error("failed to get days after race", slog.Int("peer_id", ctx.obj.Message.PeerID), slog.Any("error", err))
		return err
	}
	_, err = ctx.vk.sendAndLog(ctx.log, msg, ctx.obj.Message.PeerID, nil, nil, nil, "daysAfterRace")
	return err
}

func handleLastQual(ctx handlerContext) error {
	msg, err := ctx.vk.messageService.GetQualifyingResultsMessage(ctx.userDate, ctx.raceID)
	if err != nil {
		ctx.log.Error("failed to get qualifying result", slog.Any("error", err))
		return err
	}
	_, err = ctx.vk.sendAndLog(ctx.log, msg, ctx.obj.Message.PeerID, nil, nil, nil, "lastQual")
	return err
}

func handleCloseKeyboard(ctx handlerContext) error {
	kb, err := makeKeyboard(0, 0, 0, 0, false)
	if err != nil {
		ctx.log.Error("failed to create keyboard", slog.Any("error", err))
		return err
	}

	strKb, err := marshalKeyboard(kb)
	if err != nil {
		ctx.log.Error("failed to marshal keyboard", slog.Any("error", err))
		return err
	}

	msgResp, err := ctx.vk.sendAndLog(ctx.log, "Закрываю", ctx.obj.Message.PeerID, strKb, nil, nil, "closeKeyboard")
	if err != nil {
		return err
	}

	err = deleteMessages(ctx.vk.lp.VK, []int{msgResp[0].ConversationMessageID}, ctx.obj.Message.PeerID, true)
	if err != nil {
		ctx.log.Error("failed to delete messages", slog.Any("error", err))
	}
	return nil
}

func handleLiveries(ctx handlerContext) error {
	photo := "photo-219009582_457239026"
	_, err := ctx.vk.sendAndLog(ctx.log, "Ливреи машин 2024 года", ctx.obj.Message.PeerID, nil, nil, &photo, "liveries")
	return err
}

// ---------- Обработчики прогнозов ----------

// handlePredictionAdmin — открывает конкурс прогнозов (только для админа)
func handlePredictionAdmin(ctx handlerContext) error {
	if ctx.obj.Message.FromID != botAdminId {
		return nil
	}

	// Проверяем, нет ли уже активного раунда
	activeRace, err := ctx.vk.predictionService.GetActiveRace()
	if err != nil {
		ctx.log.Error("failed to check active race", slog.Any("error", err))
		return err
	}
	if activeRace != nil {
		_, err := ctx.vk.sendAndLog(ctx.log, fmt.Sprintf("Прогноз уже активен для гонки: %s", activeRace.RaceName), ctx.obj.Message.PeerID, nil, nil, nil, "predictionAdmin")
		return err
	}

	nxtRc, err := ctx.vk.messageService.GetNextRace(ctx.userDate, ctx.userTimestamp)
	if err != nil {
		ctx.log.Error("failed to get next race for prediction", slog.Any("error", err))
		return err
	}

	raceID := nxtRc.Season + "_" + nxtRc.Round

	// Создаём раунд в БД
	err = ctx.vk.predictionService.StartPrediction(raceID, nxtRc.RaceName)
	if err != nil {
		ctx.log.Error("failed to start prediction", slog.Any("error", err))
		return err
	}

	driversMessage, err := ctx.vk.messageService.GetDriversListMessage(ctx.userDate)
	if err != nil {
		ctx.log.Error("failed to get drivers list", slog.Any("error", err))
	}

	chats := []int{botAdminId}
	for _, chat := range chats {
		msg := "Открываем конкурс прогнозов! Укажите номера гонщиков, которые по вашему мнению займут первые 3 места по итогам будущей гонки. \n\nОтветьте на ЭТО сообщение в формате: №_топ1 №_топ2 №_топ3\nили напишите\n'/мойпрогноз №_топ1 №_топ2 №_топ3'\n\n\n Пример ответа: '23 17 29'"
		msgResp, err := ctx.vk.sendAndLog(ctx.log, msg, chat, nil, nil, nil, "predictionStart")
		if err != nil {
			continue
		}

		ctx.log.Info("Prediction started in chat", slog.Int("chat_id", chat), slog.Int("message_id", msgResp[0].ConversationMessageID))

		_, err = ctx.vk.sendAndLog(ctx.log, driversMessage, chat, nil, nil, nil, "predictionDriversList")
		if err != nil {
			continue
		}
	}
	ctx.log.Info("Prediction started globally", slog.String("race_id", raceID))
	return nil
}

// handlePredictionUser — принимает прогноз от пользователя через команду /мойпрогноз
func handlePredictionUser(ctx handlerContext) error {
	activeRace, err := ctx.vk.predictionService.GetActiveRace()
	if err != nil {
		ctx.log.Error("failed to get active race", slog.Any("error", err))
		return nil
	}
	if activeRace == nil {
		return nil
	}

	d1, d2, d3, err := parsePredictionNumbers(strings.TrimPrefix(ctx.messageText, "/мойпрогноз "))
	if err != nil {
		msg := "Неверный формат сообщения! Повторите попытку и укажите номера гонщиков, которые на ваш взгляд займут первые 3 места, в формате:\n\n/мойпрогноз №_топ1 №_топ2 №_топ3"
		ctx.vk.sendAndLog(ctx.log, msg, ctx.obj.Message.PeerID, nil, nil, nil, "predictionError")
		return nil
	}

	err = ctx.vk.predictionService.SubmitPrediction(ctx.obj.Message.FromID, activeRace.RaceID, d1, d2, d3)
	if err != nil {
		ctx.log.Error("failed to save prediction", slog.Any("error", err))
		msg := "Не удалось сохранить прогноз. Возможно вы уже отправляли прогноз на эту гонку."
		ctx.vk.sendAndLog(ctx.log, msg, ctx.obj.Message.PeerID, nil, nil, nil, "predictionSaveError")
		return nil
	}

	msg := fmt.Sprintf("Ваш прогноз принят: 1. №%d, 2. №%d, 3. №%d", d1, d2, d3)
	resp, err := ctx.vk.sendAndLog(ctx.log, msg, ctx.obj.Message.PeerID, nil, nil, nil, "predictionConfirm")
	if err == nil && len(resp) > 0 {
		ctx.log.Info("Prediction recorded", slog.Int("user_id", ctx.obj.Message.FromID), slog.Int("d1", int(d1)), slog.Int("d2", int(d2)), slog.Int("d3", int(d3)), slog.Int("message_id", resp[0].MessageID))
	}
	return nil
}

// handleClosePrediction — закрывает приём прогнозов (только для админа)
func handleClosePrediction(ctx handlerContext) error {
	if ctx.obj.Message.FromID != botAdminId {
		return nil
	}

	activeRace, err := ctx.vk.predictionService.GetActiveRace()
	if err != nil {
		ctx.log.Error("failed to get active race", slog.Any("error", err))
		return err
	}
	if activeRace == nil {
		_, err := ctx.vk.sendAndLog(ctx.log, "Нет активного конкурса прогнозов.", ctx.obj.Message.PeerID, nil, nil, nil, "closePrediction")
		return err
	}

	err = ctx.vk.predictionService.ClosePrediction(activeRace.RaceID)
	if err != nil {
		ctx.log.Error("failed to close prediction", slog.Any("error", err))
		return err
	}

	_, err = ctx.vk.sendAndLog(ctx.log, fmt.Sprintf("Приём прогнозов на гонку '%s' закрыт.", activeRace.RaceName), ctx.obj.Message.PeerID, nil, nil, nil, "closePrediction")
	return err
}

// handlePredictionResult — сохраняет реальные результаты гонки (только для админа)
func handlePredictionResult(ctx handlerContext) error {
	if ctx.obj.Message.FromID != botAdminId {
		return nil
	}

	// Ищем последнюю закрытую гонку без результатов
	allRaces, err := ctx.vk.predictionService.GetAllRaces()
	if err != nil {
		ctx.log.Error("failed to get all races", slog.Any("error", err))
		return err
	}

	var targetRace *models.PredictionRace
	for i := range allRaces {
		r := &allRaces[i]
		if !r.IsActive && r.Driver1 == nil {
			targetRace = r
			break
		}
	}

	if targetRace == nil {
		_, err := ctx.vk.sendAndLog(ctx.log, "Нет закрытых гонок без результатов.", ctx.obj.Message.PeerID, nil, nil, nil, "predictionResult")
		return err
	}

	// Парсим "N1 N2 N3" из сообщения (убираем префикс команды)
	text := strings.TrimPrefix(ctx.messageText, "результатпрогноза ")
	d1, d2, d3, err := parsePredictionNumbers(text)
	if err != nil {
		msg := "Неверный формат. Используйте: результатпрогноза №_топ1 №_топ2 №_топ3"
		ctx.vk.sendAndLog(ctx.log, msg, ctx.obj.Message.PeerID, nil, nil, nil, "predictionResultError")
		return nil
	}

	err = ctx.vk.predictionService.SetRaceResult(targetRace.RaceID, d1, d2, d3)
	if err != nil {
		ctx.log.Error("failed to set race result", slog.Any("error", err))
		return err
	}

	_, err = ctx.vk.sendAndLog(ctx.log, fmt.Sprintf("Результаты гонки '%s' сохранены: 1. №%d, 2. №%d, 3. №%d", targetRace.RaceName, d1, d2, d3), ctx.obj.Message.PeerID, nil, nil, nil, "predictionResult")
	return err
}

// handlePredictionSummary — подводит итоги прогнозов (только для админа)
func handlePredictionSummary(ctx handlerContext) error {
	if ctx.obj.Message.FromID != botAdminId {
		return nil
	}

	// Ищем последнюю закрытую гонку с результатами, но без подсчитанных очков
	allRaces, err := ctx.vk.predictionService.GetAllRaces()
	if err != nil {
		ctx.log.Error("failed to get all races", slog.Any("error", err))
		return err
	}

	var targetRace *models.PredictionRace
	for i := range allRaces {
		r := &allRaces[i]
		if !r.IsActive && r.Driver1 != nil {
			targetRace = r
			break
		}
	}

	if targetRace == nil {
		_, err := ctx.vk.sendAndLog(ctx.log, "Нет закрытых гонок с результатами.", ctx.obj.Message.PeerID, nil, nil, nil, "predictionSummary")
		return err
	}

	results, err := ctx.vk.predictionService.CalculateResults(targetRace.RaceID)
	if err != nil {
		ctx.log.Error("failed to calculate results", slog.Any("error", err))
		return err
	}

	if len(results) == 0 {
		_, err := ctx.vk.sendAndLog(ctx.log, fmt.Sprintf("На гонку '%s' не было прогнозов.", targetRace.RaceName), ctx.obj.Message.PeerID, nil, nil, nil, "predictionSummary")
		return err
	}

	// Формируем сообщение с результатами
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("🏁 Итоги конкурса прогнозов на гонку '%s'\n\n", targetRace.RaceName))
	sb.WriteString(fmt.Sprintf("Результаты гонки: 1. №%d, 2. №%d, 3. №%d\n\n", *targetRace.Driver1, *targetRace.Driver2, *targetRace.Driver3))

	for i, r := range results {
		place := i + 1
		sb.WriteString(fmt.Sprintf("%d. Пользователь [id%d|]\n", place, r.UserID))
		sb.WriteString(fmt.Sprintf("   Прогноз: %s\n", r.Details))
		sb.WriteString(fmt.Sprintf("   Очки: %d\n\n", r.Points))
	}

	_, err = ctx.vk.sendAndLog(ctx.log, sb.String(), ctx.obj.Message.PeerID, nil, nil, nil, "predictionSummary")
	return err
}

// handlePredictionRating — показывает таблицу лидеров
func handlePredictionRating(ctx handlerContext) error {
	leaderboard, err := ctx.vk.predictionService.GetLeaderboard()
	if err != nil {
		ctx.log.Error("failed to get leaderboard", slog.Any("error", err))
		return err
	}

	if len(leaderboard) == 0 {
		_, err := ctx.vk.sendAndLog(ctx.log, "Таблица лидеров пока пуста.", ctx.obj.Message.PeerID, nil, nil, nil, "predictionRating")
		return err
	}

	var sb strings.Builder
	sb.WriteString("🏆 Таблица лидеров прогнозов:\n\n")
	for i, st := range leaderboard {
		place := i + 1
		sb.WriteString(fmt.Sprintf("%d. Пользователь [id%d|] — %d очков (%d гонок, среднее: %.1f)\n",
			place, st.UserID, st.TotalPoints, st.TotalRaces, st.AvgPoints))
	}

	_, err = ctx.vk.sendAndLog(ctx.log, sb.String(), ctx.obj.Message.PeerID, nil, nil, nil, "predictionRating")
	return err
}

// handleMyPredictionRating — показывает статистику пользователя
func handleMyPredictionRating(ctx handlerContext) error {
	stats, err := ctx.vk.predictionService.GetUserStats(ctx.obj.Message.FromID)
	if err != nil {
		ctx.log.Error("failed to get user stats", slog.Any("error", err))
		return err
	}

	if stats.TotalRaces == 0 {
		_, err := ctx.vk.sendAndLog(ctx.log, "У вас пока нет прогнозов.", ctx.obj.Message.PeerID, nil, nil, nil, "myPredictionRating")
		return err
	}

	msg := fmt.Sprintf("📊 Ваша статистика прогнозов:\n\nВсего очков: %d\nУчастий в гонках: %d\nСреднее очков: %.1f\nЛучший результат: %d (гонка: %s)",
		stats.TotalPoints, stats.TotalRaces, stats.AvgPoints, stats.BestPoints, stats.BestRaceID)

	_, err = ctx.vk.sendAndLog(ctx.log, msg, ctx.obj.Message.PeerID, nil, nil, nil, "myPredictionRating")
	return err
}

func handleCheckStream(ctx handlerContext) error {
	streamCheckMu.Lock()
	defer streamCheckMu.Unlock()

	if ctx.messageText == "strstart" {
		if streamChecking {
			ctx.vk.sendAndLog(ctx.log, "Отслеживание уже запущено! Сначала завершите текущее отслеживание.", ctx.obj.Message.PeerID, nil, nil, nil, "checkStreamStart")
			return nil
		}

		lastVideo, err := getLastVideos(*ctx.myUsrVk, 1)
		if err != nil {
			ctx.log.Error("failed to get last videos", slog.Any("error", err))
			return err
		}
		lastStreamId = lastVideo[0].ID

		// Немедленная первая проверка: если она упадёт, тикер не запускаем
		if !checkStreamOnce(ctx.log, ctx.vk, ctx.myUsrVk, ctx.obj) {
			ctx.log.Info("Video check aborted on start")
			return nil
		}

		streamChecking = true
		streamQuit = make(chan bool)
		streamTicker = time.NewTicker(5 * time.Minute)

		ctx.vk.sendAndLog(ctx.log, "Команда принята!", ctx.obj.Message.PeerID, nil, nil, nil, "checkStreamStart")
		ctx.log.Info("Start video check")
		go checkLastStream(streamQuit, streamTicker, ctx.log, ctx.vk, ctx.myUsrVk, ctx.obj)
		return nil
	}

	// strend
	if !streamChecking {
		ctx.vk.sendAndLog(ctx.log, "Отслеживание уже остановлено! Нечего завершать.", ctx.obj.Message.PeerID, nil, nil, nil, "checkStreamEnd")
		return nil
	}

	streamQuit <- true
	ctx.vk.sendAndLog(ctx.log, "Команда принята!", ctx.obj.Message.PeerID, nil, nil, nil, "checkStreamEnd")
	return nil
}

// ---------- Обработчики команд из payload (кнопки) ----------

func handleRaceRes(ctx handlerContext) error {
	msg, err := ctx.vk.messageService.GetRaceResultsMessage(ctx.userDate, ctx.raceID)
	if err != nil {
		ctx.log.Error("failed to get race result", slog.Any("error", err))
		return err
	}
	_, err = ctx.vk.sendAndLog(ctx.log, msg, ctx.obj.Message.PeerID, nil, nil, nil, "raceRes")
	return err
}

func handleQualRes(ctx handlerContext) error {
	msg, err := ctx.vk.messageService.GetQualifyingResultsMessage(ctx.userDate, ctx.raceID)
	if err != nil {
		ctx.log.Error("failed to get qualifying result", slog.Any("error", err))
		return err
	}
	_, err = ctx.vk.sendAndLog(ctx.log, msg, ctx.obj.Message.PeerID, nil, nil, nil, "qualRes")
	return err
}

func handleSprRes(ctx handlerContext) error {
	msg := ctx.vk.messageService.GetSprintResultsMessage(ctx.userDate, ctx.raceID)
	_, err := ctx.vk.sendAndLog(ctx.log, msg, ctx.obj.Message.PeerID, nil, nil, nil, "sprRes")
	return err
}

// ---------- Обработчики event-команд ----------

func handleGpListPage(ctx eventHandlerContext) error {
	var numPage int
	switch ctx.payload {
	case "gpListPage_1":
		numPage = 1
	case "gpListPage_2":
		numPage = 2
	case "gpListPage_3":
		numPage = 3
	}

	newKeyboard, err := makeKeyboard(2, 4, numPage, 24, false)
	if err != nil {
		ctx.log.Error("failed to make keyboard", slog.Any("error", err))
		return err
	}

	strKb, err := marshalKeyboard(newKeyboard)
	if err != nil {
		ctx.log.Error("failed to marshal keyboard", slog.Any("error", err))
		return err
	}

	msgResp, err := ctx.vk.sendAndLog(ctx.log, "Обновление", ctx.obj.PeerID, strKb, nil, nil, fmt.Sprintf("gpListPage_%d", numPage))
	if err != nil {
		return err
	}

	sendEventAnswerAndDelete(ctx.log, ctx.vk, ctx.obj.PeerID, ctx.obj.EventID, ctx.obj.UserID, msgResp[0].ConversationMessageID, fmt.Sprintf("gpListPage_%d", numPage))
	return nil
}

func handleGpInfo(ctx eventHandlerContext) error {
	timeNow := time.Now()
	number := strings.Split(ctx.payload, "_")

	curRace, err := ctx.vk.eventService.GetGPInfoCarousel(timeNow, number[1])
	if err != nil {
		ctx.log.Error("failed to get GP info carousel", slog.Any("error", err))
		return err
	}

	ctx.vk.sendAndLog(ctx.log, "Информация о гран-при:", ctx.obj.PeerID, nil, &curRace, nil, "gpInfo")

	evResp, err := sendEventMessageToUser(ctx.vk.lp.VK, ctx.obj.PeerID, ctx.obj.EventID, ctx.obj.UserID)
	if err != nil {
		ctx.log.Error("failed to send event answer", slog.Int("peer_id", ctx.obj.PeerID), slog.Any("error", err))
	} else {
		ctx.log.Info("Event sent", slog.Int("response", evResp))
	}
	return nil
}

// ---------- Обработчик неизвестной команды (reply-прогноз) ----------

func handleUnknownWithPrediction(ctx handlerContext) error {
	activeRace, err := ctx.vk.predictionService.GetActiveRace()
	if err != nil {
		ctx.log.Error("failed to get active race", slog.Any("error", err))
		return nil
	}
	if activeRace == nil || ctx.obj.Message.ReplyMessage == nil {
		return nil
	}

	d1, d2, d3, err := parsePredictionNumbers(ctx.messageText)
	if err != nil {
		msg := "Неверный формат сообщения! Повторите попытку и укажите только номера гонщиков, которые на ваш взгляд займут первые 3 места."
		ctx.vk.sendAndLog(ctx.log, msg, ctx.obj.Message.PeerID, nil, nil, nil, "predictionParseError")
		return nil
	}

	err = ctx.vk.predictionService.SubmitPrediction(ctx.obj.Message.FromID, activeRace.RaceID, d1, d2, d3)
	if err != nil {
		ctx.log.Error("failed to save prediction", slog.Any("error", err))
		msg := "Не удалось сохранить прогноз. Возможно вы уже отправляли прогноз на эту гонку."
		ctx.vk.sendAndLog(ctx.log, msg, ctx.obj.Message.PeerID, nil, nil, nil, "predictionSaveError")
		return nil
	}

	msg := fmt.Sprintf("Ваш прогноз принят: 1. №%d, 2. №%d, 3. №%d", d1, d2, d3)

	resp, err := ctx.vk.sendAndLog(ctx.log, msg, ctx.obj.Message.PeerID, nil, nil, nil, "predictionReplyConfirm")
	if err == nil && len(resp) > 0 {
		ctx.log.Info("Prediction recorded", slog.Int("user_id", ctx.obj.Message.FromID), slog.Int("d1", int(d1)), slog.Int("d2", int(d2)), slog.Int("d3", int(d3)), slog.Int("message_id", resp[0].MessageID))
	}
	return nil
}

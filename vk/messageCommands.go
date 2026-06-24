package vk

import "regexp"

const (
	commandDrSt             command = `личн.*зач[её]т`
	commandCld              command = `календар.*сезона`
	commandNxRc             command = `следующ.*гонк`
	commandConsStFull       command = `куб.*конструктор`
	commandConsSt           command = `кк`
	commandLstRc            command = `результат.?\sгонк`
	commandLstQual          command = `результат.?\sквалы`
	commandLstSpr           command = `результат.?\sспринта`
	commandHelp             command = `что умеешь`
	commandHello            command = `начать`
	commandDaysAfterRace    command = `дней без формулы|F1`
	commandDaysAfterRaceСut command = `дбф`
	commandStartCheckStream command = `strstart`
	commandEndCheckStream   command = `strend`
	commandLstGP            command = `ласт гп`
	commandGPs              command = `этапы`
	commandRaceRes          command = `raceRes_\d{1,2}`
	commandQualRes          command = `qualRes_\d{1,2}`
	commandSprRes           command = `sprRes_\d{1,2}`
	commandClsKb            command = `выклкб`
	commandLvrsList         command = `ливреи`
	commandPredictionAdmin  command = `\Aпрогноз`
	commandPredictionUser   command = `мойпрогноз`
	commandUnknown          command = ``
)

type command string

// Предварительно скомпилированные регулярные выражения для всех команд
var compiledCommands = func() []struct {
	cmd   command
	regex *regexp.Regexp
} {
	patterns := []struct {
		cmd   command
		regex string
	}{
		{commandDrSt, `личн.*зач[её]т`},
		{commandCld, `календар.*сезона`},
		{commandNxRc, `следующ.*гонк`},
		{commandConsStFull, `куб.*конструктор`},
		{commandConsSt, `кк`},
		{commandLstRc, `результат.?\sгонк`},
		{commandLstQual, `результат.?\sквалы`},
		{commandLstSpr, `результат.?\sспринта`},
		{commandHelp, `что умеешь`},
		{commandHello, `начать`},
		{commandDaysAfterRace, `дней без формулы|F1`},
		{commandDaysAfterRaceСut, `дбф`},
		{commandStartCheckStream, `strstart`},
		{commandEndCheckStream, `strend`},
		{commandLstGP, `ласт гп`},
		{commandGPs, `этапы`},
		{commandRaceRes, `raceRes_\d{1,2}`},
		{commandQualRes, `qualRes_\d{1,2}`},
		{commandSprRes, `sprRes_\d{1,2}`},
		{commandClsKb, `выклкб`},
		{commandLvrsList, `ливреи`},
		{commandPredictionAdmin, `\Aпрогноз`},
		{commandPredictionUser, `мойпрогноз`},
	}

	result := make([]struct {
		cmd   command
		regex *regexp.Regexp
	}, 0, len(patterns))

	for _, p := range patterns {
		result = append(result, struct {
			cmd   command
			regex *regexp.Regexp
		}{cmd: p.cmd, regex: regexp.MustCompile(p.regex)})
	}
	return result
}()

func getCommand(message string) command {
	for _, entry := range compiledCommands {
		if entry.regex.MatchString(message) {
			return entry.cmd
		}
	}
	return commandUnknown
}

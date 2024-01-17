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
	commandStream           command = `strstart`
	commandLstGP            command = `ласт гп`
	commandGPs              command = `этапы`
	commandRaceRes          command = `raceRes_\d{1,2}`
	commandQualRes          command = `qualRes_\d{1,2}`
	commandSprRes           command = `sprRes_\d{1,2}`
	commandClsKb            command = `выклкб`
	commandUnknown          command = ``
)

type command string

func getCommand(message string) command {
	commands := []command{
		commandDrSt,
		commandCld,
		commandNxRc,
		commandConsStFull,
		commandConsSt,
		commandLstRc,
		commandLstQual,
		commandLstSpr,
		commandHelp,
		commandHello,
		commandDaysAfterRace,
		commandDaysAfterRaceСut,
		commandStream,
		commandLstGP,
		commandGPs,
		commandRaceRes,
		commandQualRes,
		commandSprRes,
		commandClsKb,
	}

	for _, command := range commands {
		matched, _ := regexp.MatchString(string(command), message)

		if matched {
			return command
		}
	}

	return commandUnknown
}

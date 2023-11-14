package vk

import "regexp"

const (
	commandGpInfo  eventCommand = `gpPage_\d{1,2}`
	commandGpList1 eventCommand = `gpListPage_1`
	commandGpList2 eventCommand = `gpListPage_2`
	commandGpList3 eventCommand = `gpListPage_3`
	commandNothing eventCommand = ``
)

type eventCommand string

func getEventCommand(event string) eventCommand {
	eventCommands := []eventCommand{
		commandGpInfo,
		commandGpList1,
		commandGpList2,
		commandGpList3,
		commandNothing,
	}

	for _, eventCommand := range eventCommands {
		matched, _ := regexp.MatchString(string(eventCommand), event)

		if matched {
			return eventCommand
		}
	}

	return commandNothing
}

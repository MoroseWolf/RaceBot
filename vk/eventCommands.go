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

// Предварительно скомпилированные регулярные выражения для event-команд
var compiledEventCommands = func() []struct {
	cmd   eventCommand
	regex *regexp.Regexp
} {
	patterns := []struct {
		cmd   eventCommand
		regex string
	}{
		{commandGpInfo, `gpPage_\d{1,2}`},
		{commandGpList1, `gpListPage_1`},
		{commandGpList2, `gpListPage_2`},
		{commandGpList3, `gpListPage_3`},
	}

	result := make([]struct {
		cmd   eventCommand
		regex *regexp.Regexp
	}, 0, len(patterns))

	for _, p := range patterns {
		result = append(result, struct {
			cmd   eventCommand
			regex *regexp.Regexp
		}{cmd: p.cmd, regex: regexp.MustCompile(p.regex)})
	}
	return result
}()

func getEventCommand(event string) eventCommand {
	for _, entry := range compiledEventCommands {
		if entry.regex.MatchString(event) {
			return entry.cmd
		}
	}
	return commandNothing
}

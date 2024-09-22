package commands

type Command struct {
	Name        string
	Description string
}

var Commands = struct {
	Start             Command
	Today             Command
	Week              Command
	Settings          Command
	SetDailyTime      Command
	SetWeeklyTime     Command
	SetReminderOffset Command
	SetWebCal         Command
}{
	Start:             Command{Name: "start", Description: "Start the bot"},
	SetWebCal:         Command{Name: "set_webcal", Description: "Set or update your WebCal link"},
	Today:             Command{Name: "today", Description: "Get today's lecture schedule"},
	Week:              Command{Name: "week", Description: "Get this week's lecture schedule"},
	Settings:          Command{Name: "settings", Description: "View and update your notification settings"},
	SetDailyTime:      Command{Name: "set_daily_time", Description: "Set the time for daily notifications"},
	SetWeeklyTime:     Command{Name: "set_weekly_time", Description: "Set the day and time for weekly notifications"},
	SetReminderOffset: Command{Name: "set_reminder_offset", Description: "Set the reminder offset in minutes"},
}

func GetAllCommands() []Command {
	return []Command{
		Commands.Start,
		Commands.Today,
		Commands.Week,
		Commands.Settings,
		Commands.SetDailyTime,
		Commands.SetWeeklyTime,
		Commands.SetReminderOffset,
		Commands.SetWebCal,
	}
}

func GetCommandByName(name string) (Command, bool) {
	for _, cmd := range GetAllCommands() {
		if cmd.Name == name {
			return cmd, true
		}
	}
	return Command{}, false
}

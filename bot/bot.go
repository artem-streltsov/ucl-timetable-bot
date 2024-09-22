package bot

import (
	"context"
	"database/sql"
	"fmt"
	"log"

	"github.com/artem-streltsov/ucl-timetable-bot/commands"
	"github.com/artem-streltsov/ucl-timetable-bot/common"
	"github.com/artem-streltsov/ucl-timetable-bot/handlers"
	"github.com/artem-streltsov/ucl-timetable-bot/scheduler"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

type Bot struct {
	api     common.BotAPI
	updates tgbotapi.UpdatesChannel
	state   map[int64]string
}

func InitBot(telegramToken string, db *sql.DB) (*Bot, error) {
	botAPI, err := tgbotapi.NewBotAPI(telegramToken)
	if err != nil {
		return nil, err
	}
	botAPI.Debug = false

	b := &Bot{
		api:   common.NewBotAPIWrapper(botAPI),
		state: make(map[int64]string),
	}

	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60
	b.updates = botAPI.GetUpdatesChan(u)

	if err := scheduler.RescheduleNotificationsOnStartup(b.api, db); err != nil {
		return nil, err
	}

	return b, nil
}

func (b *Bot) Run(ctx context.Context, db *sql.DB) error {
	log.Println("Bot is running...")

	for {
		select {
		case <-ctx.Done():
			log.Println("Shutting down bot...")
			return ctx.Err()
		case update := <-b.updates:
			if update.Message == nil {
				continue
			}

			chatID := update.Message.Chat.ID

			if update.Message.IsCommand() {
				command := update.Message.Command()

				switch command {
				case commands.Commands.Start.Name:
					b.state[chatID] = commands.Commands.SetWebCal.Name
					handlers.HandleStartCommand(b.api, chatID)
				case commands.Commands.Today.Name:
					handlers.HandleTodayCommand(b.api, db, chatID)
				case commands.Commands.Week.Name:
					handlers.HandleWeekCommand(b.api, db, chatID)
				case commands.Commands.Settings.Name:
					handlers.HandleSettingsCommand(b.api, db, chatID)
				case commands.Commands.SetDailyTime.Name:
					b.state[chatID] = commands.Commands.SetDailyTime.Name
					handlers.HandleSetDailyTimePrompt(b.api, chatID)
				case commands.Commands.SetWeeklyTime.Name:
					b.state[chatID] = commands.Commands.SetWeeklyTime.Name
					handlers.HandleSetWeeklyTimePrompt(b.api, chatID)
				case commands.Commands.SetReminderOffset.Name:
					b.state[chatID] = commands.Commands.SetReminderOffset.Name
					handlers.HandleSetReminderOffsetPrompt(b.api, chatID)
				case commands.Commands.SetWebCal.Name:
					b.state[chatID] = commands.Commands.SetWebCal.Name
					handlers.HandleSetWebCalPrompt(b.api, chatID)
				default:
					availableCommands := getAvailableCommandsMessage()
					msg := b.api.NewMessage(chatID, fmt.Sprintf("Unknown command. Available commands:\n%s", availableCommands))
					b.api.Send(msg)
				}
			} else if update.Message.Text != "" {
				if state, ok := b.state[chatID]; ok {
					switch state {
					case commands.Commands.SetDailyTime.Name:
						if handlers.HandleSetDailyTime(b.api, db, chatID, update.Message.Text) {
							delete(b.state, chatID)
						}
					case commands.Commands.SetWeeklyTime.Name:
						if handlers.HandleSetWeeklyTime(b.api, db, chatID, update.Message.Text) {
							delete(b.state, chatID)
						}
					case commands.Commands.SetReminderOffset.Name:
						if handlers.HandleSetReminderOffset(b.api, db, chatID, update.Message.Text) {
							delete(b.state, chatID)
						}
					case commands.Commands.SetWebCal.Name:
						if handlers.HandleWebCalLink(b.api, db, chatID, update.Message.Text) {
							delete(b.state, chatID)
						}
					}
				} else {
					availableCommands := getAvailableCommandsMessage()
					msg := b.api.NewMessage(chatID, fmt.Sprintf("Unknown command. Available commands:\n%s", availableCommands))
					b.api.Send(msg)
				}
			}
		}
	}
}

func getAvailableCommandsMessage() string {
	var commandList string
	for _, cmd := range commands.GetAllCommands() {
		commandList += fmt.Sprintf("/%s - %s\n", cmd.Name, cmd.Description)
	}
	return commandList
}

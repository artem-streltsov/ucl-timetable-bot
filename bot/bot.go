package bot

import (
	"context"
	"database/sql"
	"log"
	"strings"

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
				args := update.Message.CommandArguments()

				switch command {
				case "start":
					handlers.HandleStartCommand(b.api, chatID)
				case "today":
					handlers.HandleTodayCommand(b.api, db, chatID)
				case "week":
					handlers.HandleWeekCommand(b.api, db, chatID)
				case "settings":
					handlers.HandleSettingsCommand(b.api, db, chatID)
				case "set_daily_time":
					if args == "" {
						b.state[chatID] = "set_daily_time"
						handlers.HandleSetDailyTimePrompt(b.api, chatID)
					} else {
						if !handlers.HandleSetDailyTime(b.api, db, chatID, args) {
							b.state[chatID] = "set_daily_time"
						} else {
							delete(b.state, chatID)
						}
					}
				case "set_weekly_time":
					if args == "" {
						b.state[chatID] = "set_weekly_time"
						handlers.HandleSetWeeklyTimePrompt(b.api, chatID)
					} else {
						if !handlers.HandleSetWeeklyTime(b.api, db, chatID, args) {
							b.state[chatID] = "set_weekly_time"
						} else {
							delete(b.state, chatID)
						}
					}
				case "set_reminder_offset":
					if args == "" {
						b.state[chatID] = "set_reminder_offset"
						handlers.HandleSetReminderOffsetPrompt(b.api, chatID)
					} else {
						if !handlers.HandleSetReminderOffset(b.api, db, chatID, args) {
							b.state[chatID] = "set_reminder_offset"
						} else {
							delete(b.state, chatID)
						}
					}
				default:
					msg := b.api.NewMessage(chatID, "Unknown command. Please use /settings to see available commands.")
					b.api.Send(msg)
				}
			} else if update.Message.Text != "" {
				if state, ok := b.state[chatID]; ok {
					switch state {
					case "set_daily_time":
						if !handlers.HandleSetDailyTime(b.api, db, chatID, update.Message.Text) {
							b.state[chatID] = "set_daily_time"
						} else {
							delete(b.state, chatID)
						}
					case "set_weekly_time":
						if !handlers.HandleSetWeeklyTime(b.api, db, chatID, update.Message.Text) {
							b.state[chatID] = "set_weekly_time"
						} else {
							delete(b.state, chatID)
						}
					case "set_reminder_offset":
						if !handlers.HandleSetReminderOffset(b.api, db, chatID, update.Message.Text) {
							b.state[chatID] = "set_reminder_offset"
						} else {
							delete(b.state, chatID)
						}
					}
				} else if strings.HasPrefix(strings.ToLower(update.Message.Text), "webcal://") {
					handlers.HandleWebCalLink(b.api, db, chatID, update.Message.Text)
					if err := setDefaultPreferences(db, chatID); err != nil {
						log.Printf("Error setting default preferences: %v", err)
					}
				} else {
					msg := b.api.NewMessage(chatID, "I'm expecting a WebCal link. Please send a link starting with 'webcal://'.")
					b.api.Send(msg)
				}
			}
		}
	}
}

func setDefaultPreferences(db *sql.DB, chatID int64) error {
	dailyTime := "07:00"
	weeklyTime := "SUN 18:00"
	reminderOffset := 30

	return handlers.UpdateUserPreference(db, chatID, "all", dailyTime, weeklyTime, reminderOffset)
}

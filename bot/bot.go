package bot

import (
	"context"
	"database/sql"
	"log"

	"github.com/artem-streltsov/ucl-timetable-bot/common"
	"github.com/artem-streltsov/ucl-timetable-bot/handlers"
	"github.com/artem-streltsov/ucl-timetable-bot/scheduler"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

type Bot struct {
	api     common.BotAPI
	updates tgbotapi.UpdatesChannel
}

func InitBot(telegramToken string, db *sql.DB) (*Bot, error) {
	botAPI, err := tgbotapi.NewBotAPI(telegramToken)
	if err != nil {
		log.Fatalf("Failed to initialize bot: %v", err)
	}
	botAPI.Debug = false

	b := &Bot{
		api: common.NewBotAPIWrapper(botAPI),
	}

	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60
	b.updates = botAPI.GetUpdatesChan(u)

	if err := scheduler.RescheduleNotificationsOnStartup(b.api, db); err != nil {
		return nil, err
	}

	return b, nil
}

func (b *Bot) Run(ctx context.Context, db *sql.DB) {
	log.Println("Bot is running...")

	for {
		select {
		case <-ctx.Done():
			log.Println("Shutting down bot...")
			return
		case update := <-b.updates:
			if update.Message == nil {
				continue
			}

			chatID := update.Message.Chat.ID

			if update.Message.IsCommand() {
				switch update.Message.Command() {
				case "start":
					handlers.HandleStartCommand(b.api, chatID)
				}
			} else if update.Message.Text != "" {
				handlers.HandleWebCalLink(b.api, db, chatID, update.Message.Text)
			}
		}
	}
}

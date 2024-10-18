package bot

import (
	"context"
	"log"

	"github.com/artem-streltsov/ucl-timetable-bot/database"
	"github.com/artem-streltsov/ucl-timetable-bot/handlers"
	"github.com/artem-streltsov/ucl-timetable-bot/scheduler"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

type Bot struct {
	api       *tgbotapi.BotAPI
	db        *database.DB
	handler   *handlers.Handler
	updates   tgbotapi.UpdatesChannel
	scheduler *scheduler.Scheduler
}

func NewBot(token string, db *database.DB) (*Bot, error) {
	api, err := tgbotapi.NewBotAPI(token)
	if err != nil {
		return nil, err
	}

	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60

	updates := api.GetUpdatesChan(u)

	scheduler := scheduler.NewScheduler(api, db)
	scheduler.ScheduleAll()

	handler := handlers.NewHandler(api, db, scheduler)

	return &Bot{
		api:       api,
		db:        db,
		handler:   handler,
		updates:   updates,
		scheduler: scheduler,
	}, nil
}

func (b *Bot) Run(ctx context.Context) error {
	log.Println("Bot started")
	for {
		select {
		case update, ok := <-b.updates:
			if !ok {
				return nil
			}
			if update.Message == nil {
				continue
			}
			username := update.Message.From.UserName
			if update.Message.IsCommand() {
				cmd := update.Message.Command()
				b.handler.HandleCommand(update.Message.Chat.ID, cmd, username)
			} else {
				b.handler.HandleMessage(update.Message.Chat.ID, update.Message.Text, username)
			}
		case <-ctx.Done():
			log.Println("Context canceled, stopping bot")
			b.api.StopReceivingUpdates()
			b.scheduler.StopAll()
			return nil
		}
	}
}

func (b *Bot) Stop() {
	b.api.StopReceivingUpdates()
	b.scheduler.StopAll()
}

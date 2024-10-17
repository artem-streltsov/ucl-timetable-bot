package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/artem-streltsov/ucl-timetable-bot/bot"
	"github.com/artem-streltsov/ucl-timetable-bot/config"
	"github.com/artem-streltsov/ucl-timetable-bot/database"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	db, err := database.New(cfg.DBPath)
	if err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	defer db.Close()

	ctx, cancel := context.WithCancel(context.Background())

	botInstance, err := bot.NewBot(cfg.TelegramBotToken, db)
	if err != nil {
		log.Fatalf("Failed to initialize bot: %v", err)
	}

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigChan
		log.Println("Received shutdown signal")
		cancel()
	}()

	if err := botInstance.Run(ctx); err != nil {
		log.Fatalf("Bot stopped with error: %v", err)
	}

	log.Println("Bot has been shut down")
}

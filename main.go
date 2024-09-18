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
	_ "github.com/mattn/go-sqlite3"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	db, err := database.InitDB(cfg.DBPath)
	if err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	defer db.Close()

	bot, err := bot.InitBot(cfg.TelegramBotToken, db)
	if err != nil {
		log.Fatalf("Failed to initialize bot: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		log.Println("Starting the bot...")
		if err := bot.Run(ctx, db); err != nil {
			log.Printf("Error running bot: %v", err)
			cancel()
		}
	}()

	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, syscall.SIGINT, syscall.SIGTERM)

	select {
	case sig := <-signalChan:
		log.Printf("Received signal: %v", sig)
	case <-ctx.Done():
		log.Println("Context canceled")
	}

	log.Println("Shutting down gracefully...")
}

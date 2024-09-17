package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/artem-streltsov/ucl-timetable-bot/bot"
	"github.com/artem-streltsov/ucl-timetable-bot/database"
	"github.com/joho/godotenv"
	_ "github.com/mattn/go-sqlite3"
)

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Fatalf("Error loading .env file: %v", err)
	}

	telegramToken := os.Getenv("TELEGRAM_BOT_TOKEN")
	if telegramToken == "" {
		log.Fatalf("TELEGRAM_BOT_TOKEN not found in environment variables")
	}

	dbPath := os.Getenv("DB_PATH")
	if dbPath == "" {
		log.Fatalf("DB_PATH not found in environment variables")
	}

	db, err := database.InitDB(dbPath)
	if err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	defer db.Close()

	bot, err := bot.InitBot(telegramToken, db)
	if err != nil {
		log.Fatalf("Failed to initialize bot: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		log.Println("Starting the bot...")
		bot.Run(ctx, db)
	}()

	sig := <-signalChan
	log.Printf("Received signal: %v", sig)

	cancel()
	log.Println("Bot shut down successfully. Exiting program.")
}

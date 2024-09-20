package database

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/artem-streltsov/ucl-timetable-bot/config"
)

type BackupManager struct {
	DB     *sql.DB
	Config *config.Config
}

func InitBackupManager(db *sql.DB, cfg *config.Config) (*BackupManager, error) {
	if db == nil {
		return nil, fmt.Errorf("database is nil")
	}
	if cfg == nil {
		return nil, fmt.Errorf("config is nil")
	}
	return &BackupManager{
		DB:     db,
		Config: cfg,
	}, nil
}

func (bm *BackupManager) StartBackups() {
	go func() {
		for {
			if err := bm.PerformBackup(); err != nil {
				fmt.Printf("Error performing backup: %v\n", err)
			}
			time.Sleep(time.Duration(bm.Config.BackupIntervalHours) * time.Hour)
		}
	}()
}

func (bm *BackupManager) PerformBackup() error {
	backupDir := bm.Config.BackupDirectory

	if err := os.MkdirAll(backupDir, os.ModePerm); err != nil {
		return fmt.Errorf("failed to create backup directory: %w", err)
	}

	timestamp := time.Now().Format("20060102_150405")
	backupFileName := fmt.Sprintf("backup_%s.db", timestamp)
	backupPath := filepath.Join(backupDir, backupFileName)

	if err := copyFile(bm.Config.DBPath, backupPath); err != nil {
		return fmt.Errorf("failed to create backup: %w", err)
	}

	fmt.Printf("Backup created successfully: %s\n", backupPath)
	return nil
}

func copyFile(src, dst string) error {
	input, err := os.ReadFile(src)
	if err != nil {
		return err
	}

	err = os.WriteFile(dst, input, 0644)
	if err != nil {
		return err
	}

	return nil
}

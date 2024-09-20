package database_test

import (
	"database/sql"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/artem-streltsov/ucl-timetable-bot/config"
	"github.com/artem-streltsov/ucl-timetable-bot/database"
	"github.com/stretchr/testify/assert"
)

func TestInitBackupManager(t *testing.T) {
	db, _ := sql.Open("sqlite3", ":memory:")
	defer db.Close()

	cfg := &config.Config{
		DBPath:              "test.db",
		BackupDirectory:     "test_backups",
		BackupIntervalHours: 24,
	}

	t.Run("Valid initialization", func(t *testing.T) {
		bm, err := database.InitBackupManager(db, cfg)
		assert.NoError(t, err)
		assert.NotNil(t, bm)
		assert.Equal(t, db, bm.DB)
		assert.Equal(t, cfg, bm.Config)
	})

	t.Run("Nil database", func(t *testing.T) {
		bm, err := database.InitBackupManager(nil, cfg)
		assert.Error(t, err)
		assert.Nil(t, bm)
		assert.Contains(t, err.Error(), "database is nil")
	})

	t.Run("Nil config", func(t *testing.T) {
		bm, err := database.InitBackupManager(db, nil)
		assert.Error(t, err)
		assert.Nil(t, bm)
		assert.Contains(t, err.Error(), "config is nil")
	})
}

func TestPerformBackup(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "backup_test")
	assert.NoError(t, err)
	defer os.RemoveAll(tempDir)

	dbPath := filepath.Join(tempDir, "test.db")
	err = os.WriteFile(dbPath, []byte("test database content"), 0644)
	assert.NoError(t, err)

	db, _ := sql.Open("sqlite3", dbPath)
	defer db.Close()

	cfg := &config.Config{
		DBPath:              dbPath,
		BackupDirectory:     filepath.Join(tempDir, "backups"),
		BackupIntervalHours: 24,
	}

	bm, err := database.InitBackupManager(db, cfg)
	assert.NoError(t, err)

	t.Run("Successful backup", func(t *testing.T) {
		err := bm.PerformBackup()
		assert.NoError(t, err)

		files, err := os.ReadDir(cfg.BackupDirectory)
		assert.NoError(t, err)
		assert.Len(t, files, 1)

		backupContent, err := os.ReadFile(filepath.Join(cfg.BackupDirectory, files[0].Name()))
		assert.NoError(t, err)
		assert.Equal(t, []byte("test database content"), backupContent)
	})

	t.Run("Invalid backup directory", func(t *testing.T) {
		invalidCfg := &config.Config{
			DBPath:              dbPath,
			BackupDirectory:     "/invalid/directory",
			BackupIntervalHours: 24,
		}
		invalidBm, _ := database.InitBackupManager(db, invalidCfg)

		err := invalidBm.PerformBackup()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to create backup directory")
	})
}

type TestBackupManager struct {
	database.BackupManager
	backupCount int
	mu          sync.Mutex
}

func (tbm *TestBackupManager) PerformBackup() error {
	tbm.mu.Lock()
	defer tbm.mu.Unlock()
	tbm.backupCount++
	return nil
}

func (tbm *TestBackupManager) GetBackupCount() int {
	tbm.mu.Lock()
	defer tbm.mu.Unlock()
	return tbm.backupCount
}

func TestStartBackups(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "backup_test")
	assert.NoError(t, err)
	defer os.RemoveAll(tempDir)

	dbPath := filepath.Join(tempDir, "test.db")
	err = os.WriteFile(dbPath, []byte("test database content"), 0644)
	assert.NoError(t, err)

	db, _ := sql.Open("sqlite3", dbPath)
	defer db.Close()

	backupDir := filepath.Join(tempDir, "backups")
	cfg := &config.Config{
		DBPath:              dbPath,
		BackupDirectory:     backupDir,
		BackupIntervalHours: 1,
	}

	bm, err := database.InitBackupManager(db, cfg)
	assert.NoError(t, err)

	bm.StartBackups()

	timeout := time.After(2 * time.Second)
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			files, err := os.ReadDir(backupDir)
			assert.NoError(t, err)
			if len(files) >= 1 {
				assert.GreaterOrEqual(t, len(files), 1)
				t.Logf("Backup occurred. Number of backup files: %d", len(files))

				backupContent, err := os.ReadFile(filepath.Join(backupDir, files[0].Name()))
				assert.NoError(t, err)
				assert.Equal(t, []byte("test database content"), backupContent)

				return
			}
		case <-timeout:
			files, err := os.ReadDir(backupDir)
			assert.NoError(t, err)
			t.Fatalf("Timeout waiting for backup to occur. Number of backup files: %d", len(files))
		}
	}
}

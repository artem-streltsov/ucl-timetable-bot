CREATE TABLE IF NOT EXISTS users (
    chatID INTEGER PRIMARY KEY,
    webcalURL TEXT,
    lastDailySent DATETIME,
    lastWeeklySent DATETIME,
    dailyNotificationTime TEXT DEFAULT '07:00',
    weeklyNotificationTime TEXT DEFAULT 'SUN 18:00',
    reminderOffset INTEGER DEFAULT 30
);

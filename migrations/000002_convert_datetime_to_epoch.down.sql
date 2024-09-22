ALTER TABLE users ADD COLUMN lastDailySent_datetime DATETIME;
UPDATE users SET lastDailySent_datetime = datetime(lastDailySent, 'unixepoch');
ALTER TABLE users DROP COLUMN lastDailySent;
ALTER TABLE users RENAME COLUMN lastDailySent_datetime TO lastDailySent;

ALTER TABLE users ADD COLUMN lastWeeklySent_datetime DATETIME;
UPDATE users SET lastWeeklySent_datetime = datetime(lastWeeklySent, 'unixepoch');
ALTER TABLE users DROP COLUMN lastWeeklySent;
ALTER TABLE users RENAME COLUMN lastWeeklySent_datetime TO lastWeeklySent;
ALTER TABLE users ADD COLUMN lastDailySent_epoch INTEGER;
ALTER TABLE users ADD COLUMN lastWeeklySent_epoch INTEGER;

UPDATE users SET lastDailySent_epoch = strftime('%s', lastDailySent);
UPDATE users SET lastWeeklySent_epoch = strftime('%s', lastWeeklySent);

ALTER TABLE users DROP COLUMN lastDailySent;
ALTER TABLE users DROP COLUMN lastWeeklySent;

ALTER TABLE users RENAME COLUMN lastDailySent_epoch TO lastDailySent;
ALTER TABLE users RENAME COLUMN lastWeeklySent_epoch TO lastWeeklySent;

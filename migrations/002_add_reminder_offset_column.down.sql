BEGIN TRANSACTION;
CREATE TABLE users_new (
    chat_id INTEGER PRIMARY KEY, 
    webcal_url TEXT, 
    daily_time TEXT, 
    weekly_time TEXT
);
INSERT INTO users_new (chat_id, webcal_url, daily_time, weekly_time)
SELECT chat_id, webcal_url, daily_time, weekly_time FROM users;
DROP TABLE users;
ALTER TABLE users_new RENAME TO users;
COMMIT;

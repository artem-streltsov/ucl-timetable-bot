ALTER TABLE users ADD COLUMN reminder_offset TEXT;
UPDATE users SET reminder_offset = '15' WHERE reminder_offset IS NULL;

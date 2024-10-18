CREATE TABLE IF NOT EXISTS friends (
    user_id1 INT NOT NULL,
    user_id2 INT NOT NULL,
    PRIMARY KEY (user_id1, user_id2),
    CONSTRAINT fk_user1 FOREIGN KEY (user_id1) REFERENCES users(id),
    CONSTRAINT fk_user2 FOREIGN KEY (user_id2) REFERENCES users(id),
    CONSTRAINT chk_user_order CHECK (user_id1 < user_id2)
);

CREATE TABLE IF NOT EXISTS friend_requests (
    requestor_id INT NOT NULL,
    requestee_id INT NOT NULL,
    PRIMARY KEY (requestor_id, requestee_id),
    CONSTRAINT fk_requestor FOREIGN KEY (requestor_id) REFERENCES users(id),
    CONSTRAINT fk_requestee FOREIGN KEY (requestee_id) REFERENCES users(id),
    CONSTRAINT chk_requestor_requestee CHECK (requestor_id <> requestee_id)
);

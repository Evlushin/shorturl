CREATE TABLE IF NOT EXISTS shorteners (
                            ID VARCHAR(36) NOT NULL PRIMARY KEY,
                            URL TEXT NOT NULL,
                            created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP
);
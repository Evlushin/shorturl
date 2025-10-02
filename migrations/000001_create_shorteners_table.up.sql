CREATE TABLE IF NOT EXISTS shorteners (
                            ID VARCHAR(36) NOT NULL PRIMARY KEY,
                            URL TEXT NOT NULL,
                            USER_ID VARCHAR(36) NOT NULL,
                            CREATED_AT TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP
);

ALTER TABLE shorteners ADD CONSTRAINT unique_url_user UNIQUE (URL, USER_ID);
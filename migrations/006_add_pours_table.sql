-- Add pours table for tracking multiple water pours during brewing

CREATE TABLE IF NOT EXISTS pours (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    brew_id INTEGER NOT NULL,
    pour_number INTEGER NOT NULL,
    water_amount INTEGER NOT NULL,
    time_seconds INTEGER NOT NULL,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (brew_id) REFERENCES brews(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_pours_brew_id ON pours(brew_id);

-- Add brewers table for brewing devices/methods
CREATE TABLE IF NOT EXISTS brewers (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT UNIQUE NOT NULL,
    description TEXT,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

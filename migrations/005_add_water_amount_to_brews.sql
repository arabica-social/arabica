-- Add water_amount column to brews table

ALTER TABLE brews ADD COLUMN water_amount INTEGER DEFAULT 0;

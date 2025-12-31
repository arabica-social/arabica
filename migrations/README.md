# Database Migrations

This directory contains SQL migration files for the Arabica database schema.

## How Migrations Work

- Migrations are run automatically when the application starts
- Each migration is tracked in the `schema_migrations` table
- Once a migration is applied, it will not run again
- Migrations are applied in order based on their filename

## Migration Files

- `001_initial.sql` - Initial schema with all tables (users, beans, roasters, grinders, brews)

## Adding a New Migration

When you need to change the database schema:

1. **Create a new migration file** with the next number:
   ```bash
   touch migrations/002_your_change_description.sql
   ```

2. **Write the SQL** for your schema change:
   ```sql
   -- Add a new column
   ALTER TABLE beans ADD COLUMN variety TEXT;
   
   -- Or create a new table
   CREATE TABLE IF NOT EXISTS brew_tags (
       id INTEGER PRIMARY KEY AUTOINCREMENT,
       brew_id INTEGER NOT NULL,
       tag TEXT NOT NULL,
       FOREIGN KEY (brew_id) REFERENCES brews(id)
   );
   ```

3. **Add the migration to the list** in `internal/database/sqlite/sqlite.go`:
   ```go
   migrations := []string{
       "migrations/001_initial.sql",
       "migrations/002_your_change_description.sql",  // Add this line
   }
   ```

4. **Rebuild and restart** the application:
   ```bash
   nix develop -c go build -o bin/arabica cmd/server/main.go
   ./bin/arabica
   ```

5. The migration will be applied automatically on startup

## Important Notes

- **Never modify existing migration files** after they've been deployed
- **Never delete migration files** that have been applied
- **Always add new migrations** with incrementing numbers
- **Test migrations** on a copy of your database first
- **Backup your database** before running new migrations in production

## Current Schema

The current schema (as of migration 001) includes:

- `users` - User accounts (default user for single-user mode)
- `beans` - Coffee bean information (name, origin, roast level, process, description, roaster)
- `roasters` - Coffee roasters (name, location, website)
- `grinders` - Coffee grinders (name, type, notes)
- `brews` - Brew records with all brew parameters and tasting notes
- `schema_migrations` - Tracks which migrations have been applied

## Preserving Your Data

Now that migrations are set up, your database will **NOT** be deleted when you:
- Rebuild the application
- Restart the server
- Add new features (as long as you use migrations for schema changes)

Simply run:
```bash
nix develop -c templ generate
nix develop -c go build -o bin/arabica cmd/server/main.go
./bin/arabica
```

Your data will persist across rebuilds!

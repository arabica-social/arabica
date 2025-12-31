.PHONY: build run dev templ css clean

# Build the application
build: templ css
	go build -o bin/arabica cmd/server/main.go

# Generate templ files
templ:
	templ generate

# Build CSS with Tailwind
css:
	tailwindcss -i web/static/css/style.css -o web/static/css/output.css --minify

# Run the application
run: build
	./bin/arabica

# Development mode with hot reload
dev:
	air

# Clean build artifacts
clean:
	rm -rf bin/
	rm -f arabica.db
	rm -f web/static/css/output.css
	find . -name "*_templ.go" -delete

# Initialize database (for testing)
init-db:
	rm -f arabica.db
	@echo "Database will be created on first run"

# Install development dependencies
install-deps:
	go install github.com/a-h/templ/cmd/templ@latest
	go install github.com/air-verse/air@latest

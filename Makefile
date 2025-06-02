.PHONY: dev server templ tailwind clean

# Variables
PORT ?= 8080
TEMPL_PROXY_PORT ?= 7331

# Run templ generation in watch mode with proxy
templ:
	templ generate --watch --proxy="http://localhost:$(PORT)" --open-browser=false

# Run air for Go hot reload
server:
	air

# Watch Tailwind CSS changes
tailwind:
	tailwindcss -i ./assets/css/input.css -o ./assets/css/output.css --watch

# Start development server with all watchers
dev:
	@echo Starting development servers...
	@echo Go server (air): http://localhost:$(PORT)
	@echo Templ proxy: http://localhost:$(TEMPL_PROXY_PORT)
	@$(MAKE) -j3 tailwind templ server

# Clean temporary files (Windows compatible)
clean:
	@if exist tmp rmdir /s /q tmp
	@if exist assets\css\output.css del /q assets\css\output.css
.PHONY: dev server templ tailwind clean

# Run templ generation in watch mode with proxy
templ:
	templ generate --watch --proxy="http://localhost:8080" --open-browser=false

# Run air for Go hot reload on port 8080
server:
	air

# Watch Tailwind CSS changes
tailwind:
	tailwindcss -i ./assets/css/input.css -o ./assets/css/output.css --watch

# Start development server with all watchers
dev:
	@echo "Starting development servers..."
	@echo "Go server (air): http://localhost:8080"
	@echo "Templ proxy: http://localhost:7331"
	make -j3 tailwind templ server

# Clean temporary files
clean:
	rm -rf tmp/
	rm -rf assets/css/output.css
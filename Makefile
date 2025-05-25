# Run templ generation in watch mode
templ:
	templ generate --watch --proxy="http://localhost:8080" --open-browser=false

# Run air for Go hot reload
server:
	air

# Watch Tailwind CSS changes
tailwind:
	tailwindcss -i ./assets/css/input.css -o ./assets/css/output.css --watch

# Start development server with all watchers
dev:
	make -j3 tailwind templ server
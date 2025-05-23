// middleware/security.go - Security and general middleware
package middleware

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"runtime/debug"
	"strings"
	"time"
)

// SecurityHeadersMiddleware adds security headers
func SecurityHeadersMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Prevent MIME type sniffing
		w.Header().Set("X-Content-Type-Options", "nosniff")

		// Prevent page from being displayed in a frame (clickjacking protection)
		w.Header().Set("X-Frame-Options", "DENY")

		// Enable XSS filtering
		w.Header().Set("X-XSS-Protection", "1; mode=block")

		// Referrer policy
		w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")

		// Content Security Policy (adjust as needed)
		csp := "default-src 'self'; " +
			"script-src 'self' 'unsafe-inline' https://cdn.tailwindcss.com https://cdnjs.cloudflare.com; " +
			"style-src 'self' 'unsafe-inline' https://cdn.tailwindcss.com; " +
			"font-src 'self' data: https:; " +
			"img-src 'self' data: https:; " +
			"connect-src 'self' https://graph.microsoft.com https://login.microsoftonline.com; " +
			"frame-ancestors 'none'"
		w.Header().Set("Content-Security-Policy", csp)

		// HSTS for production
		if getEnv("ENV", "development") == "production" {
			w.Header().Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains; preload")
		}

		next.ServeHTTP(w, r)
	})
}

// RateLimitMiddleware provides basic rate limiting
func RateLimitMiddleware(requestsPerMinute int) func(http.Handler) http.Handler {
	// Simple in-memory rate limiter (use Redis for production)
	type client struct {
		count     int
		lastReset time.Time
	}

	clients := make(map[string]*client)

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ip := getClientIP(r)
			now := time.Now()

			// Clean up old entries (simple cleanup)
			for k, v := range clients {
				if now.Sub(v.lastReset) > time.Minute {
					delete(clients, k)
				}
			}

			// Check rate limit
			clientData, exists := clients[ip]
			if !exists {
				clients[ip] = &client{count: 1, lastReset: now}
			} else {
				if now.Sub(clientData.lastReset) > time.Minute {
					clientData.count = 1
					clientData.lastReset = now
				} else {
					clientData.count++
				}

				if clientData.count > requestsPerMinute {
					w.Header().Set("Retry-After", "60")
					http.Error(w, "Rate limit exceeded", http.StatusTooManyRequests)
					return
				}
			}

			next.ServeHTTP(w, r)
		})
	}
}

// LoggingMiddleware provides enhanced logging
func LoggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		// Create a wrapper to capture response status and size
		wrapped := &responseWriter{
			ResponseWriter: w,
			statusCode:     http.StatusOK,
		}

		// Process request
		next.ServeHTTP(wrapped, r)

		// Log request details
		duration := time.Since(start)
		clientIP := getClientIP(r)
		userAgent := r.Header.Get("User-Agent")

		log.Printf(
			"%s - %s %s %s - %d - %d bytes - %v - %s",
			clientIP,
			r.Method,
			r.URL.Path,
			r.Proto,
			wrapped.statusCode,
			wrapped.size,
			duration,
			userAgent,
		)
	})
}

// RecoveryMiddleware provides panic recovery with proper logging
func RecoveryMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if err := recover(); err != nil {
				// Log the panic with stack trace
				stack := debug.Stack()
				log.Printf("PANIC: %v\n%s", err, stack)

				// Send appropriate error response
				if !headersSent(w) {
					w.Header().Set("Content-Type", "text/html; charset=utf-8")
					w.WriteHeader(http.StatusInternalServerError)

					if getEnv("ENV", "development") == "development" {
						// Show detailed error in development
						fmt.Fprintf(w, `
						<html>
							<head><title>Internal Server Error</title></head>
							<body>
								<h1>Internal Server Error</h1>
								<h2>Panic: %v</h2>
								<pre>%s</pre>
							</body>
						</html>`, err, stack)
					} else {
						// Show generic error in production
						fmt.Fprint(w, `
						<html>
							<head><title>Internal Server Error</title></head>
							<body>
								<h1>Internal Server Error</h1>
								<p>Something went wrong. Please try again later.</p>
							</body>
						</html>`)
					}
				}
			}
		}()

		next.ServeHTTP(w, r)
	})
}

// RequestIDMiddleware adds a unique request ID to each request
func RequestIDMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestID := generateRequestID()
		w.Header().Set("X-Request-ID", requestID)

		// Add to context for use in logging
		ctx := context.WithValue(r.Context(), "request_id", requestID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// TimeoutMiddleware adds request timeout
func TimeoutMiddleware(timeout time.Duration) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx, cancel := context.WithTimeout(r.Context(), timeout)
			defer cancel()

			// Channel to signal completion
			done := make(chan struct{})

			go func() {
				next.ServeHTTP(w, r.WithContext(ctx))
				close(done)
			}()

			select {
			case <-done:
				// Request completed normally
			case <-ctx.Done():
				// Request timed out
				if ctx.Err() == context.DeadlineExceeded {
					http.Error(w, "Request timeout", http.StatusRequestTimeout)
				}
			}
		})
	}
}

// CacheControlMiddleware sets appropriate cache headers
func CacheControlMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path

		// Set cache headers based on file type
		if strings.HasPrefix(path, "/static/") {
			// Static files - cache for 1 year
			w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
		} else if strings.HasSuffix(path, ".css") || strings.HasSuffix(path, ".js") {
			// CSS/JS files - cache for 1 week
			w.Header().Set("Cache-Control", "public, max-age=604800")
		} else if strings.HasSuffix(path, ".png") || strings.HasSuffix(path, ".jpg") || strings.HasSuffix(path, ".jpeg") || strings.HasSuffix(path, ".gif") || strings.HasSuffix(path, ".svg") {
			// Images - cache for 1 month
			w.Header().Set("Cache-Control", "public, max-age=2592000")
		} else {
			// Dynamic content - no cache
			w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
			w.Header().Set("Pragma", "no-cache")
			w.Header().Set("Expires", "0")
		}

		next.ServeHTTP(w, r)
	})
}

// CompressionMiddleware adds response compression
func CompressionMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check if client supports compression
		if !strings.Contains(r.Header.Get("Accept-Encoding"), "gzip") {
			next.ServeHTTP(w, r)
			return
		}

		// Skip compression for already compressed content
		if shouldSkipCompression(r.URL.Path) {
			next.ServeHTTP(w, r)
			return
		}

		// Note: Chi's middleware.Compress is better for production use
		// This is a simplified version for demonstration
		next.ServeHTTP(w, r)
	})
}

// MaintenanceMiddleware checks for maintenance mode
func MaintenanceMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check if maintenance mode is enabled
		if getEnv("MAINTENANCE_MODE", "false") == "true" {
			// Allow access to health check and admin routes during maintenance
			if r.URL.Path == "/health" || strings.HasPrefix(r.URL.Path, "/admin") {
				next.ServeHTTP(w, r)
				return
			}

			// Show maintenance page
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			w.Header().Set("Retry-After", "3600") // 1 hour
			w.WriteHeader(http.StatusServiceUnavailable)

			fmt.Fprint(w, `
			<html>
				<head>
					<title>Maintenance Mode</title>
					<meta charset="utf-8">
					<meta name="viewport" content="width=device-width, initial-scale=1">
				</head>
				<body style="font-family: Arial, sans-serif; text-align: center; padding: 50px;">
					<h1>System Maintenance</h1>
					<p>The system is currently undergoing maintenance. Please try again later.</p>
					<p>Expected completion: within 1 hour</p>
				</body>
			</html>`)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// Helper types and functions

type responseWriter struct {
	http.ResponseWriter
	statusCode int
	size       int
}

func (rw *responseWriter) WriteHeader(statusCode int) {
	rw.statusCode = statusCode
	rw.ResponseWriter.WriteHeader(statusCode)
}

func (rw *responseWriter) Write(b []byte) (int, error) {
	size, err := rw.ResponseWriter.Write(b)
	rw.size += size
	return size, err
}

// getClientIP extracts the real client IP
func getClientIP(r *http.Request) string {
	// Check X-Forwarded-For header
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		// Take the first IP (client IP)
		ips := strings.Split(xff, ",")
		return strings.TrimSpace(ips[0])
	}

	// Check X-Real-IP header
	if xri := r.Header.Get("X-Real-IP"); xri != "" {
		return xri
	}

	// Fall back to remote address
	return strings.Split(r.RemoteAddr, ":")[0]
}

// generateRequestID generates a unique request ID
func generateRequestID() string {
	return fmt.Sprintf("%d", time.Now().UnixNano())
}

// headersSent checks if headers have been sent
func headersSent(w http.ResponseWriter) bool {
	// This is a simplified check - in practice you might need a wrapper
	return false
}

// shouldSkipCompression checks if compression should be skipped
func shouldSkipCompression(path string) bool {
	// Skip compression for already compressed files
	extensions := []string{".gz", ".zip", ".png", ".jpg", ".jpeg", ".gif", ".pdf"}
	for _, ext := range extensions {
		if strings.HasSuffix(path, ext) {
			return true
		}
	}
	return false
}

// getEnv gets environment variable with default
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

package router

import (
	"net/http"
	"path/filepath"
	"strings"
)

// StaticOptions contains configuration for static file serving
type StaticOptions struct {
	// The URL prefix where files will be served from
	URLPrefix string
	// The filesystem directory to serve files from
	Directory string
	// Cache control header value (e.g., "max-age=3600")
	CacheControl string
	// List of file extensions to compress if browser supports it
	CompressExtensions []string
	// Whether to generate directory listings for directories without index.html
	DirectoryListing bool
	// Whether to set Content-Type headers based on file extensions
	SetContentType bool
}

// StaticFilesOption adds middleware to serve static files from a directory
func StaticFilesOption(urlPrefix, dir string) Option {
	return WithStaticFilesAdvanced(StaticOptions{
		URLPrefix:      urlPrefix,
		Directory:      dir,
		SetContentType: true,
		CacheControl:   "max-age=86400", // Default cache of 24 hours
		CompressExtensions: []string{
			".html", ".css", ".js", ".json", ".txt", ".xml", ".svg",
		},
	})
}

// WithStaticFiles is an alias to StaticFilesOption for backward compatibility
var WithStaticFiles = StaticFilesOption

// WithStaticFilesAdvanced adds middleware to serve static files with advanced options
func WithStaticFilesAdvanced(options StaticOptions) Option {
	return func(r *MoraRouter) {
		fileServer := http.FileServer(http.Dir(options.Directory))

		// Ensure prefix starts with /
		if !strings.HasPrefix(options.URLPrefix, "/") {
			options.URLPrefix = "/" + options.URLPrefix
		}

		// Ensure prefix ends with /
		if !strings.HasSuffix(options.URLPrefix, "/") {
			options.URLPrefix += "/"
		}

		// Strip the URL prefix when serving files
		handler := http.StripPrefix(options.URLPrefix, fileServer)

		// Register the handler for GET and HEAD requests
		r.Get(options.URLPrefix+"*path", func(w http.ResponseWriter, req *http.Request, p Params) {
			path := p["path"]

			// Handle content type if enabled
			if options.SetContentType {
				ext := filepath.Ext(path)
				switch ext {
				case ".css":
					w.Header().Set("Content-Type", "text/css")
				case ".js":
					w.Header().Set("Content-Type", "application/javascript")
				case ".json":
					w.Header().Set("Content-Type", "application/json")
				case ".svg":
					w.Header().Set("Content-Type", "image/svg+xml")
					// More types can be added as needed
				}
			}

			// Set cache control if provided
			if options.CacheControl != "" {
				w.Header().Set("Cache-Control", options.CacheControl)
			}

			// Serve the file using the standard file server
			handler.ServeHTTP(w, req)
		})
	}
}

// SPA serves a single-page app with client-side routing support
func WithSPA(urlPrefix, dir string, indexFile string) Option {
	if indexFile == "" {
		indexFile = "index.html"
	}

	return func(r *MoraRouter) {
		fs := http.Dir(dir)
		fileServer := http.FileServer(fs)

		// Ensure prefix starts with /
		if !strings.HasPrefix(urlPrefix, "/") {
			urlPrefix = "/" + urlPrefix
		}
		// Serve the main route and any sub-route
		r.Get(urlPrefix+"*path", func(w http.ResponseWriter, req *http.Request, p Params) {
			path := p["path"]

			// First check if the file exists
			if path != "" {
				if _, err := fs.Open(path); err == nil {
					// File exists, serve it directly
					fileServer.ServeHTTP(w, req)
					return
				}
			}

			// File doesn't exist, fallback to index.html
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			http.ServeFile(w, req, filepath.Join(dir, indexFile))
		})
	}
}

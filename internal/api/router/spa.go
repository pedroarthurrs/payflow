package router

import (
	"embed"
	"io/fs"
	"net/http"

	"github.com/gin-gonic/gin"
)

//go:embed all:web-dist
var embeddedFiles embed.FS

// spaFileSystem returns the sub-filesystem rooted at the embedded web-dist folder.
func spaFileSystem() http.FileSystem {
	fsys, err := fs.Sub(embeddedFiles, "web-dist")
	if err != nil {
		panic(err)
	}
	return http.FS(fsys)
}

// SPAHandler serves the embedded frontend static assets.
// For any path not found in the filesystem it falls back to index.html,
// allowing Vue Router to handle client-side navigation (history mode).
func SPAHandler() gin.HandlerFunc {
	fileServer := http.FileServer(spaFileSystem())

	return func(c *gin.Context) {
		fsys, err := fs.Sub(embeddedFiles, "web-dist")
		if err != nil {
			c.AbortWithStatus(http.StatusInternalServerError)
			return
		}

		// Check if the requested file exists; if not, fall back to index.html.
		path := c.Request.URL.Path
		if _, err := fs.Stat(fsys, path[1:]); err != nil {
			// File not found — serve index.html so Vue Router can handle the route.
			c.Request.URL.Path = "/"
		}

		fileServer.ServeHTTP(c.Writer, c.Request)
	}
}

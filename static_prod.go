//go:build !dev

package main

import (
	"embed"
	"io/fs"
	"log"
	"net/http"
	"strings"

	"github.com/labstack/echo/v4"
)

func registerStaticRoutes(e *echo.Echo, distFS embed.FS) {
	distSub, err := fs.Sub(distFS, "web/dist")
	if err != nil {
		log.Fatal(err)
	}
	fileServer := http.FileServer(http.FS(distSub))
	e.GET("/*", echo.WrapHandler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path
		if _, err := fs.Stat(distSub, strings.TrimPrefix(path, "/")); err == nil {
			fileServer.ServeHTTP(w, r)
			return
		}
		indexData, err := distFS.ReadFile("web/dist/index.html")
		if err != nil {
			w.WriteHeader(500)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(200)
		w.Write(indexData)
	})))
}

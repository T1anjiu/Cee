//go:build dev

package main

import (
	"embed"
	"log"
	"net/http/httputil"
	"net/url"

	"github.com/labstack/echo/v4"
)

func registerStaticRoutes(e *echo.Echo, distFS embed.FS) {
	viteURL, err := url.Parse("http://localhost:5173")
	if err != nil {
		log.Fatal(err)
	}
	proxy := httputil.NewSingleHostReverseProxy(viteURL)
	e.Any("/*", echo.WrapHandler(proxy))
}

package main

import (
	"embed"
	"flag"
	"log"
	"net/http"
	"time"

	"github.com/cee/watch-together/internal/handler"
	"github.com/cee/watch-together/internal/room"
	"github.com/cee/watch-together/internal/ws"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
)

//go:embed web/dist/*
var distFS embed.FS

func main() {
	listen := flag.String("listen", ":8080", "listen address")
	uploadDir := flag.String("upload-dir", "./uploads", "upload directory")
	maxRoomUploads := flag.Int64("max-room-uploads", 8*1024*1024*1024, "max room uploads in bytes")
	minFreeDisk := flag.Int64("min-free-disk", 5*1024*1024*1024, "minimum free disk space in bytes")
	uploadIdleTimeout := flag.Int("upload-idle-timeout", 30, "upload idle timeout in seconds")
	baseURL := flag.String("base-url", "", "base URL for join links")
	trustProxy := flag.Bool("trust-proxy", false, "trust X-Forwarded-* headers")
	flag.Parse()

	_ = baseURL
	_ = trustProxy

	uploadIdleDuration := time.Duration(*uploadIdleTimeout) * time.Second

	rm := room.NewManager(room.ManagerConfig{
		MaxRoomUploads: *maxRoomUploads,
		MinFreeDisk:    *minFreeDisk,
		UploadDir:      *uploadDir,
	}, uploadIdleDuration)

	hub := ws.NewHub(rm)

	rm.SetOnMediaReady(func(roomID, sourceURL, title string) {
		hub.BroadcastMediaReady(roomID, sourceURL, title)
	})

	e := echo.New()
	e.HideBanner = true

	e.Use(middleware.Recover())
	e.Use(middleware.Logger())
	e.Use(middleware.CORSWithConfig(middleware.CORSConfig{
		AllowOrigins: []string{"*"},
		AllowMethods: []string{http.MethodGet, http.MethodHead, http.MethodPut, http.MethodPatch, http.MethodPost, http.MethodDelete},
		AllowHeaders: []string{"Origin", "Content-Type", "Accept", "X-Member-Token"},
	}))

	e.POST("/api/rooms", handler.CreateRoom(rm, *baseURL))
	e.GET("/api/rooms/:id", handler.GetRoom(rm))
	e.POST("/api/rooms/:room_id/uploads", handler.CreateUpload(rm))
	e.PUT("/api/uploads/:upload_id/chunks/:index", handler.UploadChunk(rm))
	e.POST("/api/uploads/:upload_id/complete", handler.CompleteUpload(rm))
	e.DELETE("/api/uploads/:upload_id", handler.DeleteUpload(rm))
	e.GET("/media/:upload_id", handler.StreamMedia(rm))
	e.GET("/ws/:roomId", hub.HandleWebSocket())

	registerStaticRoutes(e, distFS)

	log.Printf("Cee watch-together listening on %s", *listen)
	if err := e.Start(*listen); err != nil {
		log.Fatal(err)
	}
}

package handler

import (
	"encoding/json"
	"io"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/cee/watch-together/internal/room"
	"github.com/cee/watch-together/internal/upload"
	"github.com/labstack/echo/v4"
)

type CreateUploadRequest struct {
	Filename   string `json:"filename"`
	Size       int64  `json:"size"`
	Mime       string `json:"mime"`
	ChunkSize  int64  `json:"chunk_size"`
}

type CreateUploadResponse struct {
	UploadID    string `json:"upload_id"`
	ChunkSize   int64  `json:"chunk_size"`
	TotalChunks int    `json:"total_chunks"`
	MaxSize     int64  `json:"max_size"`
}

type ChunkResponse struct {
	UploadID     string `json:"upload_id"`
	Index        int    `json:"index"`
	Received     int    `json:"received"`
	TotalChunks  int    `json:"total_chunks"`
}

type CompleteResponse struct {
	UploadID   string `json:"upload_id"`
	Title      string `json:"title"`
	Size       int64  `json:"size"`
	MediaType  string `json:"media_type"`
}

func CreateUpload(rm *room.Manager) echo.HandlerFunc {
	return func(c echo.Context) error {
		roomID := c.Param("room_id")
		if !rm.RoomExists(roomID) {
			return c.JSON(http.StatusNotFound, map[string]string{"error": "room_not_found"})
		}

		memberToken := c.Request().Header.Get("X-Member-Token")
		r := rm.GetRoom(roomID)
		if r == nil {
			return c.JSON(http.StatusNotFound, map[string]string{"error": "room_not_found"})
		}

		member := r.GetMemberByToken(memberToken)
		if member == nil {
			return c.JSON(http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		}

		var req CreateUploadRequest
		if err := json.NewDecoder(c.Request().Body).Decode(&req); err != nil {
			return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid_request"})
		}

		if req.Size <= 0 || req.Size > 4*1024*1024*1024 {
			return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid_size"})
		}

		um := rm.GetUploadManager()

		task, err := um.CreateUpload(roomID, member.ID, req.Filename, req.Size, req.ChunkSize)
		if err != nil {
			errMsg := err.Error()
			if strings.HasPrefix(errMsg, "disk_full") || strings.HasPrefix(errMsg, "quota_exceeded") ||
				strings.HasPrefix(errMsg, "upload_in_progress") || strings.HasPrefix(errMsg, "unsupported_media") {
				return c.JSON(http.StatusConflict, map[string]string{"error": errMsg})
			}
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": errMsg})
		}

		return c.JSON(http.StatusCreated, CreateUploadResponse{
			UploadID:    task.UploadID,
			ChunkSize:   task.ChunkSize,
			TotalChunks: task.TotalChunks,
			MaxSize:     4 * 1024 * 1024 * 1024,
		})
	}
}

func UploadChunk(rm *room.Manager) echo.HandlerFunc {
	return func(c echo.Context) error {
		uploadID := c.Param("upload_id")
		indexStr := c.Param("index")

		index, err := strconv.Atoi(indexStr)
		if err != nil {
			return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid_index"})
		}

		memberToken := c.Request().Header.Get("X-Member-Token")

		um := rm.GetUploadManager()
		task := um.GetTask(uploadID)
		if task == nil {
			return c.JSON(http.StatusNotFound, map[string]string{"error": "upload_not_found"})
		}

		// Verify member token
		r := rm.GetRoom(task.RoomID)
		if r == nil || r.GetMemberByToken(memberToken) == nil {
			return c.JSON(http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		}

		body, err := io.ReadAll(c.Request().Body)
		if err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": "read_error"})
		}

		if err := um.WriteChunk(uploadID, index, body); err != nil {
			errMsg := err.Error()
			if strings.HasPrefix(errMsg, "invalid_chunk_size") || strings.HasPrefix(errMsg, "invalid_chunk_index") {
				return c.JSON(http.StatusBadRequest, map[string]string{"error": errMsg})
			}
			if strings.HasPrefix(errMsg, "upload_not_found") || strings.HasPrefix(errMsg, "upload_not_ready") {
				return c.JSON(http.StatusNotFound, map[string]string{"error": errMsg})
			}
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": errMsg})
		}

		return c.JSON(http.StatusOK, ChunkResponse{
			UploadID:    uploadID,
			Index:       index,
			Received:    len(task.ReceivedChunks),
			TotalChunks: task.TotalChunks,
		})
	}
}

func CompleteUpload(rm *room.Manager) echo.HandlerFunc {
	return func(c echo.Context) error {
		uploadID := c.Param("upload_id")

		memberToken := c.Request().Header.Get("X-Member-Token")

		um := rm.GetUploadManager()
		task := um.GetTask(uploadID)
		if task == nil {
			return c.JSON(http.StatusNotFound, map[string]string{"error": "upload_not_found"})
		}

		// Verify member
		r := rm.GetRoom(task.RoomID)
		if r == nil || r.GetMemberByToken(memberToken) == nil {
			return c.JSON(http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		}

		filePath, size, err := um.CompleteUpload(uploadID)
		if err != nil {
			errMsg := err.Error()
			if strings.HasPrefix(errMsg, "assemble_failed") {
				return c.JSON(http.StatusConflict, map[string]string{"error": errMsg})
			}
			if strings.HasPrefix(errMsg, "file_type_rejected") {
				return c.JSON(http.StatusUnprocessableEntity, map[string]string{"error": errMsg})
			}
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": errMsg})
		}

		_ = filePath

		ext := ""
		if idx := strings.LastIndex(task.Filename, "."); idx >= 0 {
			ext = task.Filename[idx:]
		}
		mediaType := "direct"
		if ext == ".webm" {
			mediaType = "direct"
		}

		return c.JSON(http.StatusOK, CompleteResponse{
			UploadID:  uploadID,
			Title:     task.Filename,
			Size:      size,
			MediaType: mediaType,
		})
	}
}

func DeleteUpload(rm *room.Manager) echo.HandlerFunc {
	return func(c echo.Context) error {
		uploadID := c.Param("upload_id")

		memberToken := c.Request().Header.Get("X-Member-Token")

		um := rm.GetUploadManager()
		task := um.GetTask(uploadID)
		if task == nil {
			return c.JSON(http.StatusNotFound, map[string]string{"error": "upload_not_found"})
		}

		// Verify member
		r := rm.GetRoom(task.RoomID)
		if r == nil || r.GetMemberByToken(memberToken) == nil {
			return c.JSON(http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		}

		if err := um.CancelUpload(uploadID); err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
		}

		return c.JSON(http.StatusOK, map[string]string{"status": "cancelled"})
	}
}

func StreamMedia(rm *room.Manager) echo.HandlerFunc {
	return func(c echo.Context) error {
		uploadID := c.Param("upload_id")

		um := rm.GetUploadManager()
		task, diskPath := um.GetTaskWithFallback(uploadID)
		if task == nil && diskPath == "" {
			return c.JSON(http.StatusNotFound, map[string]string{"error": "media_not_found"})
		}

		if task != nil && task.State != upload.StateCompleted {
			return c.JSON(http.StatusNotFound, map[string]string{"error": "media_not_ready"})
		}

		var filePath string
		if task != nil {
			ext := ""
			if idx := strings.LastIndex(task.Filename, "."); idx >= 0 {
				ext = task.Filename[idx:]
			}
			filePath = filepath.Join(rm.GetUploadDir(), task.RoomID, task.UploadID+ext)
		} else {
			filePath = diskPath
		}

		c.Response().Header().Set("Accept-Ranges", "bytes")
		http.ServeFile(c.Response(), c.Request(), filePath)
		return nil
	}
}

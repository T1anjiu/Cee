package handler

import (
	"net/http"

	"github.com/cee/watch-together/internal/room"
	"github.com/labstack/echo/v4"
)

type CreateRoomResponse struct {
	RoomID  string `json:"room_id"`
	JoinURL string `json:"join_url"`
}

func CreateRoom(rm *room.Manager, baseURL string) echo.HandlerFunc {
	return func(c echo.Context) error {
		room, code := rm.CreateRoom()
		if room == nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{
				"error": "failed to create room",
			})
		}

		joinURL := baseURL + "/r/" + code
		if baseURL == "" {
			joinURL = "http://" + c.Request().Host + "/r/" + code
		}

		return c.JSON(http.StatusCreated, CreateRoomResponse{
			RoomID:  code,
			JoinURL: joinURL,
		})
	}
}

func GetRoom(rm *room.Manager) echo.HandlerFunc {
	return func(c echo.Context) error {
		id := c.Param("id")
		if rm.RoomExists(id) {
			return c.JSON(http.StatusOK, map[string]interface{}{
				"room_id": id,
				"exists":  true,
			})
		}
		return c.JSON(http.StatusNotFound, map[string]interface{}{
			"room_id": id,
			"exists":  false,
		})
	}
}

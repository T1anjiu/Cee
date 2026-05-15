package ws

import (
	"encoding/json"
	"net/url"
	"path"
	"strings"
	"sync"
	"time"

	"github.com/cee/watch-together/internal/model"
	"github.com/cee/watch-together/internal/room"
	"github.com/labstack/echo/v4"
	"github.com/olahol/melody"
)

type Hub struct {
	m          *melody.Melody
	rm         *room.Manager
	sessions   map[string]*melody.Session
	mu         sync.RWMutex
	rateLimits sync.Map
}

type rateLimiter struct {
	mu       sync.Mutex
	times    []time.Time
	maxCount int
	window   time.Duration
}

func newRateLimiter(maxCount int, window time.Duration) *rateLimiter {
	return &rateLimiter{
		times:    make([]time.Time, 0),
		maxCount: maxCount,
		window:   window,
	}
}

func (rl *rateLimiter) Allow() bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	cutoff := now.Add(-rl.window)
	valid := make([]time.Time, 0)
	for _, t := range rl.times {
		if t.After(cutoff) {
			valid = append(valid, t)
		}
	}
	if len(valid) >= rl.maxCount {
		rl.times = valid
		return false
	}
	rl.times = append(valid, now)
	return true
}

func NewHub(rm *room.Manager) *Hub {
	m := melody.New()
	m.Config.MaxMessageSize = 1024 * 1024

	h := &Hub{
		m:        m,
		rm:       rm,
		sessions: make(map[string]*melody.Session),
	}

	m.HandleConnect(func(s *melody.Session) {
		s.Set("connected", true)
	})

	m.HandleDisconnect(func(s *melody.Session) {
		h.handleDisconnect(s)
	})

	m.HandleMessage(func(s *melody.Session, msg []byte) {
		h.handleMessage(s, msg)
	})

	return h
}

func (h *Hub) HandleWebSocket() func(c echo.Context) error {
	return func(c echo.Context) error {
		roomId := c.Param("roomId")
		if !h.rm.RoomExists(roomId) {
			return c.JSON(404, map[string]string{"error": "room_not_found"})
		}
		return h.m.HandleRequestWithKeys(c.Response(), c.Request(), map[string]any{
			"room_id": roomId,
		})
	}
}

func (h *Hub) handleDisconnect(s *melody.Session) {
	h.mu.Lock()
	defer h.mu.Unlock()

	roomID, ok := s.Get("room_id")
	if !ok {
		return
	}
	memberID, ok := s.Get("member_id")
	if !ok {
		return
	}

	r := h.rm.GetRoom(roomID.(string))
	if r == nil {
		return
	}

	r.RemoveMember(memberID.(string))
	delete(h.sessions, memberID.(string))

	now := time.Now().UnixMilli()
	if t := r.GetPSM().RemoveBufferingMember(memberID.(string), now); t != nil {
		broadcastToRoom(h.m, roomID.(string), model.Message{
			Type: "player_update",
			Payload: model.PlayerUpdatePayload{
				Playing:   t.NewPlaying,
				Position:  t.NewPosition,
				UpdatedAt: t.NewUpdatedAt,
				Reason:    t.Reason,
			},
		})
	}

	broadcastToRoom(h.m, roomID.(string), model.Message{
		Type: "member_leave",
		Payload: model.MemberLeavePayload{
			ID: memberID.(string),
		},
	})
}

func (h *Hub) handleMessage(s *melody.Session, msg []byte) {
	var message model.Message
	if err := json.Unmarshal(msg, &message); err != nil {
		h.sendError(s, "invalid_message", "invalid JSON", message.ClientMsgID)
		return
	}

	switch message.Type {
	case "join":
		h.handleJoin(s, message)
	case "chat":
		h.handleChat(s, message)
	case "ping":
		h.handlePing(s, message)
	case "change_media":
		h.handleChangeMedia(s, message)
	case "play":
		h.handlePlay(s, message)
	case "pause":
		h.handlePause(s, message)
	case "seek":
		h.handleSeek(s, message)
	case "buffering":
		h.handleBuffering(s, message)
	case "heartbeat":
		h.handleHeartbeat(s, message)
	case "upload_progress":
		h.handleUploadProgress(s, message)
	case "upload_cancel":
		h.handleUploadCancel(s, message)
	default:
		h.sendError(s, "unknown_type", "unknown message type: "+message.Type, message.ClientMsgID)
	}
}

func (h *Hub) handleJoin(s *melody.Session, msg model.Message) {
	var payload model.JoinPayload
	b, _ := json.Marshal(msg.Payload)
	json.Unmarshal(b, &payload)

	if payload.Nickname == "" {
		h.sendError(s, "invalid_message", "nickname is required", msg.ClientMsgID)
		return
	}

	roomID, _ := s.Get("room_id")
	if roomID == nil {
		return
	}

	r := h.rm.GetRoom(roomID.(string))
	if r == nil {
		h.sendError(s, "room_not_found", "room does not exist", msg.ClientMsgID)
		return
	}

	member, resumed := r.AddMember(payload.Nickname, payload.MemberToken)

	h.mu.Lock()
	s.Set("room_id", roomID)
	s.Set("member_id", member.ID)
	s.Set("nickname", member.Nickname)
	h.sessions[member.ID] = s
	h.mu.Unlock()

	if !resumed {
		r.GetPSM().SetColdStart(time.Now().UnixMilli())
	}

	roomIDStr := roomID.(string)
	var joinType string
	if resumed {
		joinType = "member_resume"
	} else {
		joinType = "member_join"
	}

	broadcastToRoom(h.m, roomIDStr, model.Message{
		Type: joinType,
		Payload: model.MemberJoinPayload{
			ID:       member.ID,
			Nickname: member.Nickname,
		},
	})

	mediaState := r.GetMediaState()
	var mediaPayload *model.MediaState
	if mediaState != nil {
		mediaPayload = &model.MediaState{
			Kind:       mediaState.Kind,
			SourceURL:  mediaState.SourceURL,
			MediaType:  mediaState.MediaType,
			Title:      mediaState.Title,
			Status:     mediaState.Status,
			UploaderID: mediaState.UploaderID,
		}
	}

	ps := r.GetPlayerState()
	chatHistory := r.GetChatHistory()

	chatMsgs := make([]model.ChatMessage, len(chatHistory))
	for i, ch := range chatHistory {
		chatMsgs[i] = model.ChatMessage{
			SenderID: ch.SenderID,
			Nickname: ch.Nickname,
			Text:     ch.Text,
			Ts:       ch.Ts,
		}
	}

	stateResp := model.Message{
		Type: "room_state",
		Payload: model.RoomStatePayload{
			RoomID:    roomIDStr,
			SelfID:    member.ID,
			SelfToken: member.Token,
			Members:   toMemberInfos(r.ListMembers()),
			Media:     mediaPayload,
			Player: &model.PlayerState{
				Playing:   ps.Playing,
				Position:  ps.Position,
				UpdatedAt: ps.UpdatedAt,
			},
			ChatHistory: chatMsgs,
		},
	}

	data, _ := json.Marshal(stateResp)
	s.Write(data)
}

func (h *Hub) handleChangeMedia(s *melody.Session, msg model.Message) {
	roomID, _ := s.Get("room_id")
	if roomID == nil {
		h.sendError(s, "not_in_room", "must join room first", msg.ClientMsgID)
		return
	}

	var payload model.ChangeMediaPayload
	b, _ := json.Marshal(msg.Payload)
	json.Unmarshal(b, &payload)

	if payload.Kind != "url" && payload.Kind != "upload" {
		h.sendError(s, "invalid_message", "kind must be 'url' or 'upload'", msg.ClientMsgID)
		return
	}

	r := h.rm.GetRoom(roomID.(string))
	if r == nil {
		h.sendError(s, "room_not_found", "room does not exist", msg.ClientMsgID)
		return
	}

	now := time.Now().UnixMilli()
	roomIDStr := roomID.(string)

	if payload.Kind == "url" {
		if !isValidURL(payload.SourceURL) {
			h.sendError(s, "invalid_url", "invalid or unsupported URL", msg.ClientMsgID)
			return
		}

		mediaType := inferMediaType(payload.SourceURL, payload.Kind)
		ms := &room.MediaState{
			Kind:      payload.Kind,
			SourceURL: payload.SourceURL,
			MediaType: mediaType,
			Title:     extractTitle(payload.SourceURL, payload.Title),
			Status:    "ready",
			UpdatedAt: now,
		}
		r.SetMediaState(ms)
		r.GetPSM().ClearBufferingMembers(now)

		broadcastToRoom(h.m, roomIDStr, model.Message{
			Type: "media_update",
			Payload: model.MediaUpdatePayload{
				Kind:      payload.Kind,
				SourceURL: payload.SourceURL,
				MediaType: mediaType,
				Title:     ms.Title,
				Status:    "ready",
			},
		})
	} else {
		uploadTask := h.rm.GetUploadManager().GetTask(payload.UploadID)
		if uploadTask == nil {
			h.sendError(s, "upload_not_found", "upload task not found", msg.ClientMsgID)
			return
		}

		ms := &room.MediaState{
			Kind:      "upload",
			SourceURL: "/media/" + payload.UploadID,
			MediaType: "direct",
			Title:     extractTitle("", payload.Title),
			Status:    "uploading",
			UploaderID: func() string {
				memberID, _ := s.Get("member_id")
				return memberID.(string)
			}(),
			UpdatedAt: now,
		}
		r.SetMediaState(ms)
		r.GetPSM().ClearBufferingMembers(now)

		broadcastToRoom(h.m, roomIDStr, model.Message{
			Type: "media_update",
			Payload: model.MediaUpdatePayload{
				Kind:       "upload",
				SourceURL:  "/media/" + payload.UploadID,
				MediaType:  "direct",
				Title:      uploadTask.Filename,
				Status:     "uploading",
				UploaderID: ms.UploaderID,
			},
		})
	}

	broadcastToRoom(h.m, roomIDStr, model.Message{
		Type: "player_update",
		Payload: model.PlayerUpdatePayload{
			Playing:   false,
			Position:  0,
			UpdatedAt: now,
			Reason:    "media_change",
		},
	})
}

func (h *Hub) isMediaReady(r *room.Room) bool {
	ms := r.GetMediaState()
	return ms == nil || ms.Status != "uploading"
}

func (h *Hub) handlePlay(s *melody.Session, msg model.Message) {
	roomID, _ := s.Get("room_id")
	if roomID == nil {
		h.sendError(s, "not_in_room", "must join room first", msg.ClientMsgID)
		return
	}
	memberID, _ := s.Get("member_id")
	key := "play:" + memberID.(string)
	rl, _ := h.rateLimits.LoadOrStore(key, newRateLimiter(10, time.Second))
	if !rl.(*rateLimiter).Allow() {
		h.sendError(s, "rate_limited", "play rate limited", msg.ClientMsgID)
		return
	}

	var payload model.PlayPayload
	b, _ := json.Marshal(msg.Payload)
	json.Unmarshal(b, &payload)

	r := h.rm.GetRoom(roomID.(string))
	if r == nil {
		return
	}
	if !h.isMediaReady(r) {
		h.sendError(s, "media_not_ready", "media is uploading", msg.ClientMsgID)
		return
	}

	now := time.Now().UnixMilli()
	t := r.GetPSM().HandlePlay(payload.Position, now)

	broadcastPlayerUpdate(h.m, roomID.(string), t)
}

func (h *Hub) handlePause(s *melody.Session, msg model.Message) {
	roomID, _ := s.Get("room_id")
	if roomID == nil {
		h.sendError(s, "not_in_room", "must join room first", msg.ClientMsgID)
		return
	}
	memberID, _ := s.Get("member_id")
	key := "pause:" + memberID.(string)
	rl, _ := h.rateLimits.LoadOrStore(key, newRateLimiter(10, time.Second))
	if !rl.(*rateLimiter).Allow() {
		h.sendError(s, "rate_limited", "pause rate limited", msg.ClientMsgID)
		return
	}

	var payload model.PausePayload
	b, _ := json.Marshal(msg.Payload)
	json.Unmarshal(b, &payload)

	r := h.rm.GetRoom(roomID.(string))
	if r == nil {
		return
	}
	if !h.isMediaReady(r) {
		h.sendError(s, "media_not_ready", "media is uploading", msg.ClientMsgID)
		return
	}

	now := time.Now().UnixMilli()
	t := r.GetPSM().HandlePause(payload.Position, now)

	broadcastPlayerUpdate(h.m, roomID.(string), t)
}

func (h *Hub) handleSeek(s *melody.Session, msg model.Message) {
	roomID, _ := s.Get("room_id")
	if roomID == nil {
		h.sendError(s, "not_in_room", "must join room first", msg.ClientMsgID)
		return
	}
	memberID, _ := s.Get("member_id")
	key := "seek:" + memberID.(string)
	rl, _ := h.rateLimits.LoadOrStore(key, newRateLimiter(10, time.Second))
	if !rl.(*rateLimiter).Allow() {
		h.sendError(s, "rate_limited", "seek rate limited", msg.ClientMsgID)
		return
	}

	var payload model.SeekPayload
	b, _ := json.Marshal(msg.Payload)
	json.Unmarshal(b, &payload)

	r := h.rm.GetRoom(roomID.(string))
	if r == nil {
		return
	}
	if !h.isMediaReady(r) {
		h.sendError(s, "media_not_ready", "media is uploading", msg.ClientMsgID)
		return
	}

	now := time.Now().UnixMilli()
	t := r.GetPSM().HandleSeek(payload.Position, now, r.GetPSM().GetDuration())
	if t != nil {
		broadcastPlayerUpdate(h.m, roomID.(string), t)
	}
}

func (h *Hub) handleBuffering(s *melody.Session, msg model.Message) {
	roomID, _ := s.Get("room_id")
	if roomID == nil {
		return
	}
	memberID, _ := s.Get("member_id")
	if memberID == nil {
		return
	}

	var payload model.BufferingPayload
	b, _ := json.Marshal(msg.Payload)
	json.Unmarshal(b, &payload)

	r := h.rm.GetRoom(roomID.(string))
	if r == nil {
		return
	}

	now := time.Now().UnixMilli()
	t := r.GetPSM().HandleBuffering(memberID.(string), payload.Buffering, now)
	if t != nil {
		broadcastPlayerUpdate(h.m, roomID.(string), t)
	}
}

func (h *Hub) handleHeartbeat(s *melody.Session, msg model.Message) {
	roomID, _ := s.Get("room_id")
	if roomID == nil {
		return
	}

	var payload model.HeartbeatPayload
	b, _ := json.Marshal(msg.Payload)
	json.Unmarshal(b, &payload)

	r := h.rm.GetRoom(roomID.(string))
	if r == nil {
		return
	}

	if payload.Duration != nil && *payload.Duration > 0 {
		r.GetPSM().UpdateDuration(*payload.Duration)
	}

	now := time.Now().UnixMilli()
	if t := r.GetPSM().CheckEnded(now); t != nil {
		broadcastPlayerUpdate(h.m, roomID.(string), t)
	}
}

func (h *Hub) handleUploadProgress(s *melody.Session, msg model.Message) {
	roomID, _ := s.Get("room_id")
	if roomID == nil {
		return
	}

	var payload model.UploadProgressPayload
	b, _ := json.Marshal(msg.Payload)
	json.Unmarshal(b, &payload)

	memberID, _ := s.Get("member_id")

	broadcastToRoom(h.m, roomID.(string), model.Message{
		Type: "upload_progress",
		Payload: model.UploadProgressBroadcast{
			MemberID:      memberID.(string),
			UploadID:      payload.UploadID,
			BytesUploaded: payload.BytesUploaded,
			BytesTotal:    payload.BytesTotal,
		},
	})
}

func (h *Hub) handleUploadCancel(s *melody.Session, msg model.Message) {
	roomID, _ := s.Get("room_id")
	if roomID == nil {
		return
	}

	var payload model.UploadCancelPayload
	b, _ := json.Marshal(msg.Payload)
	json.Unmarshal(b, &payload)

	memberID, _ := s.Get("member_id")

	um := h.rm.GetUploadManager()
	um.CancelUpload(payload.UploadID)

	broadcastToRoom(h.m, roomID.(string), model.Message{
		Type: "upload_cancel",
		Payload: model.UploadCancelBroadcast{
			MemberID: memberID.(string),
			UploadID: payload.UploadID,
		},
	})
}

func (h *Hub) handleChat(s *melody.Session, msg model.Message) {
	roomID, _ := s.Get("room_id")
	if roomID == nil {
		h.sendError(s, "not_in_room", "must join room first", msg.ClientMsgID)
		return
	}

	memberID, _ := s.Get("member_id")
	key := "chat:" + memberID.(string)
	rl, _ := h.rateLimits.LoadOrStore(key, newRateLimiter(3, time.Second))
	if !rl.(*rateLimiter).Allow() {
		h.sendError(s, "rate_limited", "chat rate limited", msg.ClientMsgID)
		return
	}

	var payload model.ChatPayload
	b, _ := json.Marshal(msg.Payload)
	json.Unmarshal(b, &payload)

	if len(payload.Text) > 500 {
		h.sendError(s, "message_too_long", "message too long", msg.ClientMsgID)
		return
	}

	nickname, _ := s.Get("nickname")

	r := h.rm.GetRoom(roomID.(string))
	if r != nil {
		r.AddChatMessage(room.ChatMessage{
			SenderID: memberID.(string),
			Nickname: nickname.(string),
			Text:     payload.Text,
			Ts:       time.Now().UnixMilli(),
		})
	}

	broadcastToRoom(h.m, roomID.(string), model.Message{
		Type: "chat",
		Payload: model.ChatBroadcastPayload{
			SenderID: memberID.(string),
			Nickname: nickname.(string),
			Text:     payload.Text,
			Ts:       time.Now().UnixMilli(),
		},
	})
}

func (h *Hub) handlePing(s *melody.Session, msg model.Message) {
	var payload model.PingPayload
	b, _ := json.Marshal(msg.Payload)
	json.Unmarshal(b, &payload)

	resp := model.Message{
		Type: "pong",
		Payload: model.PongPayload{
			T1:         payload.T1,
			T2:         time.Now().UnixMilli(),
			CauseMsgID: msg.ClientMsgID,
		},
	}
	data, _ := json.Marshal(resp)
	s.Write(data)
}

func (h *Hub) sendError(s *melody.Session, code, message string, causeMsgID string) {
	resp := model.Message{
		Type: "error",
		Payload: model.ErrorPayload{
			Code:       code,
			Message:    message,
			CauseMsgID: causeMsgID,
		},
	}
	data, _ := json.Marshal(resp)
	s.Write(data)
}

func broadcastPlayerUpdate(m *melody.Melody, roomID string, t *room.PlayerTransition) {
	if t == nil {
		return
	}
	broadcastToRoom(m, roomID, model.Message{
		Type: "player_update",
		Payload: model.PlayerUpdatePayload{
			Playing:   t.NewPlaying,
			Position:  t.NewPosition,
			UpdatedAt: t.NewUpdatedAt,
			Reason:    t.Reason,
		},
	})
}

func (h *Hub) BroadcastMediaReady(roomID string, sourceURL string, title string) {
	broadcastToRoom(h.m, roomID, model.Message{
		Type: "media_update",
		Payload: model.MediaUpdatePayload{
			Kind:      "upload",
			SourceURL: sourceURL,
			MediaType: "direct",
			Title:     title,
			Status:    "ready",
		},
	})
}

func broadcastToRoom(m *melody.Melody, roomID string, msg model.Message) {
	data, _ := json.Marshal(msg)
	m.BroadcastFilter(data, func(s *melody.Session) bool {
		rID, ok := s.Get("room_id")
		return ok && rID == roomID
	})
}

func toMemberInfos(members []room.Member) []model.MemberInfo {
	result := make([]model.MemberInfo, len(members))
	for i, m := range members {
		result[i] = model.MemberInfo{
			ID:       m.ID,
			Nickname: m.Nickname,
		}
	}
	return result
}

func isValidURL(raw string) bool {
	if len(raw) > 2048 {
		return false
	}
	u, err := url.Parse(raw)
	if err != nil {
		return false
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return false
	}
	ext := strings.ToLower(path.Ext(u.Path))
	if ext == "" {
		return false
	}
	allowed := map[string]bool{".mp4": true, ".webm": true, ".m3u8": true}
	return allowed[ext]
}

func inferMediaType(sourceURL string, kind string) string {
	if kind == "upload" {
		return "direct"
	}
	ext := strings.ToLower(path.Ext(sourceURL))
	if ext == ".m3u8" {
		return "hls"
	}
	return "direct"
}

func extractTitle(sourceURL string, title string) string {
	if title != "" {
		return title
	}
	u, err := url.Parse(sourceURL)
	if err != nil {
		return "Unknown"
	}
	base := path.Base(u.Path)
	if base == "" || base == "/" {
		return "Unknown"
	}
	return base
}

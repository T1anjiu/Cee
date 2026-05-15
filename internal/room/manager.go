package room

import (
	"sync"
	"time"

	"github.com/cee/watch-together/internal/upload"
	"github.com/cee/watch-together/internal/util"
)

type ManagerConfig struct {
	MaxRoomUploads int64
	MinFreeDisk    int64
	UploadDir      string
}

type Manager struct {
	rooms      map[string]*Room
	mu         sync.RWMutex
	config     ManagerConfig
	onRoomEmpty  func(roomID string)
	onMediaReady func(roomID string, sourceURL string, title string)
	uploadMgr    *upload.Manager
}

func NewManager(config ManagerConfig, idleTimeout time.Duration) *Manager {
	um := upload.NewManager(
		config.UploadDir,
		4*1024*1024*1024,
		config.MaxRoomUploads,
		config.MinFreeDisk,
		idleTimeout,
	)

	m := &Manager{
		rooms:     make(map[string]*Room),
		config:    config,
		uploadMgr: um,
	}

	um.SetOnUploadComplete(func(roomID, uploadID, filePath, filename string, size int64) {
		m.handleUploadComplete(roomID, uploadID, filePath, filename, size)
	})

	um.SetOnUploadCancel(func(roomID, uploadID string) {
		m.handleUploadCancel(roomID, uploadID)
	})

	go m.cleanupTicker()
	return m
}

func (m *Manager) GetUploadManager() *upload.Manager {
	return m.uploadMgr
}

func (m *Manager) GetUploadDir() string {
	return m.config.UploadDir
}

func (m *Manager) handleUploadComplete(roomID, uploadID, filePath, filename string, size int64) {
	m.mu.RLock()
	r, exists := m.rooms[roomID]
	m.mu.RUnlock()

	if !exists {
		return
	}

	now := time.Now().UnixMilli()
	sourceURL := "/media/" + uploadID
	r.SetMediaState(&MediaState{
		Kind:      "upload",
		SourceURL: sourceURL,
		MediaType: "direct",
		Title:     filename,
		Status:    "ready",
		UpdatedAt: now,
	})

	r.GetPSM().ClearBufferingMembers(now)

	if m.onMediaReady != nil {
		m.onMediaReady(roomID, sourceURL, filename)
	}
}

func (m *Manager) handleUploadCancel(roomID, uploadID string) {
	m.mu.RLock()
	r, exists := m.rooms[roomID]
	m.mu.RUnlock()

	if !exists {
		return
	}

	ms := r.GetMediaState()
	if ms != nil && ms.SourceURL == "/media/"+uploadID {
		r.SetMediaState(nil)
	}
}

func (m *Manager) SetOnRoomEmpty(fn func(roomID string)) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.onRoomEmpty = fn
}

func (m *Manager) SetOnMediaReady(fn func(roomID string, sourceURL string, title string)) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.onMediaReady = fn
}

func (m *Manager) CreateRoom() (*Room, string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	for i := 0; i < 10; i++ {
		code := util.GenerateRoomCode()
		if _, exists := m.rooms[code]; !exists {
			room := NewRoom(code)
			m.rooms[code] = room
			return room, code
		}
	}

	for i := 0; i < 10; i++ {
		code := util.GenerateRoomCode() + util.GenerateRoomCode()[:1]
		if _, exists := m.rooms[code]; !exists {
			room := NewRoom(code)
			m.rooms[code] = room
			return room, code
		}
	}

	return nil, ""
}

func (m *Manager) GetRoom(id string) *Room {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.rooms[id]
}

func (m *Manager) RoomExists(id string) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	_, exists := m.rooms[id]
	return exists
}

func (m *Manager) RemoveRoom(id string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.rooms, id)
	m.uploadMgr.CleanupRoom(id)
}

func (m *Manager) cleanupTicker() {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()
	for range ticker.C {
		m.cleanup()
	}
}

func (m *Manager) cleanup() {
	m.mu.Lock()
	defer m.mu.Unlock()

	now := time.Now()
	toDelete := make([]string, 0)
	for id, room := range m.rooms {
		emptyAt := room.GetEmptyAt()
		if emptyAt != nil && now.Sub(*emptyAt) > 5*time.Minute {
			toDelete = append(toDelete, id)
		}
	}

	for _, id := range toDelete {
		delete(m.rooms, id)
		if m.onRoomEmpty != nil {
			m.onRoomEmpty(id)
		}
		m.uploadMgr.CleanupRoom(id)
	}
}

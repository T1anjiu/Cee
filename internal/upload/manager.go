package upload

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/cee/watch-together/internal/util"
)

type TaskState string

const (
	StateReady      TaskState = "ready"
	StateAssembling TaskState = "assembling"
	StateCompleted  TaskState = "completed"
)

type UploadTask struct {
	UploadID     string
	RoomID       string
	MemberID     string
	Filename     string
	Size         int64
	ChunkSize    int64
	TotalChunks  int
	ReceivedChunks map[int]bool
	State        TaskState
	CreatedAt    time.Time
	LastActiveAt time.Time
}

type metaJSON struct {
	ChunkSize    int64     `json:"chunk_size"`
	TotalChunks  int       `json:"total_chunks"`
	Received     []int     `json:"received"`
	State        TaskState `json:"state"`
	Filename     string    `json:"filename"`
	Size         int64     `json:"size"`
	MemberID     string    `json:"member_id"`
	RoomID       string    `json:"room_id"`
}

type Manager struct {
	uploadDir      string
	maxUploadSize  int64
	maxRoomUploads int64
	minFreeDisk    int64
	idleTimeout    time.Duration

	tasks map[string]*UploadTask
	mu    sync.RWMutex

	reservedBytes atomic.Int64

	onUploadComplete func(roomID string, uploadID string, filePath string, filename string, size int64)
	onUploadCancel   func(roomID string, uploadID string)
}

func NewManager(uploadDir string, maxUploadSize int64, maxRoomUploads int64, minFreeDisk int64, idleTimeout time.Duration) *Manager {
	m := &Manager{
		uploadDir:      uploadDir,
		maxUploadSize:  maxUploadSize,
		maxRoomUploads: maxRoomUploads,
		minFreeDisk:    minFreeDisk,
		idleTimeout:    idleTimeout,
		tasks:          make(map[string]*UploadTask),
	}

	m.runStartupGC()
	go m.backgroundGC()
	go m.idleTimeoutWatcher()

	return m
}

func (m *Manager) SetOnUploadComplete(fn func(roomID string, uploadID string, filePath string, filename string, size int64)) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.onUploadComplete = fn
}

func (m *Manager) SetOnUploadCancel(fn func(roomID string, uploadID string)) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.onUploadCancel = fn
}

func (m *Manager) CreateUpload(roomID, memberID, filename string, size int64, suggestedChunkSize int64) (*UploadTask, error) {
	if size > m.maxUploadSize {
		return nil, fmt.Errorf("quota_exceeded: file too large")
	}

	ext := strings.ToLower(filepath.Ext(filename))
	if ext != ".mp4" && ext != ".webm" {
		return nil, fmt.Errorf("unsupported_media: only .mp4 and .webm allowed")
	}

	m.mu.Lock()

	// Check room-level concurrent upload
	for _, t := range m.tasks {
		if t.RoomID == roomID && t.State == StateReady {
			m.mu.Unlock()
			return nil, fmt.Errorf("upload_in_progress: room already has an active upload")
		}
	}

	// Check member-level concurrent upload
	for _, t := range m.tasks {
		if t.MemberID == memberID && t.State == StateReady {
			m.mu.Unlock()
			return nil, fmt.Errorf("upload_in_progress: member already has an active upload")
		}
	}

	// Check disk space with reservation — must be inside lock to be atomic
	freeDisk := getFreeDisk(m.uploadDir)
	reserved := m.reservedBytes.Load()
	if freeDisk-reserved-size < m.minFreeDisk {
		m.mu.Unlock()
		return nil, fmt.Errorf("disk_full: insufficient disk space")
	}

	uploadID := util.GenerateUUID()

	// Reserve bytes atomically while holding the lock
	m.reservedBytes.Add(size)

	m.mu.Unlock()

	chunkSize := suggestedChunkSize
	if chunkSize < 1*1024*1024 {
		chunkSize = 1 * 1024 * 1024
	}
	if chunkSize > 16*1024*1024 {
		chunkSize = 16 * 1024 * 1024
	}

	totalChunks := int((size + chunkSize - 1) / chunkSize)
	if totalChunks < 1 {
		totalChunks = 1
	}

	// Create room dir and task dir
	roomDir := filepath.Join(m.uploadDir, roomID)
	taskDir := filepath.Join(roomDir, uploadID)
	if err := os.MkdirAll(taskDir, 0755); err != nil {
		// Rollback reservation on failure
		m.reservedBytes.Add(-size)
		return nil, fmt.Errorf("internal_error: %w", err)
	}

	task := &UploadTask{
		UploadID:       uploadID,
		RoomID:         roomID,
		MemberID:       memberID,
		Filename:       filename,
		Size:           size,
		ChunkSize:      chunkSize,
		TotalChunks:    totalChunks,
		ReceivedChunks: make(map[int]bool),
		State:          StateReady,
		CreatedAt:      time.Now(),
		LastActiveAt:   time.Now(),
	}

	// Write meta.json
	meta := metaJSON{
		ChunkSize:   chunkSize,
		TotalChunks: totalChunks,
		Received:    []int{},
		State:       StateReady,
		Filename:    filename,
		Size:        size,
		MemberID:    memberID,
		RoomID:      roomID,
	}
	m.writeMeta(taskDir, meta)

	m.mu.Lock()
	m.tasks[uploadID] = task
	m.mu.Unlock()

	return task, nil
}

func (m *Manager) WriteChunk(uploadID string, index int, data []byte) error {
	m.mu.RLock()
	task, exists := m.tasks[uploadID]
	m.mu.RUnlock()

	if !exists {
		return fmt.Errorf("upload_not_found")
	}
	if task.State != StateReady {
		return fmt.Errorf("upload_not_ready")
	}
	if index < 0 || index >= task.TotalChunks {
		return fmt.Errorf("invalid_chunk_index")
	}

	// Size check: all chunks except last must be exactly chunkSize; last can be smaller
	if int64(len(data)) > task.ChunkSize && index < task.TotalChunks-1 {
		return fmt.Errorf("invalid_chunk_size: chunk exceeds negotiated size")
	}

	// Double-check disk space
	freeDisk := getFreeDisk(m.uploadDir)
	reserved := m.reservedBytes.Load()
	if freeDisk-reserved < m.minFreeDisk {
		return fmt.Errorf("disk_full")
	}

	chunkPath := filepath.Join(m.uploadDir, task.RoomID, uploadID, fmt.Sprintf("%04d.part", index))
	if err := os.WriteFile(chunkPath, data, 0644); err != nil {
		return fmt.Errorf("internal_error: %w", err)
	}

	// Update task
	m.mu.Lock()
	if t, ok := m.tasks[uploadID]; ok {
		t.ReceivedChunks[index] = true
		t.LastActiveAt = time.Now()
	}
	m.mu.Unlock()

	// Update meta.json
	m.updateMeta(uploadID, func(meta *metaJSON) {
		meta.Received = append(meta.Received, index)
	})

	return nil
}

func (m *Manager) CompleteUpload(uploadID string) (string, int64, error) {
	m.mu.Lock()
	task, exists := m.tasks[uploadID]
	if !exists {
		m.mu.Unlock()
		return "", 0, fmt.Errorf("upload_not_found")
	}
	if task.State != StateReady {
		m.mu.Unlock()
		return "", 0, fmt.Errorf("upload_not_ready")
	}

	// Verify all chunks received
	for i := 0; i < task.TotalChunks; i++ {
		if !task.ReceivedChunks[i] {
			m.mu.Unlock()
			return "", 0, fmt.Errorf("assemble_failed: missing chunk %d", i)
		}
	}
	m.mu.Unlock()

	taskDir := filepath.Join(m.uploadDir, task.RoomID, uploadID)

	// Update state to assembling
	m.updateMeta(uploadID, func(meta *metaJSON) {
		meta.State = StateAssembling
	})

	ext := filepath.Ext(task.Filename)
	outPath := filepath.Join(m.uploadDir, task.RoomID, uploadID+ext)
	tmpPath := outPath + ".tmp"

	// Assemble chunks
	f, err := os.Create(tmpPath)
	if err != nil {
		m.updateMeta(uploadID, func(meta *metaJSON) { meta.State = StateReady })
		return "", 0, fmt.Errorf("assemble_failed: %w", err)
	}

	var totalWritten int64
	for i := 0; i < task.TotalChunks; i++ {
		chunkPath := filepath.Join(taskDir, fmt.Sprintf("%04d.part", i))
		data, err := os.ReadFile(chunkPath)
		if err != nil {
			f.Close()
			os.Remove(tmpPath)
			m.updateMeta(uploadID, func(meta *metaJSON) { meta.State = StateReady })
			return "", 0, fmt.Errorf("assemble_failed: %w", err)
		}
		n, err := f.Write(data)
		if err != nil {
			f.Close()
			os.Remove(tmpPath)
			m.updateMeta(uploadID, func(meta *metaJSON) { meta.State = StateReady })
			return "", 0, fmt.Errorf("assemble_failed: %w", err)
		}
		totalWritten += int64(n)
	}

	f.Sync()
	f.Close()

	// Atomic rename
	if err := os.Rename(tmpPath, outPath); err != nil {
		os.Remove(tmpPath)
		return "", 0, fmt.Errorf("assemble_failed: %w", err)
	}

	// Magic number check
	if err := validateMagic(outPath, ext); err != nil {
		os.Remove(outPath)
		m.cleanupTask(uploadID)
		return "", 0, fmt.Errorf("file_type_rejected: %w", err)
	}

	// Write final meta
	metaPath := filepath.Join(m.uploadDir, task.RoomID, uploadID+".meta.json")
	finalMeta := metaJSON{
		ChunkSize:   task.ChunkSize,
		TotalChunks: task.TotalChunks,
		State:       StateCompleted,
		Filename:    task.Filename,
		Size:        totalWritten,
		MemberID:    task.MemberID,
		RoomID:      task.RoomID,
	}
	metaData, _ := json.Marshal(finalMeta)
	os.WriteFile(metaPath, metaData, 0644)

	// Update task state
	m.mu.Lock()
	if t, ok := m.tasks[uploadID]; ok {
		t.State = StateCompleted
	}
	m.mu.Unlock()

	// Cleanup chunk dir
	os.RemoveAll(taskDir)

	// Release reserved bytes
	m.reservedBytes.Add(-task.Size)

	// Notify
	if m.onUploadComplete != nil {
		internalPath := "/media/" + uploadID
		m.onUploadComplete(task.RoomID, uploadID, internalPath, task.Filename, totalWritten)
	}

	return outPath, totalWritten, nil
}

func (m *Manager) CancelUpload(uploadID string) error {
	m.mu.Lock()
	task, exists := m.tasks[uploadID]
	if !exists {
		m.mu.Unlock()
		return fmt.Errorf("upload_not_found")
	}
	m.mu.Unlock()

	m.cleanupTask(uploadID)

	if m.onUploadCancel != nil {
		m.onUploadCancel(task.RoomID, uploadID)
	}

	return nil
}

func (m *Manager) GetTask(uploadID string) *UploadTask {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.tasks[uploadID]
}

// GetTaskWithFallback looks up a task in memory, then falls back to scanning
// disk for completed uploads (handles server restart scenario).
func (m *Manager) GetTaskWithFallback(uploadID string) (*UploadTask, string) {
	m.mu.RLock()
	task := m.tasks[uploadID]
	m.mu.RUnlock()
	if task != nil {
		return task, ""
	}

	// Scan disk for completed files
	entries, err := os.ReadDir(m.uploadDir)
	if err != nil {
		return nil, ""
	}
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		roomDir := filepath.Join(m.uploadDir, entry.Name())
		roomEntries, _ := os.ReadDir(roomDir)
		for _, re := range roomEntries {
			if re.IsDir() {
				continue
			}
			name := re.Name()
			if strings.HasSuffix(name, ".meta.json") {
				continue
			}
			baseName := strings.TrimSuffix(name, filepath.Ext(name))
			if baseName == uploadID {
				return nil, filepath.Join(roomDir, name)
			}
		}
	}
	return nil, ""
}

func (m *Manager) CleanupRoom(roomID string) {
	m.mu.Lock()
	toDelete := make([]string, 0)
	for id, task := range m.tasks {
		if task.RoomID == roomID {
			toDelete = append(toDelete, id)
		}
	}
	m.mu.Unlock()

	for _, id := range toDelete {
		m.cleanupTask(id)
	}

	roomDir := filepath.Join(m.uploadDir, roomID)
	os.RemoveAll(roomDir)
}

func (m *Manager) deleteFile(uploadID string) {
	m.mu.Lock()
	task, exists := m.tasks[uploadID]
	m.mu.Unlock()

	if !exists {
		return
	}

	// Delete assembled file
	ext := filepath.Ext(task.Filename)
	filePath := filepath.Join(m.uploadDir, task.RoomID, uploadID+ext)
	os.Remove(filePath)

	// Delete meta
	metaPath := filepath.Join(m.uploadDir, task.RoomID, uploadID+".meta.json")
	os.Remove(metaPath)

	// Release bytes
	m.reservedBytes.Add(-task.Size)
}

func (m *Manager) cleanupTask(uploadID string) {
	m.mu.Lock()
	task, exists := m.tasks[uploadID]
	if !exists {
		m.mu.Unlock()
		return
	}
	delete(m.tasks, uploadID)
	m.mu.Unlock()

	taskDir := filepath.Join(m.uploadDir, task.RoomID, uploadID)
	os.RemoveAll(taskDir)

	m.reservedBytes.Add(-task.Size)
}

func (m *Manager) updateMeta(uploadID string, fn func(*metaJSON)) {
	m.mu.RLock()
	task, exists := m.tasks[uploadID]
	m.mu.RUnlock()
	if !exists {
		return
	}

	taskDir := filepath.Join(m.uploadDir, task.RoomID, uploadID)
	metaPath := filepath.Join(taskDir, "meta.json")

	var meta metaJSON
	data, err := os.ReadFile(metaPath)
	if err == nil {
		json.Unmarshal(data, &meta)
	}

	fn(&meta)
	metaData, _ := json.Marshal(meta)
	os.WriteFile(metaPath, metaData, 0644)
}

func (m *Manager) writeMeta(taskDir string, meta metaJSON) {
	metaPath := filepath.Join(taskDir, "meta.json")
	data, _ := json.Marshal(meta)
	os.WriteFile(metaPath, data, 0644)
}

func (m *Manager) runStartupGC() {
	entries, err := os.ReadDir(m.uploadDir)
	if err != nil {
		return
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			// Clean up orphaned .tmp files
			if filepath.Ext(entry.Name()) == ".tmp" {
				os.Remove(filepath.Join(m.uploadDir, entry.Name()))
			}
			continue
		}

		roomDir := filepath.Join(m.uploadDir, entry.Name())
		roomEntries, _ := os.ReadDir(roomDir)

		for _, re := range roomEntries {
			// Check for .meta.json files
			if !re.IsDir() && strings.HasSuffix(re.Name(), ".meta.json") {
				continue
			}

			itemPath := filepath.Join(roomDir, re.Name())
			if re.IsDir() {
				// Check if it's an upload task dir with meta.json
				metaPath := filepath.Join(itemPath, "meta.json")
				if _, err := os.Stat(metaPath); err == nil {
					var meta metaJSON
					data, _ := os.ReadFile(metaPath)
					json.Unmarshal(data, &meta)
					if meta.State == StateAssembling {
						os.RemoveAll(itemPath)
					}
					// For ready state without a matching task, leave it for IDLE timeout
				}
			} else if strings.HasSuffix(re.Name(), ".tmp") {
				os.Remove(itemPath)
			} else if !strings.HasSuffix(re.Name(), ".meta.json") {
				// Orphaned assembled file without meta?
				baseName := strings.TrimSuffix(re.Name(), filepath.Ext(re.Name()))
				metaPath := filepath.Join(roomDir, baseName+".meta.json")
				if _, err := os.Stat(metaPath); os.IsNotExist(err) {
					os.Remove(itemPath)
				}
			}
		}
	}
}

func (m *Manager) backgroundGC() {
	ticker := time.NewTicker(10 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		m.gcPass()
	}
}

func (m *Manager) gcPass() {
	entries, err := os.ReadDir(m.uploadDir)
	if err != nil {
		return
	}

	now := time.Now()

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		roomDir := filepath.Join(m.uploadDir, entry.Name())
		roomEntries, _ := os.ReadDir(roomDir)

		for _, re := range roomEntries {
			if !re.IsDir() {
				continue
			}

			taskDir := filepath.Join(roomDir, re.Name())
			metaPath := filepath.Join(taskDir, "meta.json")
			metaData, err := os.ReadFile(metaPath)
			if err != nil {
				// Can't read meta, cleanup
				os.RemoveAll(taskDir)
				continue
			}

			var meta metaJSON
			if err := json.Unmarshal(metaData, &meta); err != nil {
				os.RemoveAll(taskDir)
				continue
			}

			if meta.State == StateAssembling {
				os.RemoveAll(taskDir)
				continue
			}

			// Check if it's a stale upload (>1 hour idle)
			info, err := os.Stat(metaPath)
			if err != nil {
				continue
			}
			if now.Sub(info.ModTime()) > 1*time.Hour {
				os.RemoveAll(taskDir)
			}
		}
	}

	// Clean up assembled files > 24h without a matching room task
	m.mu.RLock()
	activeTaskIDs := make(map[string]bool)
	for _, task := range m.tasks {
		activeTaskIDs[task.UploadID+task.RoomID] = true
	}
	m.mu.RUnlock()

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		roomDir := filepath.Join(m.uploadDir, entry.Name())
		roomEntries, _ := os.ReadDir(roomDir)
		for _, re := range roomEntries {
			if re.IsDir() {
				continue
			}
			name := re.Name()
			if strings.HasSuffix(name, ".meta.json") {
				continue
			}
			baseName := strings.TrimSuffix(name, filepath.Ext(name))
			if !activeTaskIDs[baseName+entry.Name()] {
				info, err := re.Info()
				if err != nil {
					continue
				}
				if now.Sub(info.ModTime()) > 24*time.Hour {
					os.Remove(filepath.Join(roomDir, name))
					metaPath := filepath.Join(roomDir, baseName+".meta.json")
					os.Remove(metaPath)
				}
			}
		}
	}
}

func (m *Manager) idleTimeoutWatcher() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		now := time.Now()
		m.mu.Lock()
		toCancel := make([]string, 0)
		for id, task := range m.tasks {
			if task.State == StateReady && now.Sub(task.LastActiveAt) > m.idleTimeout {
				toCancel = append(toCancel, id)
			}
		}
		m.mu.Unlock()

		for _, id := range toCancel {
			m.CancelUpload(id)
		}
	}
}

func getFreeDisk(path string) int64 {
	return getFreeDiskSyscall(path)
}

func validateMagic(filePath string, ext string) error {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return err
	}

	if len(data) < 16 {
		return fmt.Errorf("file too small for magic check")
	}

	switch ext {
	case ".mp4":
		// MP4 starts with ftyp box
		if data[4] != 'f' || data[5] != 't' || data[6] != 'y' || data[7] != 'p' {
			return fmt.Errorf("not a valid MP4 file")
		}
	case ".webm":
		// WebM starts with EBML header
		if data[0] != 0x1A || data[1] != 0x45 || data[2] != 0xDF || data[3] != 0xA3 {
			return fmt.Errorf("not a valid WebM file")
		}
	}

	return nil
}

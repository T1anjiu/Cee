package room

import (
	"sync"
	"time"

	"github.com/cee/watch-together/internal/util"
)

type Member struct {
	ID        string
	Nickname  string
	Token     string
	JoinedAt  time.Time
	LastSeen  time.Time
}

type Room struct {
	ID        string
	Members   map[string]*Member
	mu        sync.RWMutex
	CreatedAt time.Time
	EmptyAt   *time.Time

	mediaState  *MediaState
	psm         *PlayerStateMachine
	chatHistory []ChatMessage
}

type MediaState struct {
	Kind       string
	SourceURL  string
	MediaType  string
	Title      string
	Status     string
	UploaderID string
	UpdatedAt  int64
}

type PlayerStateSnapshot struct {
	Playing   bool
	Position  float64
	UpdatedAt int64
}

type ChatMessage struct {
	SenderID string
	Nickname string
	Text     string
	Ts       int64
}

func NewRoom(id string) *Room {
	return &Room{
		ID:          id,
		Members:     make(map[string]*Member),
		CreatedAt:   time.Now(),
		psm:         NewPlayerStateMachine(),
		chatHistory: make([]ChatMessage, 0),
	}
}

func (r *Room) AddMember(nickname string, token string) (*Member, bool) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if token != "" {
		for _, m := range r.Members {
			if m.Token == token {
				m.Nickname = nickname
				m.LastSeen = time.Now()
				r.EmptyAt = nil
				return m, true
			}
		}
	}

	id := util.GenerateUUID()
	newToken := util.GenerateToken()
	m := &Member{
		ID:       id,
		Nickname: nickname,
		Token:    newToken,
		JoinedAt: time.Now(),
		LastSeen: time.Now(),
	}
	r.Members[id] = m
	r.EmptyAt = nil
	return m, false
}

func (r *Room) RemoveMember(id string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.Members, id)
	if len(r.Members) == 0 {
		now := time.Now()
		r.EmptyAt = &now
	}
}

func (r *Room) GetMemberByToken(token string) *Member {
	r.mu.RLock()
	defer r.mu.RUnlock()
	for _, m := range r.Members {
		if m.Token == token {
			return m
		}
	}
	return nil
}

func (r *Room) GetMember(id string) *Member {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.Members[id]
}

func (r *Room) MemberCount() int {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return len(r.Members)
}

func (r *Room) ListMembers() []Member {
	r.mu.RLock()
	defer r.mu.RUnlock()
	result := make([]Member, 0, len(r.Members))
	for _, m := range r.Members {
		result = append(result, Member{
			ID:       m.ID,
			Nickname: m.Nickname,
		})
	}
	return result
}

func (r *Room) GetMediaState() *MediaState {
	r.mu.RLock()
	defer r.mu.RUnlock()
	if r.mediaState == nil {
		return nil
	}
	ms := *r.mediaState
	return &ms
}

func (r *Room) SetMediaState(ms *MediaState) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.mediaState = ms
}

func (r *Room) GetPlayerState() *PlayerStateSnapshot {
	playing, pos, updatedAt := r.psm.Snapshot()
	return &PlayerStateSnapshot{
		Playing:   playing,
		Position:  pos,
		UpdatedAt: updatedAt,
	}
}

func (r *Room) GetPSM() *PlayerStateMachine {
	return r.psm
}

func (r *Room) AddChatMessage(msg ChatMessage) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.chatHistory = append(r.chatHistory, msg)
	if len(r.chatHistory) > 200 {
		r.chatHistory = r.chatHistory[len(r.chatHistory)-200:]
	}
}

func (r *Room) GetChatHistory() []ChatMessage {
	r.mu.RLock()
	defer r.mu.RUnlock()
	result := make([]ChatMessage, len(r.chatHistory))
	copy(result, r.chatHistory)
	return result
}

func (r *Room) IsEmpty() bool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return len(r.Members) == 0
}

func (r *Room) GetEmptyAt() *time.Time {
	r.mu.RLock()
	defer r.mu.RUnlock()
	if r.EmptyAt == nil {
		return nil
	}
	t := *r.EmptyAt
	return &t
}

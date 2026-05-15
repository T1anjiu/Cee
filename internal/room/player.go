package room

import (
	"sync"
)

type PlayerStateEnum int

const (
	StatePlaying PlayerStateEnum = iota
	StateUserPaused
	StateBufferingPaused
)

type PlayerStateMachine struct {
	mu sync.Mutex

	State               PlayerStateEnum
	Playing             bool
	Position            float64
	UpdatedAt           int64
	Duration            *float64

	BufferingMembers    map[string]bool
	BufferingShieldUntil int64
	ColdStartUntil      int64
}

func NewPlayerStateMachine() *PlayerStateMachine {
	return &PlayerStateMachine{
		State:             StateUserPaused,
		BufferingMembers:  make(map[string]bool),
	}
}

type PlayerTransition struct {
	NewPlaying   bool
	NewPosition  float64
	NewUpdatedAt int64
	Reason       string
}

func (psm *PlayerStateMachine) HandlePlay(position float64, now int64) *PlayerTransition {
	psm.mu.Lock()
	defer psm.mu.Unlock()

	psm.Position = position
	psm.UpdatedAt = now
	psm.BufferingShieldUntil = now + 4000
	psm.State = StatePlaying
	psm.Playing = true

	return &PlayerTransition{
		NewPlaying:   true,
		NewPosition:  position,
		NewUpdatedAt: now,
		Reason:       "user_play",
	}
}

func (psm *PlayerStateMachine) HandlePause(position float64, now int64) *PlayerTransition {
	psm.mu.Lock()
	defer psm.mu.Unlock()

	psm.Position = position
	psm.UpdatedAt = now
	psm.State = StateUserPaused
	psm.Playing = false

	return &PlayerTransition{
		NewPlaying:   false,
		NewPosition:  position,
		NewUpdatedAt: now,
		Reason:       "user_pause",
	}
}

func (psm *PlayerStateMachine) HandleSeek(position float64, now int64, duration *float64) *PlayerTransition {
	psm.mu.Lock()
	defer psm.mu.Unlock()

	if duration != nil && position > *duration {
		position = *duration
	}
	if position < 0 {
		position = 0
	}

	psm.Position = position
	psm.UpdatedAt = now

	return &PlayerTransition{
		NewPlaying:   psm.Playing,
		NewPosition:  position,
		NewUpdatedAt: now,
		Reason:       "seek",
	}
}

func (psm *PlayerStateMachine) HandleBuffering(memberID string, buffering bool, now int64) *PlayerTransition {
	psm.mu.Lock()
	defer psm.mu.Unlock()

	if buffering {
		if now < psm.ColdStartUntil || now < psm.BufferingShieldUntil {
			psm.BufferingMembers[memberID] = true
			return nil
		}
		psm.BufferingMembers[memberID] = true

		if psm.State == StatePlaying {
			psm.State = StateBufferingPaused
			psm.Playing = false
			return &PlayerTransition{
				NewPlaying:   false,
				NewPosition:  psm.Position,
				NewUpdatedAt: now,
				Reason:       "buffering",
			}
		}
	} else {
		delete(psm.BufferingMembers, memberID)

		if psm.State == StateBufferingPaused && len(psm.BufferingMembers) == 0 {
			psm.State = StatePlaying
			psm.Playing = true
			psm.UpdatedAt = now
			return &PlayerTransition{
				NewPlaying:   true,
				NewPosition:  psm.Position,
				NewUpdatedAt: now,
				Reason:       "resume",
			}
		}
	}

	return nil
}

func (psm *PlayerStateMachine) RemoveBufferingMember(memberID string, now int64) *PlayerTransition {
	psm.mu.Lock()
	defer psm.mu.Unlock()

	delete(psm.BufferingMembers, memberID)

	if psm.State == StateBufferingPaused && len(psm.BufferingMembers) == 0 {
		psm.State = StatePlaying
		psm.Playing = true
		psm.UpdatedAt = now
		return &PlayerTransition{
			NewPlaying:   true,
			NewPosition:  psm.Position,
			NewUpdatedAt: now,
			Reason:       "resume",
		}
	}

	return nil
}

func (psm *PlayerStateMachine) ClearBufferingMembers(now int64) {
	psm.mu.Lock()
	defer psm.mu.Unlock()

	psm.BufferingMembers = make(map[string]bool)
	psm.ColdStartUntil = now + 8000
}

func (psm *PlayerStateMachine) SetColdStart(now int64) {
	psm.mu.Lock()
	defer psm.mu.Unlock()
	psm.ColdStartUntil = now + 8000
}

func (psm *PlayerStateMachine) CheckEnded(now int64) *PlayerTransition {
	psm.mu.Lock()
	defer psm.mu.Unlock()

	if psm.State == StatePlaying && psm.Duration != nil {
		elapsed := psm.Position + float64(now-psm.UpdatedAt)/1000
		if elapsed >= *psm.Duration {
			psm.State = StateUserPaused
			psm.Playing = false
			psm.Position = *psm.Duration
			psm.UpdatedAt = now
			return &PlayerTransition{
				NewPlaying:   false,
				NewPosition:  *psm.Duration,
				NewUpdatedAt: now,
				Reason:       "ended",
			}
		}
	}

	return nil
}

func (psm *PlayerStateMachine) UpdateDuration(duration float64) {
	psm.mu.Lock()
	defer psm.mu.Unlock()
	psm.Duration = &duration
}

func (psm *PlayerStateMachine) GetDuration() *float64 {
	psm.mu.Lock()
	defer psm.mu.Unlock()
	if psm.Duration == nil {
		return nil
	}
	d := *psm.Duration
	return &d
}

func (psm *PlayerStateMachine) Snapshot() (bool, float64, int64) {
	psm.mu.Lock()
	defer psm.mu.Unlock()
	return psm.Playing, psm.Position, psm.UpdatedAt
}

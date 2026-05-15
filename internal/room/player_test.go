package room

import (
	"testing"
	"time"
)

func TestPlayingToUserPaused(t *testing.T) {
	psm := NewPlayerStateMachine()
	now := time.Now().UnixMilli()

	psm.HandlePlay(0, now)
	if !psm.Playing || psm.State != StatePlaying {
		t.Fatal("expected Playing state")
	}

	t1 := psm.HandlePause(10, now+1000)
	if t1 == nil || t1.NewPlaying || t1.Reason != "user_pause" {
		t.Fatal("expected user_pause transition")
	}
	if psm.State != StateUserPaused || psm.Position != 10 {
		t.Fatal("expected UserPaused state")
	}
}

func TestPlayingToBufferingPaused(t *testing.T) {
	psm := NewPlayerStateMachine()
	now := time.Now().UnixMilli()

	psm.HandlePlay(0, now)
	psm.ColdStartUntil = 0      // no cold start
	psm.BufferingShieldUntil = 0 // no shield (test cold-start behavior)

	t1 := psm.HandleBuffering("member1", true, now+1000)
	if t1 == nil || !t1.NewPlaying == false || t1.Reason != "buffering" {
		t.Fatal("expected buffering transition to pause")
	}
	if psm.State != StateBufferingPaused || psm.Playing {
		t.Fatal("expected BufferingPaused state")
	}
}

func TestBufferingPausedToPlayingOnClear(t *testing.T) {
	psm := NewPlayerStateMachine()
	now := time.Now().UnixMilli()

	psm.HandlePlay(0, now)
	psm.ColdStartUntil = 0
	psm.BufferingShieldUntil = 0
	psm.HandleBuffering("member1", true, now+1000)

	t1 := psm.HandleBuffering("member1", false, now+2000)
	if t1 == nil || !t1.NewPlaying || t1.Reason != "resume" {
		t.Fatal("expected resume transition")
	}
	if psm.State != StatePlaying || !psm.Playing {
		t.Fatal("expected Playing state after resume")
	}
	if psm.UpdatedAt != now+2000 {
		t.Fatal("expected updated_at to be refreshed")
	}
}

func TestBufferingShield(t *testing.T) {
	psm := NewPlayerStateMachine()
	now := time.Now().UnixMilli()

	psm.HandlePlay(0, now)
	psm.ColdStartUntil = 0
	psm.HandleBuffering("member1", true, now+1000)

	// User forces play, sets shield
	t1 := psm.HandlePlay(5, now+2000)
	if t1 == nil || !t1.NewPlaying || t1.Reason != "user_play" {
		t.Fatal("expected user_play")
	}
	if psm.BufferingShieldUntil < now+2000 {
		t.Fatal("expected shieldUntil to be set")
	}

	// During shield, buffering should not trigger state change
	t2 := psm.HandleBuffering("member1", true, now+2500)
	if t2 != nil {
		t.Fatal("expected no transition during shield period")
	}
	if psm.State != StatePlaying || !psm.Playing {
		t.Fatal("expected still Playing during shield")
	}
}

func TestMemberLeaveClearsBuffering(t *testing.T) {
	psm := NewPlayerStateMachine()
	now := time.Now().UnixMilli()

	psm.HandlePlay(0, now)
	psm.ColdStartUntil = 0
	psm.BufferingShieldUntil = 0
	psm.HandleBuffering("member1", true, now+1000)
	psm.HandleBuffering("member2", true, now+1000)

	if psm.State != StateBufferingPaused {
		t.Fatal("expected BufferingPaused")
	}

	// member1 leaves
	t1 := psm.RemoveBufferingMember("member1", now+2000)
	if t1 != nil || psm.State != StateBufferingPaused {
		t.Fatal("expected still BufferingPaused with member2 remaining")
	}

	// member2 leaves, set should be empty
	t2 := psm.RemoveBufferingMember("member2", now+3000)
	if t2 == nil || !t2.NewPlaying || t2.Reason != "resume" {
		t.Fatal("expected resume after all buffering members leave")
	}
}

func TestNewMemberColdStart(t *testing.T) {
	psm := NewPlayerStateMachine()
	now := time.Now().UnixMilli()

	psm.HandlePlay(0, now)
	psm.SetColdStart(now)

	t1 := psm.HandleBuffering("new-member", true, now+100)
	if t1 != nil {
		t.Fatal("expected no transition during cold start")
	}
	if psm.State != StatePlaying {
		t.Fatal("expected Playing during cold start")
	}

	// After cold start, buffering triggers transition
	t2 := psm.HandleBuffering("new-member", true, now+10000)
	if t2 == nil || t2.Reason != "buffering" {
		t.Fatal("expected buffering transition after cold start")
	}
}

func TestMediaSwitchClearsBuffering(t *testing.T) {
	psm := NewPlayerStateMachine()
	now := time.Now().UnixMilli()

	psm.HandlePlay(0, now)
	psm.ColdStartUntil = 0
	psm.HandleBuffering("member1", true, now+1000)

	psm.ClearBufferingMembers(now + 2000)
	if len(psm.BufferingMembers) != 0 {
		t.Fatal("expected buffering members cleared")
	}
	if psm.ColdStartUntil < now+2000 {
		t.Fatal("expected cold start reset")
	}
}

func TestPlayEndDetection(t *testing.T) {
	psm := NewPlayerStateMachine()
	now := time.Now().UnixMilli()

	psm.HandlePlay(0, now)
	psm.UpdateDuration(10)

	t1 := psm.CheckEnded(now + 15000)
	if t1 == nil || t1.NewPlaying || t1.Reason != "ended" {
		t.Fatal("expected ended transition")
	}
	if psm.State != StateUserPaused || psm.Position != 10 {
		t.Fatalf("expected UserPaused at position 10, got state=%d pos=%f", psm.State, psm.Position)
	}
}

func TestUpdatedAtRefreshedOnPlay(t *testing.T) {
	psm := NewPlayerStateMachine()
	now := time.Now().UnixMilli()

	psm.HandlePlay(0, now)
	psm.ColdStartUntil = 0

	// Enter buffering
	psm.HandleBuffering("member1", true, now+1000)

	// Resume (play)
	t1 := psm.HandlePlay(5, now+2000)
	if t1.NewUpdatedAt != now+2000 {
		t.Fatal("expected updated_at refreshed on entering Playing")
	}

	// Check ended detection uses refreshed time
	psm.UpdateDuration(10)
	t2 := psm.CheckEnded(now + 9000)
	if t2 == nil || t2.Reason != "ended" {
		t.Fatal("expected ended after position+elapsed >= duration")
	}
}

func TestSeekClamp(t *testing.T) {
	psm := NewPlayerStateMachine()
	now := time.Now().UnixMilli()

	dur := float64(100)
	t1 := psm.HandleSeek(200, now, &dur)
	if t1.NewPosition != 100 {
		t.Fatal("expected seek clamped to duration")
	}

	t2 := psm.HandleSeek(-10, now, &dur)
	if t2.NewPosition != 0 {
		t.Fatal("expected seek clamped to 0")
	}
}

func TestBufferingMemberAfterShield(t *testing.T) {
	psm := NewPlayerStateMachine()
	now := time.Now().UnixMilli()

	psm.HandlePlay(0, now)
	psm.ColdStartUntil = 0
	psm.HandleBuffering("member1", true, now+1000)

	// shield
	psm.HandlePlay(5, now+2000)

	// In shield, buffering=true records but doesn't transition
	psm.HandleBuffering("member2", true, now+2500)

	// After shield expires, new buffering=true triggers transition to BufferingPaused
	t1 := psm.HandleBuffering("member2", true, now+7000)
	if t1 == nil || t1.Reason != "buffering" {
		t.Fatal("expected buffering transition after shield expires")
	}
	if psm.State != StateBufferingPaused || psm.Playing {
		t.Fatal("expected BufferingPaused after shield expires with buffering member")
	}
}

func TestColdStartExpiry(t *testing.T) {
	psm := NewPlayerStateMachine()
	now := time.Now().UnixMilli()

	psm.HandlePlay(0, now)
	psm.SetColdStart(now)

	// During cold start, buffering=true records but doesn't trigger
	psm.HandleBuffering("new-guy", true, now+100)

	// After cold start expires, buffering triggers transition
	t1 := psm.HandleBuffering("new-guy", true, now+10000)
	if t1 == nil || t1.Reason != "buffering" {
		t.Fatal("expected buffering transition after cold start")
	}
	if psm.State != StateBufferingPaused || psm.Playing {
		t.Fatal("expected BufferingPaused after cold start with buffering member")
	}
}

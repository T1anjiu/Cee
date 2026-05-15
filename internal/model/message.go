package model

type Message struct {
	Type          string      `json:"type"`
	ClientMsgID   string      `json:"client_msg_id,omitempty"`
	Payload       interface{} `json:"payload,omitempty"`
}

type JoinPayload struct {
	Nickname    string `json:"nickname"`
	MemberToken string `json:"member_token,omitempty"`
}

type PlayPayload struct {
	Position float64 `json:"position"`
}

type PausePayload struct {
	Position float64 `json:"position"`
}

type SeekPayload struct {
	Position float64 `json:"position"`
}

type ChangeMediaPayload struct {
	Kind       string `json:"kind"`
	SourceURL  string `json:"source_url,omitempty"`
	UploadID   string `json:"upload_id,omitempty"`
	Title      string `json:"title,omitempty"`
}

type BufferingPayload struct {
	Buffering bool `json:"buffering"`
}

type ChatPayload struct {
	Text string `json:"text"`
}

type PingPayload struct {
	T1 int64 `json:"t1"`
}

type HeartbeatPayload struct {
	Position  float64 `json:"position"`
	Playing   bool    `json:"playing"`
	Duration  *float64 `json:"duration,omitempty"`
}

type UploadProgressPayload struct {
	UploadID      string `json:"upload_id"`
	BytesUploaded int64  `json:"bytes_uploaded"`
	BytesTotal    int64  `json:"bytes_total"`
}

type UploadCancelPayload struct {
	UploadID string `json:"upload_id"`
}

type RoomStatePayload struct {
	RoomID       string           `json:"room_id"`
	SelfID       string           `json:"self_id"`
	SelfToken    string           `json:"self_token"`
	Members      []MemberInfo     `json:"members"`
	Media        *MediaState      `json:"media"`
	Player       *PlayerState     `json:"player"`
	ChatHistory  []ChatMessage    `json:"chat_history"`
}

type MemberInfo struct {
	ID       string `json:"id"`
	Nickname string `json:"nickname"`
}

type MediaState struct {
	Kind       string `json:"kind"`
	SourceURL  string `json:"source_url"`
	MediaType  string `json:"media_type"`
	Title      string `json:"title"`
	Status     string `json:"status"`
	UploaderID string `json:"uploader_id,omitempty"`
}

type PlayerState struct {
	Playing   bool  `json:"playing"`
	Position  float64 `json:"position"`
	UpdatedAt int64 `json:"updated_at"`
	Reason    string `json:"reason,omitempty"`
}

type ChatMessage struct {
	SenderID string `json:"sender_id"`
	Nickname string `json:"nickname"`
	Text     string `json:"text"`
	Ts       int64  `json:"ts"`
}

type PlayerUpdatePayload struct {
	Playing   bool    `json:"playing"`
	Position  float64 `json:"position"`
	UpdatedAt int64   `json:"updated_at"`
	Reason    string  `json:"reason,omitempty"`
}

type MediaUpdatePayload struct {
	Kind       string `json:"kind"`
	SourceURL  string `json:"source_url"`
	MediaType  string `json:"media_type"`
	Title      string `json:"title"`
	Status     string `json:"status"`
	UploaderID string `json:"uploader_id,omitempty"`
}

type MemberJoinPayload struct {
	ID       string `json:"id"`
	Nickname string `json:"nickname"`
}

type MemberLeavePayload struct {
	ID string `json:"id"`
}

type MemberResumePayload struct {
	ID string `json:"id"`
}

type ChatBroadcastPayload struct {
	SenderID string `json:"sender_id"`
	Nickname string `json:"nickname"`
	Text     string `json:"text"`
	Ts       int64  `json:"ts"`
}

type PongPayload struct {
	T1          int64  `json:"t1"`
	T2          int64  `json:"t2"`
	CauseMsgID  string `json:"cause_msg_id,omitempty"`
}

type UploadProgressBroadcast struct {
	MemberID      string `json:"member_id"`
	UploadID      string `json:"upload_id"`
	BytesUploaded int64  `json:"bytes_uploaded"`
	BytesTotal    int64  `json:"bytes_total"`
}

type UploadCancelBroadcast struct {
	MemberID string `json:"member_id"`
	UploadID string `json:"upload_id"`
}

type ErrorPayload struct {
	Code        string `json:"code"`
	Message     string `json:"message"`
	CauseMsgID  string `json:"cause_msg_id,omitempty"`
}

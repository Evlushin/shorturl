package models

type TAction string

const (
	RAShorten TAction = "shorten"
	RAFollow  TAction = "follow"
)

type AuditEvent struct {
	TS      int64   `json:"ts"`
	Action  TAction `json:"action"`
	UserID  string  `json:"user_id,omitempty"`
	OrigURL string  `json:"url"`
}

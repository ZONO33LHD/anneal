package gateway

import "context"

// NotifyLevel は通知の緊急度・種別を表す。
type NotifyLevel string

const (
	NotifyInfo     NotifyLevel = "info"
	NotifyPriority NotifyLevel = "priority"
	NotifyApproval NotifyLevel = "approval"
	NotifySuccess  NotifyLevel = "success"
)

// NotifyMessage は人間向けの単一の通知を表す。
type NotifyMessage struct {
	Level NotifyLevel
	Title string
	Body  string
	URL   string
}

// Notifier は通知を配信する。
type Notifier interface {
	ProviderName() string
	Notify(ctx context.Context, msg NotifyMessage) error
}

package usecase

import (
	"context"
	"strings"

	"github.com/ZONO33LHD/anneal/domain/gateway"
	"github.com/ZONO33LHD/anneal/domain/model"
	"github.com/ZONO33LHD/anneal/domain/repository"
)

// WebhookSignal は外部 webhook を状態機械へ渡す前に正規化したシグナルである。
type WebhookSignal string

const (
	// WebhookSignalAlertDetected は alert 起点で新しい更新候補を検知したことを表す。
	WebhookSignalAlertDetected WebhookSignal = "alert_detected"
	// WebhookSignalCIPassed は CI が成功系の conclusion で完了したことを表す。
	WebhookSignalCIPassed WebhookSignal = "ci_passed"
	// WebhookSignalCIFailed は CI が失敗系の conclusion で完了したことを表す。
	WebhookSignalCIFailed WebhookSignal = "ci_failed"
	// WebhookSignalReviewApproved はレビューが approve されたことを表す。
	WebhookSignalReviewApproved WebhookSignal = "review_approved"
	// WebhookSignalReviewChangesRequested はレビューで変更要求が出たことを表す。
	WebhookSignalReviewChangesRequested WebhookSignal = "review_changes_requested"
	// WebhookSignalPullRequestMerged は PR が merge されたことを表す。
	WebhookSignalPullRequestMerged WebhookSignal = "pull_request_merged"
	// WebhookSignalPullRequestClosed は PR が merge されず close されたことを表す。
	WebhookSignalPullRequestClosed WebhookSignal = "pull_request_closed"
)

// WebhookEvent は delivery 層から渡される、GitHub に依存しない webhook 入力である。
type WebhookEvent struct {
	Repository        string
	PullRequestNumber int
	Branch            string
	HeadSHA           string
	Signal            WebhookSignal
	Alert             *WebhookAlert
}

// WebhookAlert は PR ではなく脆弱性 alert 等を起点に届く更新候補である。
type WebhookAlert struct {
	PackageName      string
	Ecosystem        model.Ecosystem
	CurrentVersion   string
	TargetVersion    string
	VulnerableRange  string
	AdvisoryID       string
	AdvisorySeverity string
	AdvisorySummary  string
	AdvisoryURL      string
}

// AlertUpdateKey は alert を既存/新規 record と対応付けるキーを返す。
func AlertUpdateKey(event WebhookEvent) (string, bool) {
	if event.Repository == "" || event.Alert == nil {
		return "", false
	}
	if event.Alert.PackageName == "" || event.Alert.TargetVersion == "" {
		return "", false
	}
	return model.UpdateKey(event.Repository, event.Alert.PackageName, event.Alert.TargetVersion), true
}

// AlertCVEInfo は alert payload を DependencyUpdate の CVEInfo へ写す。
func AlertCVEInfo(alert WebhookAlert) *model.CVEInfo {
	return &model.CVEInfo{
		ID:             alert.AdvisoryID,
		Severity:       strings.ToLower(alert.AdvisorySeverity),
		AffectedRange:  alert.VulnerableRange,
		PatchedVersion: alert.TargetVersion,
		Summary:        alert.AdvisorySummary,
	}
}

// WebhookUsecase は webhook イベントから該当レコードを探し、エンジンへ渡す。
type WebhookUsecase interface {
	Handle(ctx context.Context, event WebhookEvent) (bool, error)
}

type webhookUsecase struct {
	updates repository.UpdateRepository
	engine  EngineUsecase
	log     gateway.Logger
}

// NewWebhookUsecase は webhook 用ユースケースを組み立てる。
func NewWebhookUsecase(
	updates repository.UpdateRepository,
	engine EngineUsecase,
	log gateway.Logger,
) WebhookUsecase {
	return &webhookUsecase{
		updates: updates,
		engine:  engine,
		log:     log,
	}
}

// Handle は active record のうち webhook と対応する 1 件だけを Dispatch する。
func (u *webhookUsecase) Handle(ctx context.Context, event WebhookEvent) (bool, error) {
	active, err := u.updates.ListActive()
	if err != nil {
		return false, err
	}
	rec, ok := matchWebhookRecord(active, event)
	if !ok {
		u.log.Info(ctx, "webhook record not found",
			"repository", event.Repository,
			"pr_number", event.PullRequestNumber,
			"branch", event.Branch,
			"signal", string(event.Signal),
		)
		return false, nil
	}

	prepared, signalMoved := applyWebhookSignal(ctx, u.log, rec, event.Signal)
	moved, _, err := u.engine.Dispatch(ctx, prepared)
	if err != nil {
		return moved, err
	}
	if !moved && signalMoved {
		if err := u.updates.Put(prepared); err != nil {
			return false, err
		}
		moved = true
	}
	u.log.Info(ctx, "webhook dispatched",
		"update_key", rec.UpdateKey,
		"signal", string(event.Signal),
		"moved", moved,
	)
	return moved, nil
}

func matchWebhookRecord(records []model.DependencyUpdate, event WebhookEvent) (model.DependencyUpdate, bool) {
	for _, rec := range records {
		if rec.Repository != event.Repository {
			continue
		}
		if event.PullRequestNumber > 0 && rec.PullRequestNumber == event.PullRequestNumber {
			return rec, true
		}
		if event.PullRequestNumber == 0 && event.Branch != "" && rec.Branch == event.Branch {
			return rec, true
		}
	}
	return model.DependencyUpdate{}, false
}

func applyWebhookSignal(
	ctx context.Context,
	log gateway.Logger,
	rec model.DependencyUpdate,
	signal WebhookSignal,
) (model.DependencyUpdate, bool) {
	switch signal {
	case WebhookSignalCIPassed:
		return transitionCI(ctx, log, rec, model.StateCIPassed, "webhook: CI passed")
	case WebhookSignalCIFailed:
		return transitionCI(ctx, log, rec, model.StateCIFailed, "webhook: CI failed")
	case WebhookSignalReviewChangesRequested:
		return transitionIfAllowed(ctx, log, rec, model.StateChangesRequested, "webhook: changes requested")
	case WebhookSignalPullRequestMerged:
		return transitionIfAllowed(ctx, log, rec, model.StateMerged, "webhook: merged")
	case WebhookSignalPullRequestClosed:
		return transitionIfAllowed(ctx, log, rec, model.StateClosed, "webhook: closed")
	default:
		return rec, false
	}
}

func transitionCI(
	ctx context.Context,
	log gateway.Logger,
	rec model.DependencyUpdate,
	to model.State,
	reason string,
) (model.DependencyUpdate, bool) {
	next := rec
	moved := false
	if next.Status == model.StatePRCreated {
		var err error
		next, err = next.Transition(model.StateCIRunning, "webhook: CI started")
		if err != nil {
			log.Warn(ctx, "webhook CI start ignored", "update_key", rec.UpdateKey, "err", err)
			return rec, false
		}
		moved = true
	}
	if next.Status != model.StateCIRunning {
		return next, moved
	}
	var err error
	next, err = next.Transition(to, reason)
	if err != nil {
		log.Warn(ctx, "webhook CI transition ignored", "update_key", rec.UpdateKey, "err", err)
		return rec, false
	}
	if to == model.StateCIPassed {
		next.CI = &model.CIResult{Status: "passed"}
	} else {
		next.CI = &model.CIResult{Status: "failed", LogSummary: "reported by webhook"}
	}
	return next, true
}

func transitionIfAllowed(
	ctx context.Context,
	log gateway.Logger,
	rec model.DependencyUpdate,
	to model.State,
	reason string,
) (model.DependencyUpdate, bool) {
	next, err := rec.Transition(to, reason)
	if err != nil {
		log.Info(ctx, "webhook transition not applicable",
			"update_key", rec.UpdateKey,
			"from", string(rec.Status),
			"to", string(to),
			"err", err,
		)
		return rec, false
	}
	return next, next.Status != rec.Status
}

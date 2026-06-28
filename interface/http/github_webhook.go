// Package httpinterface は HTTP delivery 層を提供する。
package httpinterface

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/ZONO33LHD/anneal/domain/ctxkey"
	"github.com/ZONO33LHD/anneal/domain/gateway"
	"github.com/ZONO33LHD/anneal/usecase"
)

const maxWebhookBodyBytes = 10 << 20

// GitHubWebhookOptions は GitHub Webhook ハンドラの依存を表す。
type GitHubWebhookOptions struct {
	Secret  string
	Webhook usecase.WebhookUsecase
	Logger  gateway.Logger
}

// NewGitHubWebhookHandler は GitHub Webhook 用 HTTP ハンドラを返す。
func NewGitHubWebhookHandler(opts GitHubWebhookOptions) http.Handler {
	return traceMiddleware(opts.Logger, &githubWebhookHandler{
		secret:  opts.Secret,
		webhook: opts.Webhook,
		log:     opts.Logger,
	})
}

type githubWebhookHandler struct {
	secret  string
	webhook usecase.WebhookUsecase
	log     gateway.Logger
}

func (h *githubWebhookHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", http.MethodPost)
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if h.secret == "" {
		if h.log != nil {
			h.log.Error(r.Context(), "github webhook secret is not configured", errors.New("missing GITHUB_WEBHOOK_SECRET"))
		}
		http.Error(w, "webhook secret is not configured", http.StatusInternalServerError)
		return
	}
	body, err := io.ReadAll(http.MaxBytesReader(w, r.Body, maxWebhookBodyBytes))
	if err != nil {
		if h.log != nil {
			h.log.Warn(r.Context(), "failed to read webhook body", "err", err.Error())
		}
		http.Error(w, "failed to read request body", http.StatusBadRequest)
		return
	}
	if !validSignature(body, h.secret, r.Header.Get("X-Hub-Signature-256")) {
		http.Error(w, "invalid signature", http.StatusUnauthorized)
		return
	}

	eventName := r.Header.Get("X-GitHub-Event")
	event, ok, err := parseGitHubEvent(eventName, body)
	if err != nil {
		// 解析エラーの詳細はサーバ側ログにだけ残し、クライアントへは汎用文言を返す
		// （生のパースエラーを応答に載せて内部情報を漏らさないため）。
		if h.log != nil {
			h.log.Warn(r.Context(), "failed to parse webhook payload", "github_event", eventName, "err", err.Error())
		}
		http.Error(w, "invalid webhook payload", http.StatusBadRequest)
		return
	}
	if !ok {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	if _, err := h.webhook.Handle(r.Context(), event); err != nil {
		if h.log != nil {
			h.log.Error(r.Context(), "webhook usecase failed", err, "github_event", eventName)
		}
		http.Error(w, "webhook handling failed", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// validSignature は GitHub Webhook の署名を検証し、リクエストが本当に GitHub から
// 来た（=共有 secret を知る相手が body を改ざんせず送った）ことを確認する。
// GitHub は受信 body を secret 鍵で HMAC-SHA256 し、その結果を
// "X-Hub-Signature-256: sha256=<hex>" ヘッダで送る。こちらでも同じ body と secret で
// HMAC を計算し直し、両者を比較する。
func validSignature(body []byte, secret, header string) bool {
	const prefix = "sha256="
	// ヘッダは必ず "sha256=" で始まる。前置きが無ければ未署名として弾く。
	if !strings.HasPrefix(header, prefix) {
		return false
	}
	// 16 進文字列の署名値をバイト列へ戻す。壊れた値なら不正として弾く。
	got, err := hex.DecodeString(strings.TrimPrefix(header, prefix))
	if err != nil {
		return false
	}
	// 受信 body から期待値を再計算する。
	mac := hmac.New(sha256.New, []byte(secret))
	_, _ = mac.Write(body)
	want := mac.Sum(nil)
	// タイミング攻撃を避けるため、通常の比較ではなく定数時間比較を使う。
	return hmac.Equal(got, want)
}

func parseGitHubEvent(eventName string, body []byte) (usecase.WebhookEvent, bool, error) {
	switch eventName {
	case "check_suite":
		return parseCheckSuite(body)
	case "pull_request_review":
		return parsePullRequestReview(body)
	case "pull_request":
		return parsePullRequest(body)
	default:
		return usecase.WebhookEvent{}, false, nil
	}
}

type checkSuitePayload struct {
	Action     string `json:"action"`
	Repository struct {
		FullName string `json:"full_name"`
	} `json:"repository"`
	CheckSuite struct {
		Conclusion   string `json:"conclusion"`
		HeadBranch   string `json:"head_branch"`
		HeadSHA      string `json:"head_sha"`
		PullRequests []struct {
			Number int `json:"number"`
			Head   struct {
				Ref string `json:"ref"`
			} `json:"head"`
		} `json:"pull_requests"`
	} `json:"check_suite"`
}

func parseCheckSuite(body []byte) (usecase.WebhookEvent, bool, error) {
	var payload checkSuitePayload
	if err := json.Unmarshal(body, &payload); err != nil {
		return usecase.WebhookEvent{}, false, fmt.Errorf("parse check_suite payload: %w", err)
	}
	if payload.Action != "completed" {
		return usecase.WebhookEvent{}, false, nil
	}
	signal := usecase.WebhookSignalCIFailed
	switch payload.CheckSuite.Conclusion {
	case "success", "neutral", "skipped":
		signal = usecase.WebhookSignalCIPassed
	}
	prNumber := 0
	branch := payload.CheckSuite.HeadBranch
	if len(payload.CheckSuite.PullRequests) > 0 {
		prNumber = payload.CheckSuite.PullRequests[0].Number
		if payload.CheckSuite.PullRequests[0].Head.Ref != "" {
			branch = payload.CheckSuite.PullRequests[0].Head.Ref
		}
	}
	return usecase.WebhookEvent{
		Repository:        payload.Repository.FullName,
		PullRequestNumber: prNumber,
		Branch:            branch,
		HeadSHA:           payload.CheckSuite.HeadSHA,
		Signal:            signal,
	}, true, nil
}

type pullRequestReviewPayload struct {
	Action     string `json:"action"`
	Repository struct {
		FullName string `json:"full_name"`
	} `json:"repository"`
	Review struct {
		State string `json:"state"`
	} `json:"review"`
	PullRequest pullRequestPayload `json:"pull_request"`
}

func parsePullRequestReview(body []byte) (usecase.WebhookEvent, bool, error) {
	var payload pullRequestReviewPayload
	if err := json.Unmarshal(body, &payload); err != nil {
		return usecase.WebhookEvent{}, false, fmt.Errorf("parse pull_request_review payload: %w", err)
	}
	if payload.Action != "submitted" {
		return usecase.WebhookEvent{}, false, nil
	}
	var signal usecase.WebhookSignal
	switch payload.Review.State {
	case "approved":
		signal = usecase.WebhookSignalReviewApproved
	case "changes_requested":
		signal = usecase.WebhookSignalReviewChangesRequested
	default:
		return usecase.WebhookEvent{}, false, nil
	}
	return pullRequestEvent(payload.Repository.FullName, payload.PullRequest, signal), true, nil
}

type pullRequestEnvelope struct {
	Action     string `json:"action"`
	Repository struct {
		FullName string `json:"full_name"`
	} `json:"repository"`
	PullRequest pullRequestPayload `json:"pull_request"`
}

type pullRequestPayload struct {
	Number int  `json:"number"`
	Merged bool `json:"merged"`
	Head   struct {
		Ref string `json:"ref"`
		SHA string `json:"sha"`
	} `json:"head"`
}

func parsePullRequest(body []byte) (usecase.WebhookEvent, bool, error) {
	var payload pullRequestEnvelope
	if err := json.Unmarshal(body, &payload); err != nil {
		return usecase.WebhookEvent{}, false, fmt.Errorf("parse pull_request payload: %w", err)
	}
	if payload.Action != "closed" {
		return usecase.WebhookEvent{}, false, nil
	}
	signal := usecase.WebhookSignalPullRequestClosed
	if payload.PullRequest.Merged {
		signal = usecase.WebhookSignalPullRequestMerged
	}
	return pullRequestEvent(payload.Repository.FullName, payload.PullRequest, signal), true, nil
}

func pullRequestEvent(
	repository string,
	pr pullRequestPayload,
	signal usecase.WebhookSignal,
) usecase.WebhookEvent {
	return usecase.WebhookEvent{
		Repository:        repository,
		PullRequestNumber: pr.Number,
		Branch:            pr.Head.Ref,
		HeadSHA:           pr.Head.SHA,
		Signal:            signal,
	}
}

func traceMiddleware(log gateway.Logger, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := traceContext(r.Context(), r.Header)
		if log != nil {
			log.Debug(ctx, "http request", "method", r.Method, "path", r.URL.Path)
		}
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func traceContext(ctx context.Context, header http.Header) context.Context {
	if id := traceparentTraceID(header.Get("traceparent")); id != "" {
		return ctxkey.WithTraceID(ctx, id)
	}
	if id := cloudTraceID(header.Get("X-Cloud-Trace-Context")); id != "" {
		return ctxkey.WithTraceID(ctx, id)
	}
	ctx, _ = ctxkey.EnsureTraceID(ctx)
	return ctx
}

func traceparentTraceID(value string) string {
	parts := strings.Split(value, "-")
	if len(parts) < 4 {
		return ""
	}
	if validTraceID(parts[1]) {
		return parts[1]
	}
	return ""
}

func cloudTraceID(value string) string {
	traceID, _, _ := strings.Cut(value, "/")
	if validTraceID(traceID) {
		return traceID
	}
	return ""
}

func validTraceID(id string) bool {
	if len(id) != 32 {
		return false
	}
	if id == "00000000000000000000000000000000" {
		return false
	}
	_, err := hex.DecodeString(id)
	return err == nil
}

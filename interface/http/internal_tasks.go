package httpinterface

import (
	"context"
	"crypto/subtle"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"

	"github.com/ZONO33LHD/anneal/domain/config"
	"github.com/ZONO33LHD/anneal/domain/gateway"
	"github.com/ZONO33LHD/anneal/usecase"
)

const internalTokenHeader = "X-Anneal-Internal-Token"

// InternalTaskAuthenticator は内部 task endpoint の bearer token を検証する。
type InternalTaskAuthenticator interface {
	AuthenticateInternalTask(ctx context.Context, authorization string) error
}

// InternalTaskOptions は Cloud Scheduler 等から呼ばれる内部 task endpoint の依存を表す。
type InternalTaskOptions struct {
	Token         string
	Authenticator InternalTaskAuthenticator
	ScanTargets   []config.ScanTarget
	Scan          usecase.ScanUsecase
	Engine        usecase.EngineUsecase
	Anneal        usecase.AnnealUsecase
	Adoption      usecase.AdoptionUsecase
	Logger        gateway.Logger
}

// NewInternalTaskHandler は内部 task endpoint を返す。
func NewInternalTaskHandler(opts InternalTaskOptions) http.Handler {
	return traceMiddleware(opts.Logger, &internalTaskHandler{opts: opts})
}

type internalTaskHandler struct {
	opts InternalTaskOptions
}

func (h *internalTaskHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", http.MethodPost)
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if !h.authorized(r) {
		http.Error(w, "invalid internal authorization", http.StatusUnauthorized)
		return
	}

	var (
		body string
		err  error
	)
	switch r.URL.Path {
	case "/internal/scan":
		body, err = h.runScan(r)
	case "/internal/tick":
		body, err = h.runTick(r)
	default:
		http.NotFound(w, r)
		return
	}
	if err != nil {
		if h.opts.Logger != nil {
			h.opts.Logger.Error(r.Context(), "internal task failed", err, "path", r.URL.Path)
		}
		http.Error(w, "internal task failed: "+err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	_, _ = w.Write([]byte(body + "\n"))
}

func (h *internalTaskHandler) validToken(got string) bool {
	if h.opts.Token == "" {
		return false
	}
	return subtle.ConstantTimeCompare([]byte(got), []byte(h.opts.Token)) == 1
}

func (h *internalTaskHandler) authorized(r *http.Request) bool {
	if h.validToken(r.Header.Get(internalTokenHeader)) {
		return true
	}
	if h.opts.Authenticator == nil {
		return false
	}
	return h.opts.Authenticator.AuthenticateInternalTask(r.Context(), r.Header.Get("Authorization")) == nil
}

func (h *internalTaskHandler) runScan(r *http.Request) (string, error) {
	// push モデル: リクエストボディで owner/repo が指定されていれば、その 1 件を
	// リモート（Contents API）でスキャンする。対象 repo の CI（GitHub Actions）が
	// 自 repo を渡して起動する経路。
	repo := requestedRepository(r)
	if repo != "" {
		res, err := h.opts.Scan.RunRemote(r.Context(), repo)
		if err != nil {
			return "", err
		}
		return fmt.Sprintf("scan complete repository=%s created=%d skipped=%d",
			repo, len(res.Created), res.Skipped), nil
	}

	// フォールバック: repository 未指定なら env の ScanTargets を全件スキャン（demo /
	// 後方互換）。両方無ければ、何をスキャンすべきか不明なのでエラーにする。
	if len(h.opts.ScanTargets) == 0 {
		return "", errors.New("scan requires a repository in the request body or ANNEAL_SCAN_TARGETS")
	}
	created, skipped := 0, 0
	for _, target := range h.opts.ScanTargets {
		res, err := h.opts.Scan.Run(r.Context(), target.Path, target.Repository)
		if err != nil {
			return "", err
		}
		created += len(res.Created)
		skipped += res.Skipped
	}
	return "scan complete targets=" + strconv.Itoa(len(h.opts.ScanTargets)) +
		" created=" + strconv.Itoa(created) +
		" skipped=" + strconv.Itoa(skipped), nil
}

// requestedRepository はリクエストボディ {"repository":"owner/repo"} を読む。
// ボディが空/不正でも空文字を返し、env フォールバックへ委ねる（起動失敗にしない）。
func requestedRepository(r *http.Request) string {
	if r.Body == nil {
		return ""
	}
	body, err := io.ReadAll(io.LimitReader(r.Body, 4<<10))
	if err != nil || len(body) == 0 {
		return ""
	}
	var payload struct {
		Repository string `json:"repository"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		return ""
	}
	return strings.TrimSpace(payload.Repository)
}

func (h *internalTaskHandler) runTick(r *http.Request) (string, error) {
	changed, err := h.opts.Engine.Tick(r.Context())
	if err != nil {
		return "", err
	}
	if err := h.opts.Anneal.MaybeAnneal(r.Context()); err != nil {
		return "", err
	}
	if err := h.opts.Adoption.Evaluate(r.Context()); err != nil {
		return "", err
	}
	return "tick complete advanced=" + strconv.Itoa(changed), nil
}

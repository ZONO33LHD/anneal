package httpinterface

import (
	"context"
	"crypto/subtle"
	"errors"
	"net/http"
	"strconv"

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
	if len(h.opts.ScanTargets) == 0 {
		return "", errors.New("ANNEAL_SCAN_TARGETS is required for internal scan")
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

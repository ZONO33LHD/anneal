package httpinterface

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/ZONO33LHD/anneal/domain/ctxkey"
	"github.com/ZONO33LHD/anneal/domain/model"
	"github.com/ZONO33LHD/anneal/usecase"
)

func TestGitHubWebhookHandlerAcceptsSupportedEvents(t *testing.T) {
	secret := "webhook-secret"
	tests := []struct {
		name       string
		eventName  string
		body       string
		wantSignal usecase.WebhookSignal
		wantPR     int
		wantAlert  bool
	}{
		{
			name:      "check suite passed",
			eventName: "check_suite",
			body: `{
				"action": "completed",
				"repository": {"full_name": "acme/demo"},
				"check_suite": {
					"conclusion": "success",
					"head_branch": "anneal/npm/axios",
					"head_sha": "abc123",
					"pull_requests": [{"number": 42, "head": {"ref": "anneal/npm/axios"}}]
				}
			}`,
			wantSignal: usecase.WebhookSignalCIPassed,
			wantPR:     42,
		},
		{
			name:      "review approved",
			eventName: "pull_request_review",
			body: `{
				"action": "submitted",
				"repository": {"full_name": "acme/demo"},
				"review": {"state": "approved"},
				"pull_request": {"number": 42, "head": {"ref": "anneal/npm/axios", "sha": "abc123"}}
			}`,
			wantSignal: usecase.WebhookSignalReviewApproved,
			wantPR:     42,
		},
		{
			name:      "review changes requested",
			eventName: "pull_request_review",
			body: `{
				"action": "submitted",
				"repository": {"full_name": "acme/demo"},
				"review": {"state": "changes_requested"},
				"pull_request": {"number": 42, "head": {"ref": "anneal/npm/axios", "sha": "abc123"}}
			}`,
			wantSignal: usecase.WebhookSignalReviewChangesRequested,
			wantPR:     42,
		},
		{
			name:      "pull request merged",
			eventName: "pull_request",
			body: `{
				"action": "closed",
				"repository": {"full_name": "acme/demo"},
				"pull_request": {"number": 42, "merged": true, "head": {"ref": "anneal/npm/axios", "sha": "abc123"}}
			}`,
			wantSignal: usecase.WebhookSignalPullRequestMerged,
			wantPR:     42,
		},
		{
			name:      "pull request closed",
			eventName: "pull_request",
			body: `{
				"action": "closed",
				"repository": {"full_name": "acme/demo"},
				"pull_request": {"number": 42, "merged": false, "head": {"ref": "anneal/npm/axios", "sha": "abc123"}}
			}`,
			wantSignal: usecase.WebhookSignalPullRequestClosed,
			wantPR:     42,
		},
		{
			name:      "dependabot alert created",
			eventName: "dependabot_alert",
			body: `{
				"action": "created",
				"repository": {"full_name": "acme/demo"},
				"alert": {
					"html_url": "https://github.com/acme/demo/security/dependabot/1",
					"security_advisory": {
						"ghsa_id": "GHSA-xxxx-yyyy-zzzz",
						"cve_id": "CVE-2026-1234",
						"severity": "high",
						"summary": "demo vulnerability"
					},
					"security_vulnerability": {
						"vulnerable_version_range": "< 1.2.3",
						"package": {"ecosystem": "npm", "name": "lodash"},
						"first_patched_version": {"identifier": "1.2.3"}
					}
				}
			}`,
			wantSignal: usecase.WebhookSignalAlertDetected,
			wantAlert:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			webhook := &fakeWebhookUsecase{}
			handler := NewGitHubWebhookHandler(GitHubWebhookOptions{
				Secret:  secret,
				Webhook: webhook,
				Logger:  noopLogger{},
			})
			req := signedRequest(tt.eventName, tt.body, secret)
			rr := httptest.NewRecorder()

			handler.ServeHTTP(rr, req)

			if rr.Code != http.StatusNoContent {
				t.Fatalf("status=%d, want %d", rr.Code, http.StatusNoContent)
			}
			if webhook.calls != 1 {
				t.Fatalf("calls=%d, want 1", webhook.calls)
			}
			if webhook.event.Repository != "acme/demo" {
				t.Fatalf("repository=%s, want acme/demo", webhook.event.Repository)
			}
			if webhook.event.PullRequestNumber != tt.wantPR {
				t.Fatalf("pr=%d, want %d", webhook.event.PullRequestNumber, tt.wantPR)
			}
			if webhook.event.Signal != tt.wantSignal {
				t.Fatalf("signal=%s, want %s", webhook.event.Signal, tt.wantSignal)
			}
			if tt.wantAlert {
				if webhook.event.Alert == nil {
					t.Fatal("alert=nil, want alert")
				}
				if webhook.event.Alert.PackageName != "lodash" ||
					webhook.event.Alert.TargetVersion != "1.2.3" ||
					webhook.event.Alert.Ecosystem != model.EcosystemNPM {
					t.Fatalf("alert=%+v", webhook.event.Alert)
				}
			}
		})
	}
}

func TestGitHubWebhookHandlerRejectsInvalidSignature(t *testing.T) {
	webhook := &fakeWebhookUsecase{}
	handler := NewGitHubWebhookHandler(GitHubWebhookOptions{
		Secret:  "webhook-secret",
		Webhook: webhook,
		Logger:  noopLogger{},
	})
	req := httptest.NewRequest(http.MethodPost, "/webhooks/github", nil)
	req.Header.Set("X-GitHub-Event", "pull_request")
	req.Header.Set("X-Hub-Signature-256", "sha256=bad")
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("status=%d, want %d", rr.Code, http.StatusUnauthorized)
	}
	if webhook.calls != 0 {
		t.Fatalf("calls=%d, want 0", webhook.calls)
	}
}

func TestGitHubWebhookHandlerIgnoresUnknownEvent(t *testing.T) {
	webhook := &fakeWebhookUsecase{}
	handler := NewGitHubWebhookHandler(GitHubWebhookOptions{
		Secret:  "webhook-secret",
		Webhook: webhook,
		Logger:  noopLogger{},
	})
	req := signedRequest("issues", `{}`, "webhook-secret")
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusNoContent {
		t.Fatalf("status=%d, want %d", rr.Code, http.StatusNoContent)
	}
	if webhook.calls != 0 {
		t.Fatalf("calls=%d, want 0", webhook.calls)
	}
}

func TestGitHubWebhookHandlerRequiresSecret(t *testing.T) {
	webhook := &fakeWebhookUsecase{}
	handler := NewGitHubWebhookHandler(GitHubWebhookOptions{
		Webhook: webhook,
		Logger:  noopLogger{},
	})
	req := signedRequest("issues", `{}`, "webhook-secret")
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusInternalServerError {
		t.Fatalf("status=%d, want %d", rr.Code, http.StatusInternalServerError)
	}
	if webhook.calls != 0 {
		t.Fatalf("calls=%d, want 0", webhook.calls)
	}
}

func TestGitHubWebhookHandlerPropagatesTraceparent(t *testing.T) {
	traceID := "4bf92f3577b34da6a3ce929d0e0e4736"
	webhook := &fakeWebhookUsecase{}
	handler := NewGitHubWebhookHandler(GitHubWebhookOptions{
		Secret:  "webhook-secret",
		Webhook: webhook,
		Logger:  noopLogger{},
	})
	req := signedRequest("pull_request_review", `{
		"action": "submitted",
		"repository": {"full_name": "acme/demo"},
		"review": {"state": "approved"},
		"pull_request": {"number": 42, "head": {"ref": "anneal/npm/axios", "sha": "abc123"}}
	}`, "webhook-secret")
	req.Header.Set("traceparent", "00-"+traceID+"-00f067aa0ba902b7-01")
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusNoContent {
		t.Fatalf("status=%d, want %d", rr.Code, http.StatusNoContent)
	}
	if webhook.traceID != traceID {
		t.Fatalf("trace_id=%s, want %s", webhook.traceID, traceID)
	}
}

func TestGitHubWebhookHandlerPropagatesCloudTraceContext(t *testing.T) {
	traceID := "105445aa7843bc8bf206b12000100000"
	webhook := &fakeWebhookUsecase{}
	handler := NewGitHubWebhookHandler(GitHubWebhookOptions{
		Secret:  "webhook-secret",
		Webhook: webhook,
		Logger:  noopLogger{},
	})
	req := signedRequest("pull_request_review", `{
		"action": "submitted",
		"repository": {"full_name": "acme/demo"},
		"review": {"state": "approved"},
		"pull_request": {"number": 42, "head": {"ref": "anneal/npm/axios", "sha": "abc123"}}
	}`, "webhook-secret")
	req.Header.Set("X-Cloud-Trace-Context", traceID+"/123;o=1")
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusNoContent {
		t.Fatalf("status=%d, want %d", rr.Code, http.StatusNoContent)
	}
	if webhook.traceID != traceID {
		t.Fatalf("trace_id=%s, want %s", webhook.traceID, traceID)
	}
}

func TestGitHubWebhookHandlerEnsuresTraceID(t *testing.T) {
	webhook := &fakeWebhookUsecase{}
	handler := NewGitHubWebhookHandler(GitHubWebhookOptions{
		Secret:  "webhook-secret",
		Webhook: webhook,
		Logger:  noopLogger{},
	})
	req := signedRequest("pull_request_review", `{
		"action": "submitted",
		"repository": {"full_name": "acme/demo"},
		"review": {"state": "approved"},
		"pull_request": {"number": 42, "head": {"ref": "anneal/npm/axios", "sha": "abc123"}}
	}`, "webhook-secret")
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusNoContent {
		t.Fatalf("status=%d, want %d", rr.Code, http.StatusNoContent)
	}
	if len(webhook.traceID) != 32 {
		t.Fatalf("trace_id length=%d, want 32", len(webhook.traceID))
	}
}

func signedRequest(eventName, body, secret string) *http.Request {
	req := httptest.NewRequest(http.MethodPost, "/webhooks/github", strings.NewReader(body))
	req.Header.Set("X-GitHub-Event", eventName)
	req.Header.Set("X-Hub-Signature-256", signBody(body, secret))
	return req
}

func signBody(body, secret string) string {
	mac := hmac.New(sha256.New, []byte(secret))
	_, _ = mac.Write([]byte(body))
	return "sha256=" + hex.EncodeToString(mac.Sum(nil))
}

type fakeWebhookUsecase struct {
	calls   int
	event   usecase.WebhookEvent
	traceID string
}

func (f *fakeWebhookUsecase) Handle(ctx context.Context, event usecase.WebhookEvent) (bool, error) {
	f.calls++
	f.event = event
	f.traceID = ctxkey.TraceID(ctx)
	return true, nil
}

type noopLogger struct{}

func (noopLogger) Debug(context.Context, string, ...any) {
}

func (noopLogger) Info(context.Context, string, ...any) {
}

func (noopLogger) Warn(context.Context, string, ...any) {
}

func (noopLogger) Step(context.Context, string, ...any) {
}

func (noopLogger) Error(context.Context, string, error, ...any) {
}

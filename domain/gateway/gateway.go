// Package gateway declares the ports to external systems and adapters (LLM, git
// host, notifications, package metadata, source manifests, logging).
// Implementations live in infrastructure.
package gateway

import (
	"context"

	"github.com/ZONO33LHD/anneal/domain/model"
)

// --- LLM ---

// LLMRequest is a single generation call.
type LLMRequest struct {
	System      string
	Prompt      string
	Temperature float64
	MaxTokens   int
}

// LLM generates text from a prompt. The agent uses it only for prose/enrichment,
// so the pipeline still works end-to-end with a mock implementation.
type LLM interface {
	Name() string
	Model() string
	Generate(ctx context.Context, req LLMRequest) (string, error)
}

// --- Metadata ---

// MetadataSource answers "what is the newest version" and "is there an advisory".
type MetadataSource interface {
	Name() string
	LatestVersion(ctx context.Context, eco model.Ecosystem, name, current string) (string, error)
	Advisories(ctx context.Context, eco model.Ecosystem, name, current string) ([]model.CVEInfo, error)
}

// --- Notifications ---

// NotifyLevel is the urgency/kind of a notification.
type NotifyLevel string

const (
	NotifyInfo     NotifyLevel = "info"
	NotifyPriority NotifyLevel = "priority"
	NotifyApproval NotifyLevel = "approval"
	NotifySuccess  NotifyLevel = "success"
)

// NotifyMessage is a single human-facing notification.
type NotifyMessage struct {
	Level NotifyLevel
	Title string
	Body  string
	URL   string
}

// Notifier delivers notifications.
type Notifier interface {
	Name() string
	Notify(ctx context.Context, msg NotifyMessage) error
}

// --- Git host ---

// CreatePROptions describes a PR to open.
type CreatePROptions struct {
	Repository   string
	Base         string
	Branch       string
	Title        string
	Body         string
	ChangedFiles []string
}

// PushFixOptions describes a follow-up fix commit.
type PushFixOptions struct {
	Repository   string
	Branch       string
	PRNumber     int
	Message      string
	ChangedFiles []string
}

// CICheckOptions asks for the CI status of a PR. Attempt drives the mock
// self-heal scenario; ShouldFailFirst is a mock-only hint.
type CICheckOptions struct {
	Repository      string
	PRNumber        int
	Branch          string
	Attempt         int
	ShouldFailFirst bool
}

// CICheck is the outcome of a CI status query.
type CICheck struct {
	Passed     bool
	LogSummary string
}

// PRRef identifies an opened pull request.
type PRRef struct {
	Number int
	URL    string
}

// Git is the git-host abstraction.
type Git interface {
	Name() string
	CreateBranchAndPR(ctx context.Context, opts CreatePROptions) (PRRef, error)
	PushFix(ctx context.Context, opts PushFixOptions) error
	CheckCI(ctx context.Context, opts CICheckOptions) (CICheck, error)
}

// --- Source manifests & files ---

// Dependency is a dependency discovered in a manifest.
type Dependency struct {
	Name           string
	CurrentVersion string
	IsDev          bool
	Ecosystem      model.Ecosystem
}

// Ecosystem knows how to read a manifest and apply a version bump to it.
type Ecosystem interface {
	ID() model.Ecosystem
	Detect(repoPath string) bool
	Scan(repoPath string) ([]Dependency, error)
	ApplyUpdate(repoPath, name, target string) ([]string, error)
}

// EcosystemProvider discovers which ecosystems apply to a repository.
type EcosystemProvider interface {
	ForRepo(repoPath string) []Ecosystem
	ByID(id model.Ecosystem) Ecosystem
}

// SourceScanner finds where a package is used in a repository's source.
type SourceScanner interface {
	UsageSites(repoPath, packageName string) []string
}

// --- Logging ---

// Logger is the cross-cutting logging port. Step is used for one line per
// lifecycle move so the asynchronous flow is observable during demos.
type Logger interface {
	Step(msg string)
	Info(msg string)
	Warn(msg string)
	Debug(msg string)
}

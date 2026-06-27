// Package gateway は外部システムやアダプタ（LLM、git ホスト、通知、パッケージ
// メタデータ、ソースマニフェスト、ロギング）へのポートを宣言する。
// 実装は infrastructure に存在する。
package gateway

import (
	"context"

	"github.com/ZONO33LHD/anneal/domain/model"
)

// --- LLM ---

// LLMRequest は単一の生成呼び出しを表す。
type LLMRequest struct {
	System      string
	Prompt      string
	Temperature float64
	MaxTokens   int
}

// LLM はプロンプトからテキストを生成する。エージェントは文章生成や情報の付加に
// のみ利用するため、モック実装でもパイプラインは end-to-end で動作する。
type LLM interface {
	Name() string
	Model() string
	Generate(ctx context.Context, req LLMRequest) (string, error)
}

// --- Metadata ---

// MetadataSource は「最新バージョンは何か」「アドバイザリは存在するか」に答える。
type MetadataSource interface {
	Name() string
	LatestVersion(ctx context.Context, eco model.Ecosystem, name, current string) (string, error)
	Advisories(ctx context.Context, eco model.Ecosystem, name, current string) ([]model.CVEInfo, error)
}

// --- Notifications ---

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
	Name() string
	Notify(ctx context.Context, msg NotifyMessage) error
}

// --- Git host ---

// CreatePROptions は作成する PR を記述する。
type CreatePROptions struct {
	Repository   string
	Base         string
	Branch       string
	Title        string
	Body         string
	ChangedFiles []string
}

// PushFixOptions は後続の修正コミットを記述する。
type PushFixOptions struct {
	Repository   string
	Branch       string
	PRNumber     int
	Message      string
	ChangedFiles []string
}

// CICheckOptions は PR の CI ステータスを問い合わせる。Attempt はモックの自己修復
// シナリオを駆動し、ShouldFailFirst はモック専用のヒントである。
type CICheckOptions struct {
	Repository      string
	PRNumber        int
	Branch          string
	Attempt         int
	ShouldFailFirst bool
}

// CICheck は CI ステータス問い合わせの結果を表す。
type CICheck struct {
	Passed     bool
	LogSummary string
}

// PRRef は作成済みのプルリクエストを識別する。
type PRRef struct {
	Number int
	URL    string
}

// Git は git ホストの抽象化を表す。
type Git interface {
	Name() string
	CreateBranchAndPR(ctx context.Context, opts CreatePROptions) (PRRef, error)
	PushFix(ctx context.Context, opts PushFixOptions) error
	CheckCI(ctx context.Context, opts CICheckOptions) (CICheck, error)
}

// --- Source manifests & files ---

// Dependency はマニフェストから検出された依存関係を表す。
type Dependency struct {
	Name           string
	CurrentVersion string
	IsDev          bool
	Ecosystem      model.Ecosystem
}

// Ecosystem はマニフェストの読み取りと、それへのバージョン更新の適用方法を知る。
type Ecosystem interface {
	ID() model.Ecosystem
	Detect(repoPath string) bool
	Scan(repoPath string) ([]Dependency, error)
	ApplyUpdate(repoPath, name, target string) ([]string, error)
}

// EcosystemProvider はリポジトリに該当するエコシステムを発見する。
type EcosystemProvider interface {
	ForRepo(repoPath string) []Ecosystem
	ByID(id model.Ecosystem) Ecosystem
}

// SourceScanner はリポジトリのソース内でパッケージが使われている箇所を見つける。
type SourceScanner interface {
	UsageSites(repoPath, packageName string) []string
}

// --- Logging ---

// Logger は横断的なロギングのポートである。Step はライフサイクルの遷移ごとに 1 行
// 出力するために使われ、デモ中に非同期フローを観測できるようにする。
type Logger interface {
	Step(msg string)
	Info(msg string)
	Warn(msg string)
	Debug(msg string)
}

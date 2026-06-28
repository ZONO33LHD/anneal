package gateway

import "context"

// CreatePROptions は作成する PR を記述する。
type CreatePROptions struct {
	Repository   string   `json:"repository"`
	WorkDir      string   `json:"work_dir"`
	Base         string   `json:"base"`
	Branch       string   `json:"branch"`
	Title        string   `json:"title"`
	Body         string   `json:"body"`
	ChangedFiles []string `json:"changed_files"`
}

// PushFixOptions は後続の修正コミットを記述する。
type PushFixOptions struct {
	Repository   string   `json:"repository"`
	WorkDir      string   `json:"work_dir"`
	Branch       string   `json:"branch"`
	PRNumber     int      `json:"pr_number"`
	Message      string   `json:"message"`
	ChangedFiles []string `json:"changed_files"`
}

// CICheckOptions は PR の CI ステータスを問い合わせる。Attempt はモックの自己修復
// シナリオを駆動し、ShouldFailFirst はモック専用のヒントである。
type CICheckOptions struct {
	Repository      string `json:"repository"`
	PRNumber        int    `json:"pr_number"`
	Branch          string `json:"branch"`
	Attempt         int    `json:"attempt"`
	ShouldFailFirst bool   `json:"should_fail_first"`
}

// CICheck は CI ステータス問い合わせの結果を表す。
type CICheck struct {
	Complete   bool   `json:"complete"`
	Passed     bool   `json:"passed"`
	LogSummary string `json:"log_summary"`
}

// PRRef は作成済みのプルリクエストを識別する。
type PRRef struct {
	Number int    `json:"number"`
	URL    string `json:"url"`
}

// Git は git ホストの抽象化を表す。
type Git interface {
	ProviderName() string
	CreateBranchAndPR(ctx context.Context, opts CreatePROptions) (PRRef, error)
	PushFix(ctx context.Context, opts PushFixOptions) error
	CheckCI(ctx context.Context, opts CICheckOptions) (CICheck, error)
}

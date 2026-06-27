package gateway

import "context"

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

// Package policy はビジネス上の定数としきい値を一箇所にまとめる。これにより
// エージェントの挙動を調整する際に各レイヤのロジックへ手を入れる必要がなくなる。
package policy

// ScoreWeights は加重合計スコアの重みである。Regression は別途追跡され、合計には
// 含まれない。
var ScoreWeights = struct {
	CI, Review, Risk, PRQuality, FixAccuracy, Merge float64
}{
	CI:          0.30,
	Review:      0.20,
	Risk:        0.15,
	PRQuality:   0.15,
	FixAccuracy: 0.15,
	Merge:       0.05,
}

// SensitiveKeywords は更新時に必ず人間の確認を要するパッケージを示す。auth/payment/crypto
// のコードでのミスは影響範囲（blast radius）が大きいためである。
var SensitiveKeywords = []string{
	"auth", "passport", "jsonwebtoken", "jwt", "bcrypt", "crypto",
	"stripe", "payment", "oauth", "session", "helmet",
}

const (
	// MaxScanFiles は影響分析の上限を定め、大規模リポジトリでも高速さを保つ。
	MaxScanFiles = 500

	// MaxDriveRounds はデモのドライブループの上限を定める暴走防止の安全網である。
	MaxDriveRounds = 100

	// DefaultLowScoreThreshold: これを下回る確定済みスコアは failure case となり、
	// Annealing Loop に供給される。
	DefaultLowScoreThreshold = 70.0

	// DefaultImproveWindow: ループが平均する直近の確定済みスコアの件数。
	DefaultImproveWindow = 5

	// DefaultRegressionWindowDays: マージされた更新を監視する期間。
	DefaultRegressionWindowDays = 7

	// MinCanarySample: カナリア版（候補版）の採用/ロールバックを判断する前に必要な
	// 確定スコアの最小件数。少なすぎる標本での誤った採用/巻き戻しを防ぐ。
	MinCanarySample = 3

	// CanaryRegressionMargin: カナリア版の平均がベースライン版をこの点数だけ下回ったら
	// 劣化とみなしてロールバックする。小さな揺らぎでの巻き戻しを防ぐための余裕。
	CanaryRegressionMargin = 3.0
)

// AutoPRDefaultTypes は自動 PR 作成の対象となる更新種別である。
var AutoPRDefaultTypes = []string{"patch", "minor"}

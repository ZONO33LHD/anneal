// Package policy holds business constants and thresholds in one place, so tuning
// the agent's behaviour does not require touching logic across layers.
package policy

// ScoreWeights are the weights of the weighted total score. Regression is tracked
// separately and is not part of the total.
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

// SensitiveKeywords mark packages whose updates always require a human, because
// a mistake in auth/payment/crypto code is high blast-radius.
var SensitiveKeywords = []string{
	"auth", "passport", "jsonwebtoken", "jwt", "bcrypt", "crypto",
	"stripe", "payment", "oauth", "session", "helmet",
}

const (
	// MaxScanFiles bounds impact analysis so large repositories stay fast.
	MaxScanFiles = 500

	// MaxDriveRounds bounds the demo drive loop as a runaway safety net.
	MaxDriveRounds = 100

	// DefaultLowScoreThreshold: finalized scores below this become failure cases
	// and feed the Annealing Loop.
	DefaultLowScoreThreshold = 70.0

	// DefaultImproveWindow: how many recent finalized scores the loop averages.
	DefaultImproveWindow = 5

	// DefaultRegressionWindowDays: how long a merged update is watched.
	DefaultRegressionWindowDays = 7
)

// AutoPRDefaultTypes are the update kinds eligible for automatic PR creation.
var AutoPRDefaultTypes = []string{"patch", "minor"}

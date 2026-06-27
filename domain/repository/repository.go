// Package repository は永続化のポートを宣言する。実装は infrastructure に置かれ、
// usecase はこれらのインターフェースにのみ依存する（依存ルール:
// usecase -> domain/repository）。
package repository

import "github.com/ZONO33LHD/anneal/domain/model"

// UpdateRepository は依存関係更新のレコードを永続化する。
type UpdateRepository interface {
	Get(updateKey string) (*model.DependencyUpdate, error)
	// GetActive はレコードが終端状態でない場合にのみ返す。重複防止の基礎となる。
	GetActive(updateKey string) (*model.DependencyUpdate, error)
	Put(rec model.DependencyUpdate) error
	List() ([]model.DependencyUpdate, error)
	ListActive() ([]model.DependencyUpdate, error)
}

// EvaluationRepository はスコアカードを永続化する。
type EvaluationRepository interface {
	Get(updateKey string) (*model.AgentEvaluation, error)
	Put(e model.AgentEvaluation) error
	List() ([]model.AgentEvaluation, error)
}

// ImprovementRepository は failure case と改善候補を永続化する。
type ImprovementRepository interface {
	PutFailure(f model.FailureCase) error
	ListFailures() ([]model.FailureCase, error)
	PutImprovement(i model.AgentImprovement) error
	ListImprovements() ([]model.AgentImprovement, error)
}

// Package repository declares the persistence ports. Implementations live in
// infrastructure; usecases depend only on these interfaces (dependency rule:
// usecase -> domain/repository).
package repository

import "github.com/ZONO33LHD/anneal/domain/model"

// UpdateRepository persists dependency-update records.
type UpdateRepository interface {
	Get(updateKey string) (*model.DependencyUpdate, error)
	// GetActive returns the record only if it is non-terminal — the basis for
	// duplicate prevention.
	GetActive(updateKey string) (*model.DependencyUpdate, error)
	Put(rec model.DependencyUpdate) error
	List() ([]model.DependencyUpdate, error)
	ListActive() ([]model.DependencyUpdate, error)
}

// EvaluationRepository persists scorecards.
type EvaluationRepository interface {
	Get(updateKey string) (*model.AgentEvaluation, error)
	Put(e model.AgentEvaluation) error
	List() ([]model.AgentEvaluation, error)
}

// ImprovementRepository persists failure cases and improvement candidates.
type ImprovementRepository interface {
	PutFailure(f model.FailureCase) error
	ListFailures() ([]model.FailureCase, error)
	PutImprovement(i model.AgentImprovement) error
	ListImprovements() ([]model.AgentImprovement, error)
}

package persistence

import (
	"github.com/ZONO33LHD/anneal/domain/model"
	"github.com/ZONO33LHD/anneal/domain/repository"
)

// The three repositories share one DB so a single JSON file holds all state.

type updateRepository struct{ db *DB }

// NewUpdateRepository returns the UpdateRepository backed by db.
func NewUpdateRepository(db *DB) repository.UpdateRepository {
	return &updateRepository{db}
}

func (r *updateRepository) Get(key string) (*model.DependencyUpdate, error) {
	return r.db.getUpdate(key)
}

func (r *updateRepository) GetActive(key string) (*model.DependencyUpdate, error) {
	rec, err := r.db.getUpdate(key)
	if err != nil || rec == nil {
		return nil, err
	}
	if !model.IsActive(rec.Status) {
		return nil, nil
	}
	return rec, nil
}

func (r *updateRepository) Put(rec model.DependencyUpdate) error {
	return r.db.putUpdate(rec)
}

func (r *updateRepository) List() ([]model.DependencyUpdate, error) {
	return r.db.listUpdates()
}

func (r *updateRepository) ListActive() ([]model.DependencyUpdate, error) {
	all, err := r.db.listUpdates()
	if err != nil {
		return nil, err
	}
	out := make([]model.DependencyUpdate, 0, len(all))
	for _, u := range all {
		if model.IsActive(u.Status) {
			out = append(out, u)
		}
	}
	return out, nil
}

type evaluationRepository struct{ db *DB }

// NewEvaluationRepository returns the EvaluationRepository backed by db.
func NewEvaluationRepository(db *DB) repository.EvaluationRepository {
	return &evaluationRepository{db}
}

func (r *evaluationRepository) Get(key string) (*model.AgentEvaluation, error) {
	return r.db.getEval(key)
}
func (r *evaluationRepository) Put(e model.AgentEvaluation) error {
	return r.db.putEval(e)
}
func (r *evaluationRepository) List() ([]model.AgentEvaluation, error) {
	return r.db.listEvals()
}

type improvementRepository struct{ db *DB }

// NewImprovementRepository returns the ImprovementRepository backed by db.
func NewImprovementRepository(db *DB) repository.ImprovementRepository {
	return &improvementRepository{db}
}

func (r *improvementRepository) PutFailure(f model.FailureCase) error {
	return r.db.putFailure(f)
}
func (r *improvementRepository) ListFailures() ([]model.FailureCase, error) {
	return r.db.listFailures()
}
func (r *improvementRepository) PutImprovement(i model.AgentImprovement) error {
	return r.db.putImprovement(i)
}
func (r *improvementRepository) ListImprovements() ([]model.AgentImprovement, error) {
	return r.db.listImprovements()
}

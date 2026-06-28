package firestore

import (
	"context"
	"encoding/json"
	"strings"

	cloudfirestore "cloud.google.com/go/firestore"
	"github.com/ZONO33LHD/anneal/domain/model"
	"github.com/ZONO33LHD/anneal/domain/repository"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const (
	// CollectionUpdates は DependencyUpdate を保持する collection 名である。
	CollectionUpdates = "updates"
	// CollectionEvaluations は AgentEvaluation を保持する collection 名である。
	CollectionEvaluations = "evaluations"
	// CollectionFailures は FailureCase を保持する collection 名である。
	CollectionFailures = "failures"
	// CollectionImprovements は AgentImprovement を保持する collection 名である。
	CollectionImprovements = "improvements"
)

type store struct {
	client *cloudfirestore.Client
	prefix string
}

// RepositoryOption は Firestore repository の構成を変更する。
type RepositoryOption func(*store)

// WithCollectionPrefix は collection 名に prefix を付ける。
func WithCollectionPrefix(prefix string) RepositoryOption {
	return func(s *store) {
		s.prefix = strings.TrimSpace(prefix)
	}
}

func newStore(client *cloudfirestore.Client, opts ...RepositoryOption) *store {
	s := &store{client: client}
	for _, opt := range opts {
		opt(s)
	}
	return s
}

func (s *store) collection(name string) *cloudfirestore.CollectionRef {
	if s.prefix == "" {
		return s.client.Collection(name)
	}
	return s.client.Collection(s.prefix + "_" + name)
}

func docID(key string) string {
	return strings.ReplaceAll(key, "/", "%2F")
}

func encode(v any) (map[string]any, error) {
	data, err := json.Marshal(v)
	if err != nil {
		return nil, err
	}
	out := map[string]any{}
	if err := json.Unmarshal(data, &out); err != nil {
		return nil, err
	}
	return out, nil
}

func decode[T any](data map[string]any) (T, error) {
	var out T
	raw, err := json.Marshal(data)
	if err != nil {
		return out, err
	}
	if err := json.Unmarshal(raw, &out); err != nil {
		return out, err
	}
	return out, nil
}

type updateRepository struct {
	store *store
}

// NewUpdateRepository は Firestore を基盤とする UpdateRepository を返す。
func NewUpdateRepository(client *cloudfirestore.Client) repository.UpdateRepository {
	return NewUpdateRepositoryWithOptions(client)
}

// NewUpdateRepositoryWithOptions は options を適用した UpdateRepository を返す。
func NewUpdateRepositoryWithOptions(client *cloudfirestore.Client, opts ...RepositoryOption) repository.UpdateRepository {
	return &updateRepository{store: newStore(client, opts...)}
}

func (r *updateRepository) Get(key string) (*model.DependencyUpdate, error) {
	ctx := context.Background()
	doc, err := r.store.collection(CollectionUpdates).Doc(docID(key)).Get(ctx)
	if status.Code(err) == codes.NotFound {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	rec, err := decode[model.DependencyUpdate](doc.Data())
	if err != nil {
		return nil, err
	}
	return &rec, nil
}

func (r *updateRepository) GetActive(key string) (*model.DependencyUpdate, error) {
	rec, err := r.Get(key)
	if err != nil || rec == nil {
		return nil, err
	}
	if !model.IsActive(rec.Status) {
		return nil, nil
	}
	return rec, nil
}

func (r *updateRepository) Put(rec model.DependencyUpdate) error {
	data, err := encode(rec)
	if err != nil {
		return err
	}
	_, err = r.store.collection(CollectionUpdates).Doc(docID(rec.UpdateKey)).Set(context.Background(), data)
	return err
}

func (r *updateRepository) List() ([]model.DependencyUpdate, error) {
	docs, err := r.store.collection(CollectionUpdates).Documents(context.Background()).GetAll()
	if err != nil {
		return nil, err
	}
	out := make([]model.DependencyUpdate, 0, len(docs))
	for _, doc := range docs {
		rec, err := decode[model.DependencyUpdate](doc.Data())
		if err != nil {
			return nil, err
		}
		out = append(out, rec)
	}
	return out, nil
}

func (r *updateRepository) ListActive() ([]model.DependencyUpdate, error) {
	all, err := r.List()
	if err != nil {
		return nil, err
	}
	out := make([]model.DependencyUpdate, 0, len(all))
	for _, rec := range all {
		if model.IsActive(rec.Status) {
			out = append(out, rec)
		}
	}
	return out, nil
}

type evaluationRepository struct {
	store *store
}

// NewEvaluationRepository は Firestore を基盤とする EvaluationRepository を返す。
func NewEvaluationRepository(client *cloudfirestore.Client) repository.EvaluationRepository {
	return NewEvaluationRepositoryWithOptions(client)
}

// NewEvaluationRepositoryWithOptions は options を適用した EvaluationRepository を返す。
func NewEvaluationRepositoryWithOptions(client *cloudfirestore.Client, opts ...RepositoryOption) repository.EvaluationRepository {
	return &evaluationRepository{store: newStore(client, opts...)}
}

func (r *evaluationRepository) Get(key string) (*model.AgentEvaluation, error) {
	doc, err := r.store.collection(CollectionEvaluations).Doc(docID(key)).Get(context.Background())
	if status.Code(err) == codes.NotFound {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	eval, err := decode[model.AgentEvaluation](doc.Data())
	if err != nil {
		return nil, err
	}
	return &eval, nil
}

func (r *evaluationRepository) Put(e model.AgentEvaluation) error {
	data, err := encode(e)
	if err != nil {
		return err
	}
	_, err = r.store.collection(CollectionEvaluations).Doc(docID(e.UpdateKey)).Set(context.Background(), data)
	return err
}

func (r *evaluationRepository) List() ([]model.AgentEvaluation, error) {
	docs, err := r.store.collection(CollectionEvaluations).Documents(context.Background()).GetAll()
	if err != nil {
		return nil, err
	}
	out := make([]model.AgentEvaluation, 0, len(docs))
	for _, doc := range docs {
		eval, err := decode[model.AgentEvaluation](doc.Data())
		if err != nil {
			return nil, err
		}
		out = append(out, eval)
	}
	return out, nil
}

type improvementRepository struct {
	store *store
}

// NewImprovementRepository は Firestore を基盤とする ImprovementRepository を返す。
func NewImprovementRepository(client *cloudfirestore.Client) repository.ImprovementRepository {
	return NewImprovementRepositoryWithOptions(client)
}

// NewImprovementRepositoryWithOptions は options を適用した ImprovementRepository を返す。
func NewImprovementRepositoryWithOptions(client *cloudfirestore.Client, opts ...RepositoryOption) repository.ImprovementRepository {
	return &improvementRepository{store: newStore(client, opts...)}
}

func (r *improvementRepository) PutFailure(f model.FailureCase) error {
	data, err := encode(f)
	if err != nil {
		return err
	}
	_, _, err = r.store.collection(CollectionFailures).Add(context.Background(), data)
	return err
}

func (r *improvementRepository) ListFailures() ([]model.FailureCase, error) {
	docs, err := r.store.collection(CollectionFailures).Documents(context.Background()).GetAll()
	if err != nil {
		return nil, err
	}
	out := make([]model.FailureCase, 0, len(docs))
	for _, doc := range docs {
		failure, err := decode[model.FailureCase](doc.Data())
		if err != nil {
			return nil, err
		}
		out = append(out, failure)
	}
	return out, nil
}

func (r *improvementRepository) PutImprovement(i model.AgentImprovement) error {
	data, err := encode(i)
	if err != nil {
		return err
	}
	coll := r.store.collection(CollectionImprovements)
	if i.ImprovementID == "" {
		_, _, err = coll.Add(context.Background(), data)
		return err
	}
	_, err = coll.Doc(docID(i.ImprovementID)).Set(context.Background(), data)
	return err
}

func (r *improvementRepository) ListImprovements() ([]model.AgentImprovement, error) {
	docs, err := r.store.collection(CollectionImprovements).Documents(context.Background()).GetAll()
	if err != nil {
		return nil, err
	}
	out := make([]model.AgentImprovement, 0, len(docs))
	for _, doc := range docs {
		improvement, err := decode[model.AgentImprovement](doc.Data())
		if err != nil {
			return nil, err
		}
		out = append(out, improvement)
	}
	return out, nil
}

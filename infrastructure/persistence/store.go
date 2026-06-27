// Package persistence は、唯一の真実として振る舞うローカルの JSON ファイルを使って
// リポジトリのポートを実装する。同じインターフェースの背後で Firestore 実装に
// 差し替えられるよう設計されている。
package persistence

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"sort"
	"sync"

	"github.com/ZONO33LHD/anneal/domain/model"
)

type snapshot struct {
	Updates      map[string]model.DependencyUpdate `json:"updates"`
	Evaluations  map[string]model.AgentEvaluation  `json:"evaluations"`
	Failures     []model.FailureCase               `json:"failures"`
	Improvements []model.AgentImprovement          `json:"improvements"`
}

func emptySnapshot() snapshot {
	return snapshot{
		Updates:     map[string]model.DependencyUpdate{},
		Evaluations: map[string]model.AgentEvaluation{},
	}
}

// DB は、3 つのリポジトリすべてが乗る、JSON を基盤とする共有ストレージである。
// path が空の場合はすべてをメモリ内に保持する（テストで使用）。すべての読み書きは、
// 変更のたびにディスクへフラッシュされるメモリ内スナップショットを経由する。
type DB struct {
	path   string
	mu     sync.Mutex
	snap   snapshot
	loaded bool
}

// NewDB は、path のファイルを基盤とする DB を返す（"" => メモリ内）。
func NewDB(path string) *DB {
	return &DB{path: path, snap: emptySnapshot()}
}

func (d *DB) ensureLoaded() error {
	if d.loaded {
		return nil
	}
	d.loaded = true
	if d.path == "" {
		return nil
	}
	data, err := os.ReadFile(d.path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		return err
	}
	snap := emptySnapshot()
	if err := json.Unmarshal(data, &snap); err != nil {
		return err
	}
	if snap.Updates == nil {
		snap.Updates = map[string]model.DependencyUpdate{}
	}
	if snap.Evaluations == nil {
		snap.Evaluations = map[string]model.AgentEvaluation{}
	}
	d.snap = snap
	return nil
}

func (d *DB) flush() error {
	if d.path == "" {
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(d.path), 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(d.snap, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(d.path, data, 0o644)
}

// --- update 操作 ---

func (d *DB) getUpdate(key string) (*model.DependencyUpdate, error) {
	d.mu.Lock()
	defer d.mu.Unlock()
	if err := d.ensureLoaded(); err != nil {
		return nil, err
	}
	rec, ok := d.snap.Updates[key]
	if !ok {
		return nil, nil
	}
	return &rec, nil
}

func (d *DB) putUpdate(rec model.DependencyUpdate) error {
	d.mu.Lock()
	defer d.mu.Unlock()
	if err := d.ensureLoaded(); err != nil {
		return err
	}
	d.snap.Updates[rec.UpdateKey] = rec
	return d.flush()
}

func (d *DB) listUpdates() ([]model.DependencyUpdate, error) {
	d.mu.Lock()
	defer d.mu.Unlock()
	if err := d.ensureLoaded(); err != nil {
		return nil, err
	}
	out := make([]model.DependencyUpdate, 0, len(d.snap.Updates))
	for _, v := range d.snap.Updates {
		out = append(out, v)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].UpdateKey < out[j].UpdateKey })
	return out, nil
}

// --- evaluation 操作 ---

func (d *DB) getEval(key string) (*model.AgentEvaluation, error) {
	d.mu.Lock()
	defer d.mu.Unlock()
	if err := d.ensureLoaded(); err != nil {
		return nil, err
	}
	e, ok := d.snap.Evaluations[key]
	if !ok {
		return nil, nil
	}
	return &e, nil
}

func (d *DB) putEval(e model.AgentEvaluation) error {
	d.mu.Lock()
	defer d.mu.Unlock()
	if err := d.ensureLoaded(); err != nil {
		return err
	}
	d.snap.Evaluations[e.UpdateKey] = e
	return d.flush()
}

func (d *DB) listEvals() ([]model.AgentEvaluation, error) {
	d.mu.Lock()
	defer d.mu.Unlock()
	if err := d.ensureLoaded(); err != nil {
		return nil, err
	}
	out := make([]model.AgentEvaluation, 0, len(d.snap.Evaluations))
	for _, v := range d.snap.Evaluations {
		out = append(out, v)
	}
	return out, nil
}

// --- failure と improvement 操作 ---

func (d *DB) putFailure(f model.FailureCase) error {
	d.mu.Lock()
	defer d.mu.Unlock()
	if err := d.ensureLoaded(); err != nil {
		return err
	}
	d.snap.Failures = append(d.snap.Failures, f)
	return d.flush()
}

func (d *DB) listFailures() ([]model.FailureCase, error) {
	d.mu.Lock()
	defer d.mu.Unlock()
	if err := d.ensureLoaded(); err != nil {
		return nil, err
	}
	return append([]model.FailureCase(nil), d.snap.Failures...), nil
}

func (d *DB) putImprovement(i model.AgentImprovement) error {
	d.mu.Lock()
	defer d.mu.Unlock()
	if err := d.ensureLoaded(); err != nil {
		return err
	}
	d.snap.Improvements = append(d.snap.Improvements, i)
	return d.flush()
}

func (d *DB) listImprovements() ([]model.AgentImprovement, error) {
	d.mu.Lock()
	defer d.mu.Unlock()
	if err := d.ensureLoaded(); err != nil {
		return nil, err
	}
	return append([]model.AgentImprovement(nil), d.snap.Improvements...), nil
}

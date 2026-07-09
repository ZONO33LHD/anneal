package gateway

import (
	"context"

	"github.com/ZONO33LHD/anneal/domain/model"
)

// Dependency はマニフェストから検出された依存関係を表す。
type Dependency struct {
	Name           string
	CurrentVersion string
	IsDev          bool
	Ecosystem      model.Ecosystem
}

// Ecosystem はマニフェストの読み取りと、それへのバージョン更新の適用方法を知る。
// マニフェストの読み取り（Detect/Scan）は ManifestSource 経由で行い、ローカル FS と
// GitHub Contents API を差し替え可能にする。ApplyUpdate（書き込み）は実モードでは
// GitHub Git Data API 側が担うため、ここではローカルパスのまま据え置く。
type Ecosystem interface {
	ID() model.Ecosystem
	Detect(ctx context.Context, src ManifestSource) (bool, error)
	Scan(ctx context.Context, src ManifestSource) ([]Dependency, error)
	ApplyUpdate(repoPath, name, target string) ([]string, error)
}

// EcosystemProvider はリポジトリに該当するエコシステムを発見する。
type EcosystemProvider interface {
	ForRepo(ctx context.Context, src ManifestSource) ([]Ecosystem, error)
	ByID(id model.Ecosystem) Ecosystem
}

// SourceScanner はリポジトリのソース内でパッケージが使われている箇所を見つける。
// 走査に失敗した場合は error を返す（読み取り失敗を「利用箇所なし」と取り違えて
// リスクを過小評価しないため）。
type SourceScanner interface {
	UsageSites(repoPath, packageName string) ([]string, error)
}

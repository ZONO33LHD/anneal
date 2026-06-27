package gateway

import "github.com/ZONO33LHD/anneal/domain/model"

// Dependency はマニフェストから検出された依存関係を表す。
type Dependency struct {
	Name           string
	CurrentVersion string
	IsDev          bool
	Ecosystem      model.Ecosystem
}

// Ecosystem はマニフェストの読み取りと、それへのバージョン更新の適用方法を知る。
type Ecosystem interface {
	ID() model.Ecosystem
	Detect(repoPath string) bool
	Scan(repoPath string) ([]Dependency, error)
	ApplyUpdate(repoPath, name, target string) ([]string, error)
}

// EcosystemProvider はリポジトリに該当するエコシステムを発見する。
type EcosystemProvider interface {
	ForRepo(repoPath string) []Ecosystem
	ByID(id model.Ecosystem) Ecosystem
}

// SourceScanner はリポジトリのソース内でパッケージが使われている箇所を見つける。
type SourceScanner interface {
	UsageSites(repoPath, packageName string) []string
}

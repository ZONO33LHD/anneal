// Package usecase はアプリケーションロジックを保持する。ドメインサービスと
// ポート（repository/gateway）をオーケストレーションし、インフラストラクチャを
// import することは決してない（依存ルール: usecase -> domain）。
package usecase

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/ZONO33LHD/anneal/domain/gateway"
	"github.com/ZONO33LHD/anneal/domain/model"
	"github.com/ZONO33LHD/anneal/domain/repository"
	"github.com/ZONO33LHD/anneal/domain/service"
)

// ScanResult はスキャン結果を要約する。
type ScanResult struct {
	Created []model.DependencyUpdate
	Skipped int
}

// ScanUsecase はアップグレード候補を見つけ、detected レコードを作成する。
type ScanUsecase interface {
	// Run はローカルパスのリポジトリをスキャンする（CLI・demo）。
	Run(ctx context.Context, repoArg, repository string) (ScanResult, error)
	// RunRemote は owner/repo からリモート（Contents API 等）でスキャンする
	// （Cloud Run・push モデル）。
	RunRemote(ctx context.Context, repository string) (ScanResult, error)
}

type scanUsecase struct {
	updates      repository.UpdateRepository
	improvements repository.ImprovementRepository
	metadata     gateway.MetadataSource
	ecosystems   gateway.EcosystemProvider
	repoConfig   gateway.RepoConfigLoader
	notifier     gateway.Notifier
	log          gateway.Logger
	// newSource は repoPath から ManifestSource を作る（ローカル scan 用・LocalFS）。
	newSource gateway.ManifestSourceFactory
	// remoteSource は owner/repo から ManifestSource を作る（リモート scan 用・Contents API）。
	remoteSource gateway.ManifestSourceFactory
	// remoteConfig は owner/repo から .anneal.yml を読む（リモート scan 用）。
	remoteConfig gateway.RepoConfigLoader
}

// NewScanUsecase は scan ユースケースを組み立てる。
func NewScanUsecase(
	updates repository.UpdateRepository,
	improvements repository.ImprovementRepository,
	metadata gateway.MetadataSource,
	ecosystems gateway.EcosystemProvider,
	repoConfig gateway.RepoConfigLoader,
	notifier gateway.Notifier,
	log gateway.Logger,
	newSource gateway.ManifestSourceFactory,
	remoteSource gateway.ManifestSourceFactory,
	remoteConfig gateway.RepoConfigLoader,
) ScanUsecase {
	return &scanUsecase{updates, improvements, metadata, ecosystems, repoConfig, notifier, log, newSource, remoteSource, remoteConfig}
}

// activeVersion は現在有効なエージェント版を返す。採用済み/試用中の改善があれば
// その候補版を、無ければ既定版を使う。新規検出レコードに刻印して A/B 比較を可能にする。
func (s *scanUsecase) activeVersion() string {
	improvements, err := s.improvements.ListImprovements()
	if err != nil {
		return model.CurrentAgentVersion
	}
	return model.ActiveAgentVersion(improvements)
}

// Run はローカルパスのリポジトリをスキャンし、detected レコードを冪等に作成する
// （開発・CLI・demo 向け）。重複防止: 既存のアクティブなレコードを持つ update_key は
// スキップされる。
func (s *scanUsecase) Run(ctx context.Context, repoArg, repoName string) (ScanResult, error) {
	repoPath, _ := filepath.Abs(repoArg)
	if repoName == "" {
		repoName = deriveRepoName(repoPath)
	}
	cfg := s.repoConfig.Load(repoPath)
	src := s.newSource(repoPath)
	return s.scanWith(ctx, cfg, repoName, repoPath, src)
}

// RunRemote は owner/repo だけを起点に、ローカル checkout 無しでスキャンする
// （Cloud Run・対象 repo 起点 push モデル向け）。マニフェストと .anneal.yml は
// リモート（Contents API 等）から取得する。RepoPath は空になる。
func (s *scanUsecase) RunRemote(ctx context.Context, repoRef string) (ScanResult, error) {
	if repoRef == "" {
		return ScanResult{}, fmt.Errorf("scan: repository (owner/repo) is required")
	}
	cfg := s.remoteConfig.Load(repoRef)
	src := s.remoteSource(repoRef)
	return s.scanWith(ctx, cfg, repoRef, "", src)
}

// scanWith は解決済みの設定・ManifestSource を使ってスキャン本体を実行する。
// ローカル/リモート双方の入口（Run/RunRemote）が共有する。
func (s *scanUsecase) scanWith(ctx context.Context, cfg model.RepoConfig, repoName, repoPath string, src gateway.ManifestSource) (ScanResult, error) {
	ecos, err := s.ecosystems.ForRepo(ctx, src)
	if err != nil {
		return ScanResult{}, err
	}
	if len(ecos) == 0 {
		s.log.Warn(ctx, "no supported manifest found", "repo", repoName)
		return ScanResult{}, nil
	}

	var res ScanResult
	for _, eco := range ecos {
		deps, err := eco.Scan(ctx, src)
		if err != nil {
			return res, err
		}
		for _, dep := range deps {
			created, err := s.consider(ctx, cfg, repoName, repoPath, eco.ID(), dep)
			if err != nil {
				return res, err
			}
			if created != nil {
				res.Created = append(res.Created, *created)
			} else {
				res.Skipped++
			}
		}
	}
	return res, nil
}

func (s *scanUsecase) consider(
	ctx context.Context,
	cfg model.RepoConfig,
	repoName, repoPath string,
	eco model.Ecosystem,
	dep gateway.Dependency,
) (*model.DependencyUpdate, error) {
	if cfg.IsIgnored(dep.Name) {
		return nil, nil
	}
	latest, err := s.metadata.LatestVersion(ctx, eco, dep.Name, dep.CurrentVersion)
	if err != nil {
		// メタデータ取得失敗を「更新なし」と取り違えないよう、明示的に縮退として記録する。
		s.log.Warn(ctx, "metadata degraded: latest version lookup failed", "package", dep.Name, "err", err)
		return nil, nil
	}
	if latest == "" || !model.IsUpgrade(dep.CurrentVersion, latest) {
		return nil, nil
	}
	advisories, err := s.metadata.Advisories(ctx, eco, dep.Name, dep.CurrentVersion)
	if err != nil {
		// アドバイザリ取得失敗を「CVEなし」と取り違えないよう警告する（更新自体は継続）。
		s.log.Warn(ctx, "metadata degraded: advisory lookup failed", "package", dep.Name, "err", err)
	}
	var cve *model.CVEInfo
	if len(advisories) > 0 {
		cve = &advisories[0]
	}
	key := model.UpdateKey(repoName, dep.Name, latest)
	if active, _ := s.updates.GetActive(key); active != nil {
		return nil, nil // 重複するアクティブなレコードは作成しない
	}

	updateType := model.ClassifyUpdate(dep.CurrentVersion, latest)
	now := model.NowString()
	rec := model.DependencyUpdate{
		UpdateKey:       key,
		Repository:      repoName,
		RepoPath:        repoPath,
		BaseBranch:      cfg.BaseBranch,
		Ecosystem:       eco,
		PackageName:     dep.Name,
		CurrentVersion:  dep.CurrentVersion,
		TargetVersion:   latest,
		UpdateType:      updateType,
		IsDevDependency: dep.IsDev,
		Priority:        service.ClassifyPriority(updateType, dep.IsDev, cve),
		RiskLevel:       model.RiskLow,
		Status:          model.StateDetected,
		AgentVersion:    s.activeVersion(),
		CVE:             cve,
		History:         []model.TransitionLog{},
		CreatedAt:       now,
		UpdatedAt:       now,
	}
	if err := s.updates.Put(rec); err != nil {
		return nil, err
	}

	level := gateway.NotifyInfo
	title := "Update available: " + dep.Name
	if cve != nil {
		level = gateway.NotifyPriority
		title = fmt.Sprintf("Security update available: %s (%s)", dep.Name, cve.ID)
	}
	notifyOrLog(ctx, s.notifier, s.log, gateway.NotifyMessage{
		Level: level,
		Title: title,
		Body:  fmt.Sprintf("%s → %s (%s, priority=%s)", dep.CurrentVersion, latest, updateType, rec.Priority),
	})
	s.log.Step(ctx, "detected", "update_key", key, "priority", rec.Priority)
	return &rec, nil
}

func deriveRepoName(repoPath string) string {
	parts := strings.FieldsFunc(repoPath, func(r rune) bool { return r == '/' || r == '\\' })
	if len(parts) == 0 {
		return "local/repo"
	}
	return "local/" + parts[len(parts)-1]
}

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
	Run(ctx context.Context, repoArg, repository string) (ScanResult, error)
}

type scanUsecase struct {
	updates    repository.UpdateRepository
	metadata   gateway.MetadataSource
	ecosystems gateway.EcosystemProvider
	repoConfig gateway.RepoConfigLoader
	notifier   gateway.Notifier
	log        gateway.Logger
}

// NewScanUsecase は scan ユースケースを組み立てる。
func NewScanUsecase(
	updates repository.UpdateRepository,
	metadata gateway.MetadataSource,
	ecosystems gateway.EcosystemProvider,
	repoConfig gateway.RepoConfigLoader,
	notifier gateway.Notifier,
	log gateway.Logger,
) ScanUsecase {
	return &scanUsecase{updates, metadata, ecosystems, repoConfig, notifier, log}
}

// Run はリポジトリをスキャンし、detected レコードを冪等に作成する。重複防止:
// 既存のアクティブなレコードを持つ update_key はスキップされる。
func (s *scanUsecase) Run(ctx context.Context, repoArg, repoName string) (ScanResult, error) {
	repoPath, _ := filepath.Abs(repoArg)
	if repoName == "" {
		repoName = deriveRepoName(repoPath)
	}
	cfg := s.repoConfig.Load(repoPath)
	ecos := s.ecosystems.ForRepo(repoPath)
	if len(ecos) == 0 {
		s.log.Warn("no supported manifest found in " + repoPath)
		return ScanResult{}, nil
	}

	var res ScanResult
	for _, eco := range ecos {
		deps, err := eco.Scan(repoPath)
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
		s.log.Warn("metadata degraded: latest version lookup failed for " + dep.Name + ": " + err.Error())
		return nil, nil
	}
	if latest == "" || !model.IsUpgrade(dep.CurrentVersion, latest) {
		return nil, nil
	}
	advisories, err := s.metadata.Advisories(ctx, eco, dep.Name, dep.CurrentVersion)
	if err != nil {
		// アドバイザリ取得失敗を「CVEなし」と取り違えないよう警告する（更新自体は継続）。
		s.log.Warn("metadata degraded: advisory lookup failed for " + dep.Name + ": " + err.Error())
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
		AgentVersion:    model.CurrentAgentVersion,
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
	s.log.Step(fmt.Sprintf("detected %s priority=%s", key, rec.Priority))
	return &rec, nil
}

func deriveRepoName(repoPath string) string {
	parts := strings.FieldsFunc(repoPath, func(r rune) bool { return r == '/' || r == '\\' })
	if len(parts) == 0 {
		return "local/repo"
	}
	return "local/" + parts[len(parts)-1]
}

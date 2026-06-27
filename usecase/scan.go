// Package usecase holds the application logic. It orchestrates domain services
// and the ports (repository/gateway); it never imports infrastructure
// (dependency rule: usecase -> domain).
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

// ScanResult summarizes a scan.
type ScanResult struct {
	Created []model.DependencyUpdate
	Skipped int
}

// ScanUsecase finds upgrade candidates and creates detected records.
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

// NewScanUsecase wires the scan usecase.
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

// Run scans a repository and creates detected records idempotently. Duplicate
// prevention: an update_key with an existing active record is skipped.
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
	if err != nil || latest == "" || !model.IsUpgrade(dep.CurrentVersion, latest) {
		return nil, nil
	}
	advisories, _ := s.metadata.Advisories(ctx, eco, dep.Name, dep.CurrentVersion)
	var cve *model.CVEInfo
	if len(advisories) > 0 {
		cve = &advisories[0]
	}
	key := model.UpdateKey(repoName, dep.Name, latest)
	if active, _ := s.updates.GetActive(key); active != nil {
		return nil, nil // do not create a duplicate active record
	}

	updateType := model.ClassifyUpdate(dep.CurrentVersion, latest)
	now := model.NowString()
	rec := model.DependencyUpdate{
		UpdateKey:       key,
		Repository:      repoName,
		RepoPath:        repoPath,
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
	_ = s.notifier.Notify(ctx, gateway.NotifyMessage{
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

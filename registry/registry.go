// Package registry assembles the whole application. It is the only place that
// knows about concrete infrastructure; it picks real or mock implementations per
// configured secret and wires them into the usecases.
//
// Dependency direction: cmd -> registry -> usecase -> domain (ports). The
// infrastructure adapters are composed here.
package registry

import (
	"github.com/ZONO33LHD/anneal/domain/config"
	"github.com/ZONO33LHD/anneal/domain/gateway"
	"github.com/ZONO33LHD/anneal/domain/repository"
	"github.com/ZONO33LHD/anneal/infrastructure/ecosystem"
	"github.com/ZONO33LHD/anneal/infrastructure/git"
	"github.com/ZONO33LHD/anneal/infrastructure/llm"
	"github.com/ZONO33LHD/anneal/infrastructure/log"
	"github.com/ZONO33LHD/anneal/infrastructure/metadata"
	"github.com/ZONO33LHD/anneal/infrastructure/notify"
	"github.com/ZONO33LHD/anneal/infrastructure/persistence"
	"github.com/ZONO33LHD/anneal/infrastructure/repoconfig"
	"github.com/ZONO33LHD/anneal/usecase"
)

// Registry holds the wired usecases and repositories the CLI needs.
type Registry struct {
	Scan   usecase.ScanUsecase
	Engine usecase.EngineUsecase
	Anneal usecase.AnnealUsecase

	Updates      repository.UpdateRepository
	Evaluations  repository.EvaluationRepository
	Improvements repository.ImprovementRepository
}

// New assembles the application from config, selecting real vs mock providers per
// available secret (unless ForceMock is set).
func New(cfg *config.Config) *Registry {
	logger := log.NewConsole(cfg.Verbose)

	db := persistence.NewDB(cfg.StorePath)
	updates := persistence.NewUpdateRepository(db)
	evals := persistence.NewEvaluationRepository(db)
	improvements := persistence.NewImprovementRepository(db)

	llmGW := pickLLM(cfg)
	gitGW := pickGit(cfg)
	notifier := pickNotifier(cfg)
	meta := pickMetadata(cfg)
	ecosystems := ecosystem.NewProvider()
	scanner := ecosystem.NewScanner()
	repoCfg := repoconfig.NewLoader()

	simulate := gitGW.Name() == "mock"
	logger.Debug("providers: llm=" + llmGW.Name() + " git=" + gitGW.Name() +
		" notify=" + notifier.Name() + " metadata=" + meta.Name())

	return &Registry{
		Scan:         usecase.NewScanUsecase(updates, meta, ecosystems, repoCfg, notifier, logger),
		Engine:       usecase.NewEngineUsecase(updates, evals, improvements, llmGW, gitGW, notifier, ecosystems, scanner, logger, cfg.LowScoreThreshold, simulate),
		Anneal:       usecase.NewAnnealUsecase(evals, improvements, llmGW, notifier, logger, cfg.ImproveWindow, cfg.LowScoreThreshold),
		Updates:      updates,
		Evaluations:  evals,
		Improvements: improvements,
	}
}

func pickLLM(cfg *config.Config) gateway.LLM {
	if !cfg.ForceMock && cfg.GeminiAPIKey != "" {
		return llm.NewGemini(cfg.GeminiAPIKey, cfg.GeminiModel)
	}
	return llm.NewMock()
}

func pickGit(cfg *config.Config) gateway.Git {
	if !cfg.ForceMock && cfg.GitHubToken != "" {
		return git.NewGitHub(cfg.GitHubToken)
	}
	return git.NewMock()
}

func pickNotifier(cfg *config.Config) gateway.Notifier {
	if !cfg.ForceMock && cfg.SlackWebhookURL != "" {
		return notify.NewSlack(cfg.SlackWebhookURL)
	}
	return notify.NewConsole()
}

func pickMetadata(cfg *config.Config) gateway.MetadataSource {
	if !cfg.ForceMock && (cfg.GeminiAPIKey != "" || cfg.GitHubToken != "") {
		return metadata.NewOSV()
	}
	return metadata.NewMock()
}

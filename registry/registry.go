// Package registry はアプリケーション全体を組み立てます。ここは具体的な infrastructure を
// 知っている唯一の場所であり、設定された secret ごとに実装とモックのいずれかを選択し、
// usecase へ配線します。
//
// 依存方向: cmd -> registry -> usecase -> domain (ports)。infrastructure の
// アダプタはここで合成されます。
package registry

import (
	"context"

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

// Registry は CLI が必要とする、配線済みの usecase と repository を保持します。
type Registry struct {
	Scan   usecase.ScanUsecase
	Engine usecase.EngineUsecase
	Anneal usecase.AnnealUsecase

	Updates      repository.UpdateRepository
	Evaluations  repository.EvaluationRepository
	Improvements repository.ImprovementRepository
}

// New は config からアプリケーションを組み立て、利用可能な secret ごとに実装とモックの
// provider を選択します（ForceMock が設定されている場合を除く）。
func New(cfg *config.Config) *Registry {
	logger := log.New(log.Options{Verbose: cfg.Verbose, JSON: cfg.LogJSON})

	db := persistence.NewDB(cfg.StorePath)
	updates := persistence.NewUpdateRepository(db)
	evals := persistence.NewEvaluationRepository(db)
	improvements := persistence.NewImprovementRepository(db)

	llmGW := pickLLM(cfg)
	gitGW := pickGit(cfg, logger)
	notifier := pickNotifier(cfg)
	meta := pickMetadata(cfg)
	ecosystems := ecosystem.NewProvider()
	scanner := ecosystem.NewScanner()
	repoCfg := repoconfig.NewLoader()

	// simulate は git バックエンドがモックのときだけ true（人間承認ゲート/CI を自動進行
	// させデモを完結させる）。プロバイダ名の文字列ではなく具体型で判定する。
	_, simulate := gitGW.(*git.Mock)
	logger.Debug(context.Background(), "providers: llm="+llmGW.ProviderName()+" git="+gitGW.ProviderName()+
		" notify="+notifier.ProviderName()+" metadata="+meta.ProviderName())

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

func pickGit(cfg *config.Config, logger gateway.Logger) gateway.Git {
	if !cfg.ForceMock && cfg.GitHubToken != "" {
		return git.NewGitHub(cfg.GitHubToken, logger)
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
	// OSV / npm レジストリ / Go module proxy はいずれも匿名で利用できるため、
	// ForceMock のときだけオフラインのモックを使い、それ以外は実データを参照する。
	// （以前は secret の有無で mock に落ちており、通常 scan が実際の更新/CVE を
	// 見逃していた。）
	if cfg.ForceMock {
		return metadata.NewMock()
	}
	return metadata.NewOSV()
}

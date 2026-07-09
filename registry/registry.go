// Package registry はアプリケーション全体を組み立てます。ここは具体的な infrastructure を
// 知っている唯一の場所であり、設定された secret ごとに実装とモックのいずれかを選択し、
// usecase へ配線します。
//
// 依存方向: cmd -> registry -> usecase -> domain (ports)。infrastructure の
// アダプタはここで合成されます。
package registry

import (
	"context"
	"fmt"
	"strings"

	"github.com/ZONO33LHD/anneal/domain/config"
	"github.com/ZONO33LHD/anneal/domain/gateway"
	"github.com/ZONO33LHD/anneal/domain/policy"
	"github.com/ZONO33LHD/anneal/domain/repository"
	"github.com/ZONO33LHD/anneal/infrastructure/ecosystem"
	"github.com/ZONO33LHD/anneal/infrastructure/git"
	"github.com/ZONO33LHD/anneal/infrastructure/llm"
	"github.com/ZONO33LHD/anneal/infrastructure/log"
	"github.com/ZONO33LHD/anneal/infrastructure/metadata"
	"github.com/ZONO33LHD/anneal/infrastructure/notify"
	"github.com/ZONO33LHD/anneal/infrastructure/persistence"
	firestorep "github.com/ZONO33LHD/anneal/infrastructure/persistence/firestore"
	"github.com/ZONO33LHD/anneal/infrastructure/prompt"
	"github.com/ZONO33LHD/anneal/infrastructure/repoconfig"
	"github.com/ZONO33LHD/anneal/usecase"
)

// Registry は CLI が必要とする、配線済みの usecase と repository を保持します。
type Registry struct {
	Scan      usecase.ScanUsecase
	Engine    usecase.EngineUsecase
	Anneal    usecase.AnnealUsecase
	Adoption  usecase.AdoptionUsecase
	Webhook   usecase.WebhookUsecase
	Dashboard usecase.DashboardUsecase
	Logger    gateway.Logger

	// Prompts は版で解決するプロンプトカタログ。4b で engine の LLM 呼び出しへ
	// 配線する（本 PR では構築のみ）。
	Prompts gateway.PromptProvider

	Updates      repository.UpdateRepository
	Evaluations  repository.EvaluationRepository
	Improvements repository.ImprovementRepository

	// closer は backend が抱えるリソース（Firestore クライアント等）を解放する。
	// JSON backend では nil。
	closer func() error
}

// Close は backend が保持するリソースを解放する。長時間動く serve では
// 終了時に必ず呼ぶこと。JSON backend では何もしない。
func (r *Registry) Close() error {
	if r.closer == nil {
		return nil
	}
	return r.closer()
}

// repoSet は選択した backend のリポジトリ群と、その解放処理をまとめる。
type repoSet struct {
	updates      repository.UpdateRepository
	evaluations  repository.EvaluationRepository
	improvements repository.ImprovementRepository
	closer       func() error
}

// New は config からアプリケーションを組み立て、利用可能な secret ごとに実装とモックの
// provider を選択します（ForceMock が設定されている場合を除く）。
func New(cfg *config.Config) (*Registry, error) {
	logger := log.New(log.Options{Verbose: cfg.Verbose, JSON: cfg.LogJSON})

	repos, err := pickRepositories(cfg)
	if err != nil {
		return nil, err
	}
	updates, evals, improvements := repos.updates, repos.evaluations, repos.improvements

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
	prompts := prompt.NewRepoCatalog(improvements)
	engine := usecase.NewEngineUsecase(updates, evals, improvements, llmGW, prompts, gitGW, notifier, ecosystems, scanner, logger, cfg.LowScoreThreshold, simulate)

	// リモート scan（owner/repo 起点）は GitHub Contents API でマニフェスト/.anneal.yml を
	// 取得する。ローカル checkout を持たない Cloud Run 経路で使う。読み取り専用の
	// ContentsToken を最小権限で使い、未指定なら PR 作成用の GitHubToken にフォールバック
	// する（public repo は空でも可）。
	contentsToken := cfg.ContentsToken
	if contentsToken == "" {
		contentsToken = cfg.GitHubToken
	}
	remoteSource := func(repoRef string) gateway.ManifestSource {
		owner, repo, _ := strings.Cut(repoRef, "/")
		return git.NewContentsSource(contentsToken, owner, repo, "")
	}
	remoteCfg := repoconfig.NewRemoteLoader(remoteSource)

	return &Registry{
		Scan:         usecase.NewScanUsecase(updates, improvements, meta, ecosystems, repoCfg, notifier, logger, ecosystem.NewLocalFS, remoteSource, remoteCfg),
		Engine:       engine,
		Anneal:       usecase.NewAnnealUsecase(evals, improvements, llmGW, notifier, logger, cfg.ImproveWindow, cfg.LowScoreThreshold),
		Adoption:     usecase.NewAdoptionUsecase(evals, improvements, notifier, logger, policy.MinCanarySample, policy.CanaryRegressionMargin),
		Webhook:      usecase.NewWebhookUsecase(updates, engine, logger),
		Dashboard:    usecase.NewDashboardUsecase(updates, evals, improvements),
		Logger:       logger,
		Prompts:      prompts,
		Updates:      updates,
		Evaluations:  evals,
		Improvements: improvements,
		closer:       repos.closer,
	}, nil
}

func pickRepositories(cfg *config.Config) (*repoSet, error) {
	backend := strings.TrimSpace(cfg.StoreBackend)
	if backend == "" || cfg.ForceMock {
		backend = config.StoreBackendJSON
	}

	switch backend {
	case config.StoreBackendJSON:
		db := persistence.NewDB(cfg.StorePath)
		return &repoSet{
			updates:      persistence.NewUpdateRepository(db),
			evaluations:  persistence.NewEvaluationRepository(db),
			improvements: persistence.NewImprovementRepository(db),
		}, nil
	case config.StoreBackendFirestore:
		if cfg.FirestoreProjectID == "" {
			return nil, fmt.Errorf("firestore backend requires ANNEAL_FIRESTORE_PROJECT or GOOGLE_CLOUD_PROJECT")
		}
		client, err := firestorep.NewClient(context.Background(), cfg.FirestoreProjectID)
		if err != nil {
			return nil, err
		}
		opt := firestorep.WithCollectionPrefix(cfg.FirestoreCollectionPrefix)
		return &repoSet{
			updates:      firestorep.NewUpdateRepositoryWithOptions(client, opt),
			evaluations:  firestorep.NewEvaluationRepositoryWithOptions(client, opt),
			improvements: firestorep.NewImprovementRepositoryWithOptions(client, opt),
			closer:       client.Close,
		}, nil
	default:
		return nil, fmt.Errorf("unknown store backend: %s", backend)
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

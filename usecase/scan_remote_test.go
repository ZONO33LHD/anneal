package usecase_test

import (
	"context"
	"testing"

	"github.com/ZONO33LHD/anneal/domain/gateway"
	"github.com/ZONO33LHD/anneal/domain/model"
	"github.com/ZONO33LHD/anneal/infrastructure/ecosystem"
	"github.com/ZONO33LHD/anneal/infrastructure/notify"
	"github.com/ZONO33LHD/anneal/infrastructure/persistence"
	"github.com/ZONO33LHD/anneal/infrastructure/repoconfig"
	"github.com/ZONO33LHD/anneal/usecase"
)

// fakeManifestSource は owner/repo のリモート scan を、決められたファイル内容で模す。
type fakeManifestSource struct {
	files map[string]string
}

func (f fakeManifestSource) Exists(_ context.Context, rel string) (bool, error) {
	_, ok := f.files[rel]
	return ok, nil
}
func (f fakeManifestSource) ReadFile(_ context.Context, rel string) ([]byte, error) {
	return []byte(f.files[rel]), nil
}

// fakeMetadata は常に指定の最新版と、CVE なしを返す。
type fakeMetadata struct{ latest string }

func (fakeMetadata) ProviderName() string { return "fake" }
func (m fakeMetadata) LatestVersion(context.Context, model.Ecosystem, string, string) (string, error) {
	return m.latest, nil
}
func (fakeMetadata) Advisories(context.Context, model.Ecosystem, string, string) ([]model.CVEInfo, error) {
	return nil, nil
}

// RunRemote は owner/repo だけを起点に、リモート取得した package.json から更新候補を
// 検出し detected レコードを作る。RepoPath は空（ローカル checkout 無し）で、
// update_key は owner/repo で組まれる。これが効かないと push モデルで scan できない。
func TestRunRemote_DetectsFromRemoteManifest(t *testing.T) {
	db := persistence.NewDB("")
	updates := persistence.NewUpdateRepository(db)
	improvements := persistence.NewImprovementRepository(db)

	src := fakeManifestSource{files: map[string]string{
		"package.json": `{"dependencies":{"axios":"1.6.0"}}`,
	}}
	newRemote := func(string) gateway.ManifestSource { return src }

	uc := usecase.NewScanUsecase(
		updates, improvements,
		fakeMetadata{latest: "1.7.0"},
		ecosystem.NewProvider(),
		repoconfig.NewLoader(),
		notify.NewConsole(),
		webhookNoopLogger{},
		ecosystem.NewLocalFS,        // ローカル用（本テストでは未使用）
		newRemote,                   // リモート source
		repoconfig.NewRemoteLoader(newRemote),
	)

	res, err := uc.RunRemote(context.Background(), "acme/web")
	if err != nil {
		t.Fatalf("RunRemote: %v", err)
	}
	if len(res.Created) != 1 {
		t.Fatalf("created=%d want 1", len(res.Created))
	}
	got := res.Created[0]
	if got.Repository != "acme/web" {
		t.Errorf("repository=%q want acme/web", got.Repository)
	}
	if got.RepoPath != "" {
		t.Errorf("RepoPath=%q want empty (remote scan)", got.RepoPath)
	}
	if got.PackageName != "axios" || got.TargetVersion != "1.7.0" {
		t.Errorf("got %s@%s want axios@1.7.0", got.PackageName, got.TargetVersion)
	}
}

// 空の repository はエラーにする（誤起動を握りつぶさない）。
func TestRunRemote_EmptyRepoErrors(t *testing.T) {
	db := persistence.NewDB("")
	newRemote := func(string) gateway.ManifestSource { return fakeManifestSource{} }
	uc := usecase.NewScanUsecase(
		persistence.NewUpdateRepository(db), persistence.NewImprovementRepository(db),
		fakeMetadata{}, ecosystem.NewProvider(), repoconfig.NewLoader(),
		notify.NewConsole(), webhookNoopLogger{}, ecosystem.NewLocalFS, newRemote,
		repoconfig.NewRemoteLoader(newRemote),
	)
	if _, err := uc.RunRemote(context.Background(), ""); err == nil {
		t.Fatal("empty repository should error")
	}
}

package persistence_test

import (
	"path/filepath"
	"testing"

	"github.com/ZONO33LHD/anneal/domain/model"
	"github.com/ZONO33LHD/anneal/infrastructure/persistence"
	"github.com/ZONO33LHD/anneal/internal/testutil"
)

func TestPersistAndReload(t *testing.T) {
	path := filepath.Join(t.TempDir(), "store.json")
	a := persistence.NewUpdateRepository(persistence.NewDB(path))
	if err := a.Put(testutil.MakeUpdate()); err != nil {
		t.Fatal(err)
	}
	b := persistence.NewUpdateRepository(persistence.NewDB(path))
	all, err := b.List()
	if err != nil {
		t.Fatal(err)
	}
	if len(all) != 1 || all[0].PackageName != "axios" {
		t.Errorf("reload failed: %+v", all)
	}
}

func TestActiveVsTerminal(t *testing.T) {
	repo := persistence.NewUpdateRepository(persistence.NewDB(""))
	active := testutil.MakeUpdate()
	active.Status = model.StatePRCreated
	done := testutil.MakeUpdate()
	done.PackageName, done.TargetVersion = "lodash", "5.0.0"
	done.UpdateKey = model.UpdateKey(done.Repository, "lodash", "5.0.0")
	done.Status = model.StateDone
	_ = repo.Put(active)
	_ = repo.Put(done)

	if got, _ := repo.GetActive(active.UpdateKey); got == nil {
		t.Error("active record should be returned")
	}
	if got, _ := repo.GetActive(done.UpdateKey); got != nil {
		t.Error("terminal record should not be active")
	}
	if list, _ := repo.ListActive(); len(list) != 1 {
		t.Errorf("expected 1 active, got %d", len(list))
	}
}

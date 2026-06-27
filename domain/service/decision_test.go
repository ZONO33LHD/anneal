package service_test

import (
	"testing"

	"github.com/ZONO33LHD/anneal/domain/model"
	"github.com/ZONO33LHD/anneal/domain/service"
	"github.com/ZONO33LHD/anneal/internal/testutil"
)

func lowImpact() *model.ImpactAnalysis {
	return &model.ImpactAnalysis{
		UsageSites: []string{"a.ts"}, AffectedFiles: []string{"a.ts"},
		RiskLevel: model.RiskLow, Confidence: 0.9, Summary: "s",
	}
}

func TestDecide(t *testing.T) {
	t.Run("patch auto", func(t *testing.T) {
		u := testutil.MakeUpdate()
		u.UpdateType, u.Impact = model.Patch, lowImpact()
		if service.Decide(u).Decision != "auto" {
			t.Fail()
		}
	})
	t.Run("major human", func(t *testing.T) {
		u := testutil.MakeUpdate()
		u.UpdateType = model.Major
		if service.Decide(u).Decision != "human" {
			t.Fail()
		}
	})
	t.Run("sensitive human", func(t *testing.T) {
		u := testutil.MakeUpdate()
		u.PackageName, u.UpdateType = "passport", model.Patch
		if service.Decide(u).Decision != "human" {
			t.Fail()
		}
	})
	t.Run("security non-breaking auto", func(t *testing.T) {
		u := testutil.MakeUpdate()
		u.UpdateType, u.Impact = model.Patch, lowImpact()
		u.CVE = &model.CVEInfo{ID: "CVE-X", Severity: "high"}
		if service.Decide(u).Decision != "auto" {
			t.Fail()
		}
	})
	t.Run("security breaking human", func(t *testing.T) {
		u := testutil.MakeUpdate()
		u.UpdateType = model.Minor
		imp := lowImpact()
		imp.HasBreakingChange = true
		u.Impact = imp
		u.CVE = &model.CVEInfo{ID: "CVE-X", Severity: "high"}
		if service.Decide(u).Decision != "human" {
			t.Fail()
		}
	})
	t.Run("high-risk minor human", func(t *testing.T) {
		u := testutil.MakeUpdate()
		u.UpdateType = model.Minor
		imp := lowImpact()
		imp.RiskLevel = model.RiskHigh
		u.Impact = imp
		if service.Decide(u).Decision != "human" {
			t.Fail()
		}
	})
	t.Run("low confidence human", func(t *testing.T) {
		u := testutil.MakeUpdate()
		u.UpdateType = model.Patch
		imp := lowImpact()
		imp.Confidence = 0.3
		u.Impact = imp
		if service.Decide(u).Decision != "human" {
			t.Fail()
		}
	})
}

func TestClassifyPriority(t *testing.T) {
	crit := &model.CVEInfo{ID: "C", Severity: "critical"}
	if service.ClassifyPriority(model.Patch, false, crit) != model.PriorityCritical {
		t.Error("critical")
	}
	if service.ClassifyPriority(model.Minor, true, nil) != model.PriorityLow {
		t.Error("dev")
	}
	if service.ClassifyPriority(model.Major, false, nil) != model.PriorityLow {
		t.Error("major")
	}
	if service.ClassifyPriority(model.Patch, false, nil) != model.PriorityMedium {
		t.Error("patch")
	}
}

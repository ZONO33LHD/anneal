package usecase

import (
	"context"
	"fmt"
	"sort"

	"github.com/ZONO33LHD/anneal/domain/gateway"
	"github.com/ZONO33LHD/anneal/domain/model"
	"github.com/ZONO33LHD/anneal/domain/service"
)

// analyzeStep: detected -> analyzing（影響分析）。
func (e *engine) analyzeStep(ctx context.Context, rec model.DependencyUpdate) (model.DependencyUpdate, error) {
	impact := e.analyzeImpact(ctx, rec)
	next, err := rec.Transition(model.StateAnalyzing, "impact analysis complete")
	if err != nil {
		return rec, err
	}
	next.Impact = &impact
	next.RiskLevel = impact.RiskLevel
	return next, nil
}

// createPRStep: pr_creating -> pr_created（バージョン更新の適用、PR の作成とオープン）。
func (e *engine) createPRStep(ctx context.Context, rec model.DependencyUpdate) (model.DependencyUpdate, error) {
	var changed []string
	var contents map[string][]byte
	if eco := e.ecosystems.ByID(rec.Ecosystem); eco != nil {
		switch {
		case rec.RepoPath != "":
			// ローカル checkout あり: ディスク上のマニフェストを書き換える。
			c, err := eco.ApplyUpdate(rec.RepoPath, rec.PackageName, rec.TargetVersion)
			if err != nil {
				return rec, err
			}
			changed = c
		case e.manifests != nil && rec.Repository != "":
			// ローカル checkout なし（リモート push モデル）: Contents API から読み、
			// メモリ上で更新して内容ベースで PR を作る。
			m, err := eco.ApplyUpdateContent(ctx, e.manifests(rec.Repository), rec.PackageName, rec.TargetVersion)
			if err != nil {
				return rec, err
			}
			contents = m
			for p := range m {
				changed = append(changed, p)
			}
			sort.Strings(changed)
		}
	}
	title, body, branch := e.composePR(ctx, rec, changed)
	base := rec.BaseBranch
	if base == "" {
		base = "main"
	}
	ref, err := e.git.CreateBranchAndPR(ctx, gateway.CreatePROptions{
		Repository:      rec.Repository,
		WorkDir:         rec.RepoPath,
		Base:            base,
		Branch:          branch,
		Title:           title,
		Body:            body,
		ChangedFiles:    changed,
		ChangedContents: contents,
	})
	if err != nil {
		return rec, err
	}

	level := gateway.NotifyInfo
	if rec.CVE != nil {
		level = gateway.NotifyPriority
	}
	conf, risk := "n/a", "n/a"
	if rec.Impact != nil {
		conf = fmt.Sprintf("%.2f", rec.Impact.Confidence)
		risk = string(rec.Impact.RiskLevel)
	}
	notifyOrLog(ctx, e.notifier, e.log, gateway.NotifyMessage{
		Level: level,
		Title: "PR opened: " + title,
		Body:  fmt.Sprintf("Risk %s · Confidence %s", risk, conf),
		URL:   ref.URL,
	})

	next, err := rec.Transition(model.StatePRCreated, "branch + changes + PR created")
	if err != nil {
		return rec, err
	}
	next.Branch = branch
	next.PullRequestURL = ref.URL
	next.PullRequestNumber = ref.Number
	return next, nil
}

// ciStep: ci_running -> ci_passed | ci_failed。CI が未完了なら状態を進めず待つ。
func (e *engine) ciStep(ctx context.Context, rec model.DependencyUpdate) (model.DependencyUpdate, bool, error) {
	attempt := 0
	if rec.CI != nil {
		attempt = rec.CI.Attempts
	}
	shouldFailFirst := rec.UpdateType == model.Minor && rec.Impact != nil && len(rec.Impact.UsageSites) > 0
	res, err := e.git.CheckCI(ctx, gateway.CICheckOptions{
		Repository:      rec.Repository,
		PRNumber:        rec.PullRequestNumber,
		Branch:          rec.Branch,
		Attempt:         attempt,
		ShouldFailFirst: shouldFailFirst,
	})
	if err != nil {
		return rec, false, err
	}
	if !res.Complete {
		e.log.Info(ctx, "CI still pending", "update_key", rec.UpdateKey, "summary", res.LogSummary)
		return rec, false, nil
	}
	if res.Passed {
		next, err := rec.Transition(model.StateCIPassed, "CI succeeded")
		if err != nil {
			return rec, false, err
		}
		next.CI = &model.CIResult{Status: "passed", Attempts: attempt}
		return next, true, nil
	}
	next, err := rec.Transition(model.StateCIFailed, "CI failed")
	if err != nil {
		return rec, false, err
	}
	next.CI = &model.CIResult{Status: "failed", Attempts: attempt, LogSummary: res.LogSummary}
	return next, true, nil
}

// fixStep: ci_failed -> fixing | awaiting_review。
func (e *engine) fixStep(ctx context.Context, rec model.DependencyUpdate) (model.DependencyUpdate, error) {
	logSummary, attempts := "unknown failure", 0
	if rec.CI != nil {
		logSummary, attempts = rec.CI.LogSummary, rec.CI.Attempts
	} else {
		// 壊れた/古いストアで CI が nil のまま ci_failed になっていても、
		// この後の *rec.CI で panic しないよう既定値で補う。
		rec.CI = &model.CIResult{Status: "failed"}
	}
	category, fixable := service.ClassifyCIFailure(logSummary)

	// 失敗が繰り返された場合は自己修復を止め、人間に引き継ぐ。
	if attempts >= 2 {
		next, err := rec.Transition(model.StateAwaitingReview, "repeated CI failures → human")
		if err != nil {
			return rec, err
		}
		ci := *rec.CI
		ci.FailureCategory, ci.Fixable = category, false
		next.CI = &ci
		e.log.Warn(ctx, "repeated CI failures, escalating", "update_key", rec.UpdateKey)
		return next, nil
	}

	if !fixable {
		next, err := rec.Transition(model.StateAwaitingReview, "unfixable: "+string(category))
		if err != nil {
			return rec, err
		}
		ci := *rec.CI
		ci.FailureCategory, ci.Fixable = category, false
		next.CI = &ci
		return next, nil
	}

	if err := e.git.PushFix(ctx, gateway.PushFixOptions{
		Repository:   rec.Repository,
		WorkDir:      rec.RepoPath,
		Branch:       rec.Branch,
		PRNumber:     rec.PullRequestNumber,
		Message:      "fix: address " + string(category) + " after dependency bump",
		ChangedFiles: []string{"(auto-fix)"},
	}); err != nil {
		return rec, err
	}
	summary := e.ciFailureSummary(ctx, logSummary)
	notifyOrLog(ctx, e.notifier, e.log, gateway.NotifyMessage{
		Level: gateway.NotifyInfo,
		Title: "Anneal auto-fixed CI: " + string(category),
		Body:  summary,
		URL:   rec.PullRequestURL,
	})
	next, err := rec.Transition(model.StateFixing, "auto-fix for "+string(category))
	if err != nil {
		return rec, err
	}
	next.CI = &model.CIResult{Status: "running", Attempts: attempts + 1, FailureCategory: category, Fixable: true}
	return next, nil
}

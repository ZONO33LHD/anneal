package service

import (
	"regexp"

	"github.com/ZONO33LHD/anneal/domain/model"
)

type failureRule struct {
	category model.CIFailureCategory
	fixable  bool
	patterns []*regexp.Regexp
}

// Keyword rules for the failure categories, checked in order. Deterministic so
// the fix/no-fix decision is predictable; the human-readable summary is added by
// the usecase via the LLM.
var failureRules = []failureRule{
	{model.FailDependencyConflict, true, compile(`(?i)peer dep`, `ERESOLVE`, `(?i)version conflict`, `(?i)incompatible`)},
	{model.FailAPIBreaking, false, compile(`(?i)is not a function`, `(?i)has no exported member`, `(?i)breaking`, `(?i)signature`)},
	{model.FailTestUpdate, true, compile(`(?i)test.*failed`, `(?i)assertion`, `(?i)expected .* received`, `(?i)snapshot`)},
	{model.FailEnvironment, false, compile(`ENOENT`, `(?i)timeout`, `(?i)network`, `(?i)rate limit`, `ECONNREFUSED`)},
}

func compile(pats ...string) []*regexp.Regexp {
	out := make([]*regexp.Regexp, len(pats))
	for i, p := range pats {
		out[i] = regexp.MustCompile(p)
	}
	return out
}

// ClassifyCIFailure maps a CI log summary to a category and whether Anneal can
// mechanically fix it.
func ClassifyCIFailure(logSummary string) (model.CIFailureCategory, bool) {
	for _, rule := range failureRules {
		for _, p := range rule.patterns {
			if p.MatchString(logSummary) {
				return rule.category, rule.fixable
			}
		}
	}
	return model.FailUnknown, false
}

package nogo

import "regexp"

type Rule struct {
	// Regexp defines all regexp-rules which have to pass in order
	// to pass the rule.
	Regexp     []*regexp.Regexp
	Prefix     string
	Pattern    string
	Negate     bool
	OnlyFolder bool
}

var (
	GitIgnoreRule = MustCompileAll("", []byte(".git"))
)

func (r Rule) MatchPath(path string) Result {
	var match bool
	for _, reg := range r.Regexp {
		match = reg.MatchString(path)
		// All regexp have to match.
		if !match {
			return Result{
				Found: match,
				Rule:  r,
			}
		}
	}

	return Result{
		Found: match,
		Rule:  r,
	}
}

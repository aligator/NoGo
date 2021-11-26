package nogo

import "regexp"

type Rule struct {
	Regexp     *regexp.Regexp
	Prefix     string
	Pattern    string
	Negate     bool
	OnlyFolder bool
}

var (
	GitIgnoreRule = MustCompileAll("", []byte(".git"))
)

func (r Rule) MatchPath(path string) Result {
	match := r.Regexp.MatchString(path)
	return Result{
		Found: match,
		Rule:  r,
	}
}

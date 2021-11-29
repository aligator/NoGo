package nogo

import (
	"regexp"
	"strings"
)

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
	DotGitRule = MustCompileAll("", []byte(".git"))[0]
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

// These bytes won't be in any valid file, so they should be perfectly valid as temporary replacement.
const (
	doubleStar        = "\000"
	singleStar        = "\001"
	questionMark      = "\002"
	negatedMatchStart = "\003"
	matchStart        = "\004"
	matchEnd          = "\005"
	escapedMatchStart = "\006"
	escapedMatchEnd   = "\007"
)

var (
	// findRangeReg matches the replacements of [, [! and ].
	// The ? in the regexp enables ungreedy mode.
	findRangeReg = regexp.MustCompile(`[` + matchStart + negatedMatchStart + `].*?` + matchEnd)
)

// Compile the pattern into a single regexp.
// skip means that this pattern doesn't contain any rule (e.g. just a comment or empty line).
func Compile(prefix string, pattern string) (skip bool, rule Rule, err error) {
	rule = Rule{
		Prefix: prefix,

		// The original pattern of the source file.
		Pattern: pattern,
	}

	// ignoreFs empty lines.
	if len(pattern) == 0 {
		return true, Rule{}, nil
	}

	// ignoreFs lines starting with # as these are comments.
	if pattern[0] == '#' {
		return true, Rule{}, nil
	}

	// Unescape \# to #.
	if strings.HasPrefix(pattern, "\\#") {
		pattern = pattern[1:]
	}

	// ignoreFs spaces except when the last one is escaped: 'something   \ '.
	// TODO: actually I am not sure if this is correct but that's what I understand by
	//  "* Trailing spaces are ignored unless they are quoted with backslash ("\")."
	//  However I don't think that this is very often used.
	if strings.HasSuffix(pattern, "\\ ") {
		pattern = strings.TrimSuffix(pattern, "\\ ") + " "
	} else {
		pattern = strings.TrimRight(pattern, " ")
	}

	// '!' negates the pattern.
	if pattern[0] == '!' {
		rule.Negate = true
		pattern = pattern[1:]
	}

	// If any '/' is at the beginning or middle, it is relative to the prefix.
	// Else it may be anywhere bellow it and we have to apply a wildcard
	if strings.Count(strings.TrimSuffix(pattern, "/"), "/") == 0 {
		pattern = "**/" + strings.TrimPrefix(pattern, "/")
	} else if prefix != "" {
		// In most other cases we have to make sure the prefix ends with a '/'
		prefix = strings.TrimSuffix(prefix, "/") + "/"
	}

	// Replace all special chars with placeholders, then quote the rest.
	// After that the special regexp for that special cases can be replaced.

	pattern = strings.ReplaceAll(pattern, "**", doubleStar)
	pattern = strings.ReplaceAll(pattern, "*", singleStar)
	pattern = strings.ReplaceAll(pattern, "?", questionMark)

	// Re-Replace escaped replacements.
	pattern = strings.ReplaceAll(pattern, `\`+doubleStar, "**")
	pattern = strings.ReplaceAll(pattern, `\`+singleStar, "*")
	pattern = strings.ReplaceAll(pattern, `\`+questionMark, "?")

	pattern = regexp.QuoteMeta(pattern)

	// Unescape and transform character matches.
	// First replace all by the input escaped brackets to ignore them in the next replaces)
	pattern = strings.ReplaceAll(pattern, `\\[`, escapedMatchStart)
	pattern = strings.ReplaceAll(pattern, `\\]`, escapedMatchEnd)

	// Then do the same with the negated one to ignore its bracket in the next replace.
	pattern = strings.ReplaceAll(pattern, `\[!`, negatedMatchStart)
	pattern = strings.ReplaceAll(pattern, `\[`, matchStart)
	pattern = strings.ReplaceAll(pattern, `\]`, matchEnd)
	// Now we can add any new regexp using [ and ] and still
	// Do something with the placeholders later.

	// If any '/' is at the end, it matches only folders.
	// Note, as the input does not show us if it is a folder, the bool
	// is set and it has to be checked separately.
	if strings.HasSuffix(pattern, "/") {
		pattern = strings.TrimSuffix(pattern, "/")
		rule.OnlyFolder = true
	}

	// Check the placeholders:

	// '?' matches any char but '/'.
	pattern = strings.ReplaceAll(pattern, questionMark, "[^/]?")

	// Replace the placeholders:
	// A leading "**" followed by a slash means matches in all directories.
	if strings.HasPrefix(pattern, doubleStar+"/") {
		if prefix == "" {
			pattern = "(.*/)?" + strings.TrimPrefix(pattern, doubleStar+"/")
		} else {
			pattern = "(/.*)?" + strings.TrimPrefix(pattern, doubleStar)

			// Also remove a possible '/' from the prefix so that it concatenates correctly with the wildcard
			prefix = strings.TrimSuffix(prefix, "/")

		}
	}

	// A trailing "/**" matches everything inside.
	if strings.HasSuffix(pattern, "/"+doubleStar) {
		pattern = strings.TrimSuffix(pattern, doubleStar) + ".*"
	}

	// A slash followed by two consecutive asterisks then a slash matches zero or more directories.
	pattern = strings.ReplaceAll(pattern, "/"+doubleStar+"/", ".*/")

	// '*' matches anything but '/'.
	pattern = strings.ReplaceAll(pattern, singleStar, "[^/]*")

	// Now replace all still existing doubleStars and all stars by the single star rule.
	// TODO: Not sure if that is the correct behavior.
	pattern = strings.ReplaceAll(pattern, doubleStar, "[^/]*")

	// Add an additional regexp which checks for non-slash on all range patterns.
	// As the range should not match slashes, but as Go doesn't support look-ahead,
	// I just add a new rule for this.
	additionalPattern := findRangeReg.ReplaceAllString(pattern, `[^/]`)

	finishPattern := func(p string) error {
		// Now replace back the escaped brackets.
		p = strings.ReplaceAll(p, escapedMatchStart, `[`)
		p = strings.ReplaceAll(p, escapedMatchEnd, `]`)
		pattern = strings.ReplaceAll(pattern, negatedMatchStart, "[^")
		pattern = strings.ReplaceAll(pattern, matchStart, "[")
		pattern = strings.ReplaceAll(pattern, matchEnd, "]")

		reg, err := regexp.Compile("^" + regexp.QuoteMeta(prefix) + strings.TrimPrefix(p, "/") + "$")
		if err != nil {
			return err
		}

		rule.Regexp = append(rule.Regexp, reg)
		return nil
	}

	// Skip that additional pattern if nothing was replaced.
	if additionalPattern != pattern {
		err := finishPattern(additionalPattern)
		if err != nil {
			return false, Rule{}, err
		}
	}

	err = finishPattern(pattern)
	if err != nil {
		return false, Rule{}, err
	}

	return false, rule, nil
}

// CompileAll rules in the given data line by line.
// The prefix is added to all rules.
func CompileAll(prefix string, data []byte) ([]Rule, error) {
	rules := make([]Rule, 0)
	lines := strings.Split(string(data), "\n")
	for _, line := range lines {
		// Remove \r on windows.
		line = strings.TrimSuffix(line, "\r")

		skip, rule, err := Compile(prefix, line)
		if err != nil {
			return nil, err
		}

		if !skip {
			rules = append(rules, rule)
		}
	}
	return rules, nil
}

// MustCompileAll does the same as CompileAll but panics on error.
func MustCompileAll(prefix string, data []byte) []Rule {
	rule, err := CompileAll(prefix, data)
	if err != nil {
		panic(err)
	}

	return rule
}

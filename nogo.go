// Package nogo implements gitignore parsing in pure go.
// It supports the official specification. https://git-scm.com/docs/gitignore/2.34.0
//
//  PATTERN FORMAT
//
//    * A blank line matches no files, so it can serve as a separator for readability.
//
//    * A line starting with # serves as a comment. Put a backslash ("\") in front of the first hash for patterns that begin with a hash.
//
//    * Trailing spaces are ignored unless they are quoted with backslash ("\").
//
//    * An optional prefix "!" which negates the pattern; any matching file excluded by a previous pattern will become included again. It is not possible to re-include a file if a parent directory of that file is excluded. Git doesn’t list excluded directories for performance reasons, so any patterns on contained files have no effect, no matter where they are defined. Put a backslash ("\") in front of the first "!" for patterns that begin with a literal "!", for example, "\!important!.txt".
//
//    * The slash / is used as the directory separator. Separators may occur at the beginning, middle or end of the .gitignore search pattern.
//
//    * If there is a separator at the beginning or middle (or both) of the pattern, then the pattern is relative to the directory level of the particular .gitignore file itself. Otherwise the pattern may also matches at any level below the .gitignore level.
//
//    * If there is a separator at the end of the pattern then the pattern will only matches directories, otherwise the pattern can matches both files and directories.
//      For example, a pattern doc/frotz/ matches doc/frotz directory, but not a/doc/frotz directory; however frotz/ matches frotz and a/frotz that is a directory (all paths are relative from the .gitignore file).
//
//    * An asterisk "*" matches anything except a slash. The character "?" matches any one character except "/". The range notation, e.g. [a-zA-Z], can be used to matches one of the characters in a range. See fnmatch(3) and the FNM_PATHNAME flag for a more detailed description.
//
//  Two consecutive asterisks ("**") in patterns matched against full pathname may have special meaning:
//
//    * A leading "**" followed by a slash means matches in all directories. For example, "**/foo" matches file or directory "foo" anywhere, the same as pattern "foo". "**/foo/bar" matches file or directory "bar" anywhere that is directly under directory "foo".
//
//    * A trailing "/**" matches everything inside. For example, "abc/**" matches all files inside directory "abc", relative to the location of the .gitignore file, with infinite depth.
//
//    * A slash followed by two consecutive asterisks then a slash matches zero or more directories. For example, "a/**/b" matches "a/b", "a/x/b", "a/x/y/b" and so on.
//
//    * Other consecutive asterisks are considered regular asterisks and will matches according to the previous rules.
package nogo

import (
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

type Rule struct {
	*regexp.Regexp
	Prefix     string
	Pattern    string
	Negate     bool
	OnlyFolder bool
}

func (r Rule) MatchPath(path string) Result {
	match := r.MatchString(path)
	return Result{
		Matches: match,
		Rule:    r,
	}
}

type group struct {
	prefix string
	rules  []Rule
}

type Option func(noGo *NoGo)

type NoGo struct {
	fs fs.FS

	Groups []group
}

func WithFS(f fs.FS) Option {
	return func(noGo *NoGo) {
		noGo.fs = f
	}
}

func WithIgnoreDotGit() Option {
	return func(noGo *NoGo) {
		_, rule, err := Compile("", "/.git/")
		if err != nil {
			// Something is really wrong... should never happen.
			panic(err)
		}

		err = noGo.AddRule(rule)
		if err != nil {
			// Something is really wrong... should never happen.
			panic(err)
		}
	}
}

func NewGitignore(options ...Option) *NoGo {
	no := &NoGo{}

	WithIgnoreDotGit()(no)

	for _, o := range options {
		o(no)
	}

	if no.fs == nil {
		wd, err := os.Getwd()
		if err != nil {
			panic(err)
		}
		WithFS(os.DirFS(wd))
	}

	return no
}

func New(options ...Option) *NoGo {
	no := &NoGo{}

	no.Apply(options...)

	if no.fs == nil {
		wd, err := os.Getwd()
		if err != nil {
			panic(err)
		}
		WithFS(os.DirFS(wd))
	}

	return no
}

func (n *NoGo) Apply(options ...Option) {
	for _, o := range options {
		o(n)
	}
}

func (n *NoGo) AddAll(fileName string) error {
	return fs.WalkDir(n.fs, ".", func(path string, d fs.DirEntry, err error) error {
		if d.Name() == fileName {
			return n.AddFile(path)
		}
		return nil
	})
}

func (n *NoGo) AddRule(rule Rule) error {
	n.Groups = append(n.Groups, group{
		prefix: rule.Prefix,
		rules:  []Rule{rule},
	})

	return nil
}

func (n *NoGo) AddFile(path string) error {
	file, err := n.fs.Open(path)
	if err != nil {
		return err
	}

	data, err := io.ReadAll(file)
	if err != nil {
		return err
	}

	folder := filepath.Dir(path)
	if folder == "." {
		folder = ""
	}

	rules, err := CompileAll(folder, data)
	if err != nil {
		return err
	}

	n.Groups = append(n.Groups, group{
		prefix: folder,
		rules:  rules,
	})

	return nil
}

type Result struct {
	Matches bool
	Rule
}

func (n *NoGo) MatchPath(path string) Result {
	var match Result

	for _, group := range n.Groups {
		if !strings.HasPrefix(path, group.prefix) {
			continue
		}

		for _, rule := range group.rules {
			res := rule.MatchPath(path)

			if res.Matches {
				match = res
			}
		}
	}

	return match
}

func Compile(prefix string, pattern string) (skip bool, rule Rule, err error) {
	// Just make sure the regexp exists in all cases.
	defer func() {
		if rule.Regexp == nil {
			rule.Regexp = regexp.MustCompile("")
		}
	}()

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
		return true, Rule{Regexp: regexp.MustCompile("")}, nil
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

		// Also remove a possible '/' from the prefix so that it concatenates correctly with the wildcard
		prefix = strings.TrimSuffix(prefix, "/")

	} else if prefix != "" {
		// In most other cases we have to make sure the prefix ends with a '/'
		prefix = strings.TrimSuffix(prefix, "/") + "/"
	}

	// Replace all special chars with placeholders, then quote the rest.
	// After that the special regexp for that special cases can be replaced.
	// These bytes won't be in any valid file, so they should be perfectly valid as replacement.
	const (
		doubleStar        = "\000"
		singleStar        = "\001"
		questionMark      = "\002"
		negatedMatchStart = "\003"
		escapedMatchStart = "\004"
		escapedMatchEnd   = "\005"
	)

	pattern = strings.ReplaceAll(pattern, "**", doubleStar)
	pattern = strings.ReplaceAll(pattern, "*", singleStar)
	pattern = strings.ReplaceAll(pattern, "?", questionMark)

	// Re-Replace escaped replacements.
	pattern = strings.ReplaceAll(pattern, "\\"+doubleStar, "**")
	pattern = strings.ReplaceAll(pattern, "\\"+singleStar, "*")
	pattern = strings.ReplaceAll(pattern, "\\"+questionMark, "?")

	pattern = regexp.QuoteMeta(pattern)

	// Unescape and transform character matches.
	// First replace all by the input escaped brackets to ignore them in the next replaces)
	pattern = strings.ReplaceAll(pattern, "\\\\[", escapedMatchStart)
	pattern = strings.ReplaceAll(pattern, "\\\\]", escapedMatchEnd)

	// THen do the same with the negated one to ignore its bracket in the next replace.
	pattern = strings.ReplaceAll(pattern, "\\[!", negatedMatchStart)
	pattern = strings.ReplaceAll(pattern, "\\[", "[")
	// Replace that one back with the regexp compatible negation.
	pattern = strings.ReplaceAll(pattern, negatedMatchStart, "[^")
	// Replace the bracket ending
	pattern = strings.ReplaceAll(pattern, "\\]", "]")

	// Now replace back the escaped brackets.
	pattern = strings.ReplaceAll(pattern, escapedMatchStart, "[")
	pattern = strings.ReplaceAll(pattern, escapedMatchEnd, "]")

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
			pattern = "(/.*|.*)" + strings.TrimPrefix(pattern, doubleStar)
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
	// TODO: Not sure if that is the correct way.
	pattern = strings.ReplaceAll(pattern, doubleStar, "[^/]*")

	rule.Regexp, err = regexp.Compile("^" + regexp.QuoteMeta(prefix) + strings.TrimPrefix(pattern, "/") + "$")
	if err != nil {
		return false, Rule{}, err
	}

	return false, rule, nil
}

func CompileAll(prefix string, data []byte) ([]Rule, error) {
	rules := make([]Rule, 0)
	lines := strings.Split(string(data), "\n")
	for _, line := range lines {
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

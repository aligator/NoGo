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

type group struct {
	prefix string
	rules  []Rule
}

type Option func(noGo *NoGo)

type NoGo struct {
	fs              fs.FS
	groups          []group
	ignoreFileNames []string
	matchNoParents  bool
}

// Apply options to NoGo.
// This is also possible when instantiating with New.
func (n *NoGo) Apply(options ...Option) {
	for _, o := range options {
		o(n)
	}
}

// WithFS is an option which injects a filesystem.
// If this is not passed, the current working directory is used.
func WithFS(f fs.FS) Option {
	return func(noGo *NoGo) {
		noGo.fs = f
	}
}

// WithRules can be used to add rules without an extra call to an Add method.
// This can be used to add predefined rules like the GitIgnoreRule.
// For example:
//  n := nogo.New(nogo.WithRules(nogo.GitIgnoreRule...))
func WithRules(rules ...Rule) Option {
	return func(noGo *NoGo) {
		noGo.AddRules(rules...)
	}
}

// WithoutMatchParents disables the time-consuming check for all parents.
// You can use this if you know that no file gets checked without also all parents being checked before.
//
// As the parent-check is time-consuming it is for example better to disable that check when using Walk function.
// (nogo.WalkDir and nogo.WalkAfero do this automatically).
//
// Example:
//  Folder1
//   - File1
//  .gitignore -> Rule: "/Folder1"
//
// If the gitignore contains the rule "/Folder1" and you check the file
// `/Folder1/File1`, you will get a correct match.
//
// But if you check the file WITH "WithoutMatchParents", the file will not match
// as it itself is not in any ignore-list and the parent folder does not get checked.
//
// When doing file traversal with a Walk method, this doesn't matter
// as the Folder1 won't be read and therefore /Folder1/File1 won't be read either.
//
// But when checking only the file /Folder1/File1 directly, you will NOT want "WithoutMatchParents".
func WithoutMatchParents() Option {
	return func(noGo *NoGo) {
		noGo.matchNoParents = true
	}
}

// NewGitignore creates a new NoGo which is already preset for the use
// with .gitignore files, as this is very common.
//
// It not only sets the ignoreFileName to ".gitignore" but
// also adds the GitIgnoreRule to avoid .git folders.
func NewGitignore(options ...Option) *NoGo {
	no := &NoGo{
		ignoreFileNames: []string{".gitignore"},
	}

	no.AddRules(GitIgnoreRule...)

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

// New creates a NoGo instance which works for the given ignoreFileNames.
// You can pass additional options if needed.
func New(ignoreFileNames []string, options ...Option) *NoGo {
	no := &NoGo{
		ignoreFileNames: ignoreFileNames,
	}

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

// AddAll ignore files which can be found.
// It only loads ignore files which are not ignored itself by another file.
func (n *NoGo) AddAll() error {
	return fs.WalkDir(n.fs, ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		_, err = n.walkFN(path, d.IsDir())
		return err
	})
}

// AddRules to NoGo which are already compiled.
func (n *NoGo) AddRules(rules ...Rule) {
	for _, rule := range rules {
		n.groups = append(n.groups, group{
			prefix: rule.Prefix,
			rules:  []Rule{rule},
		})
	}
}

// AddFile reads the given file and tries to load the content as a ignore file.
// It does not check the filename. So you can add any file, independently of
// the configured ignoreFileNames.
//
// The folder of the give filepath is used as Prefix for the rules.
//
// Note that the order in which rules are added is very important.
// You should always first add the rules of parent folders and then of the
// children folders.
// TODO: in the future the rules could be re-sorted based on the prefix names.
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

	n.groups = append(n.groups, group{
		prefix: folder,
		rules:  rules,
	})

	return nil
}

// MatchPath calculates if the path matches any rule.
//
// It does the same as MatchPathBecauseNoStat, but it itself determines if path is a
// directory.
// If you already have that information, use MatchPathBecauseNoStat.
//
// As it has to query the filesystem, it may return an error
// due to possible filesystem errors.
func (n *NoGo) MatchPath(path string) (bool, error) {
	match, _, err := n.MatchPathBecause(path)
	return match, err
}

// MatchPathBecause calculates if the path matches any rule.
// It returns the match but also a result, where the match was calculated from.
// Use MatchPath or MatchPathNoStat if you do not need the cause.
//
// It does the same as MatchPathBecauseNoStat, but it itself determines if path is a
// directory.
// If you already have that information, use MatchPathBecauseNoStat.
//
// As it has to query the filesystem, it may return an error
// due to possible filesystem errors.
func (n *NoGo) MatchPathBecause(path string) (match bool, because Result, err error) {
	statFS, ok := n.fs.(fs.StatFS)
	var isDir bool
	if ok {
		stat, err := statFS.Stat(path)
		if err != nil {
			return false, Result{}, err
		}
		isDir = stat.IsDir()
	} else {
		f, err := n.fs.Open(path)
		if err != nil {
			return false, Result{}, err
		}

		stat, err := f.Stat()
		if err != nil {
			return false, Result{}, err
		}
		isDir = stat.IsDir()
	}

	match, because = n.MatchPathBecauseNoStat(path, isDir)
	return match, because, nil
}

// MatchPathNoStat calculates if the path matches any rule.
//
// It does the same as MatchPath, but you have to pass yourself if the
// path is a directory.
// This can be useful to avoid unneeded Stat-calls.
// (e.g. in a Walk function where you already have that information)
func (n *NoGo) MatchPathNoStat(path string, isDir bool) bool {
	match, _ := n.MatchPathBecauseNoStat(path, isDir)
	return match
}

// MatchPathBecauseNoStat calculates if the path matches any rule.
// It returns the match but also a result, where the match was calculated from.
// Use MatchPath or MatchPathNoStat if you do not need the cause.
//
// It does the same as MatchPathBecause, but you have to pass yourself if the
// path is a directory.
// This can be useful to avoid unneeded Stat-calls.
// (e.g. in a Walk function where you already have that information)
func (n *NoGo) MatchPathBecauseNoStat(path string, isDir bool) (match bool, because Result) {
	pathToCheck := []string{path}
	if !n.matchNoParents {
		pathToCheck = strings.Split(filepath.ToSlash(path), "/")
	}

	path = ""
	for i, p := range pathToCheck {
		path = filepath.Join(path, p)

		for _, g := range n.groups {
			if !strings.HasPrefix(path, g.prefix) {
				continue
			}

			for _, rule := range g.rules {
				newRes := rule.MatchPath(path)

				if newRes.Found {
					because = newRes
					because.ParentMatch = i < len(pathToCheck)-1
				}
			}
		}
	}

	if because.Found && because.OnlyFolder && !isDir && because.ParentMatch {
		return false, because
	}

	if because.Found && because.Negate {
		return false, because
	}

	return because.Found, because
}

// Compile the pattern into a single regexp.
// skip means that this pattern doesn't contain any rule (e.g. just a comment or empty line).zz
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

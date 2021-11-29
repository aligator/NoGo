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
	"path/filepath"
	"strings"
)

type group struct {
	prefix string
	rules  []Rule
}

type NoGo struct {
	groups []group
}

// New creates a NoGo instance which works for the given ignoreFileNames.
// You can pass additional options if needed.
func New(rules ...Rule) *NoGo {
	n := &NoGo{}
	n.AddRules(rules...)
	return n
}

// AddFromFS ignore files which can be found in the given fsys.
// It only loads ignore files which are not ignored itself by another file.
func (n *NoGo) AddFromFS(fsys fs.FS, ignoreFilename string) error {
	return fs.WalkDir(fsys, ".", func(path string, d fs.DirEntry, err error) error {
		_, err = n.WalkFunc(fsys, ignoreFilename, path, d.IsDir(), err)
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

// AddFile reads the given file and tries to load the content as an ignore file.
// It does not check the filename. So you can add any file, independently of
// the configured ignoreFileNames.
//
// The folder of the give filepath is used as Prefix for the rules.
//
// Note that the order in which rules are added is very important.
// You should always first add the rules of parent folders and then of the
// children folders.
// TODO: in the future the rules could be re-sorted based on the prefix names.
func (n *NoGo) AddFile(fsys fs.FS, path string) error {
	file, err := fsys.Open(path)
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

// Match calculates if the path matches any rule.
// It does the same as MatchBecause but only returns the boolean
// for more easy in-if usage.
func (n *NoGo) Match(path string, isDir bool) bool {
	match, _ := n.MatchBecause(path, isDir)
	return match
}

// MatchBecause calculates if the path matches any rule.
// It returns the match but also a result, where the match was calculated from.
// Use Match if you do not need the cause.
//
// You have to pass if the path is a directory or not using isDir.
func (n *NoGo) MatchBecause(path string, isDir bool) (match bool, because Result) {
	return n.match(path, isDir, false)
}

// MatchWithoutParents does the same as MatchBecause and Match but it
// disables a time-consuming check of all parent folder rules.
// This is faster, but it results in wrong results if the check of the parents
// is not done in another way.
//
// DO NOT USE THIS IF YOU DON'T UNDERSTAND HOW IT WORKS.
// Use MatchBecause or Match instead.
//
// You can use this if you know that no file gets checked without also
// all parents being checked before.
//
// As the parent-check is time-consuming it is for example better to disable
// that check when using Walk function.
// (NoGo.WalkDirFunc and NoGo.WalkAferoFunc use it for example).
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
func (n *NoGo) MatchWithoutParents(path string, isDir bool) (match bool, because Result) {
	return n.match(path, isDir, true)
}

func (n *NoGo) match(path string, isDir bool, noParents bool) (match bool, because Result) {
	pathToCheck := []string{path}
	if !noParents {
		// Convert to slash for windows compatibility before splitting.
		pathToCheck = strings.Split(filepath.ToSlash(path), "/")
	}

	path = ""
	for i, p := range pathToCheck {
		// Convert to slash for windows compatibility.
		path = filepath.ToSlash(filepath.Join(path, p))

		for _, g := range n.groups {
			if !strings.HasPrefix(path, g.prefix) {
				continue
			}

			for _, rule := range g.rules {
				newRes := rule.MatchPath(path)

				if newRes.Found && ((newRes.OnlyFolder && isDir) || !newRes.OnlyFolder) {
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

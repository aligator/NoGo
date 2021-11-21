package nogo

import (
	"errors"
	"io/fs"
	"path/filepath"
)

type ignoreFS struct {
	fs.FS
	*NoGo
}

// WalkDir walks the file tree rooted at root, calling fn for each file or
// directory in the tree, including root.
// This implementation skips all folders and files according to the ignore
// files found in the file-tree.
//
// All options you pass, are applied to the internal NoGo instance.
//
// All errors that arise visiting files and directories are filtered by fn:
// see the fs.WalkDirFunc documentation for details.
//
// The files are walked in lexical order, which makes the output deterministic
// but requires WalkDir to read an entire directory into memory before proceeding
// to walk that directory.
//
// WalkDir does not follow symbolic links found in directories,
// but if root itself is a symbolic link, its target will be walked.
func WalkDir(fsys fs.FS, ignoreFileName string, root string, fn fs.WalkDirFunc, options ...Option) error {
	n := New(WithFS(fsys))
	n.Apply(options...)

	ifs := &ignoreFS{
		NoGo: n,
		FS:   fsys,
	}

	return fs.WalkDir(ifs, root, func(path string, d fs.DirEntry, err error) error {
		if path != "." {
			match := ifs.MatchPath(path)
			if match.OnlyFolder && !d.IsDir() {
				match.Matches = false
			}

			// If the rule is a negation rule, still proceed.
			if match.Matches && !match.Rule.Negate {
				if d.IsDir() {
					return fs.SkipDir
				}
				return nil
			}
		}

		if d.IsDir() {
			// Add the ignore files when touching a new folder.
			// That way we do not need to read all ignore files in advance.
			// THis works because WalkDir runs in a deterministic way.
			err := ifs.AddFile(filepath.Join(path, ignoreFileName))
			if err != nil && !errors.Is(err, fs.ErrNotExist) {
				return err
			}
		}

		return fn(path, d, err)
	})
}

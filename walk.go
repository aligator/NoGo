package nogo

import (
	"io/fs"
	"path/filepath"

	"github.com/spf13/afero"
)

type ignoreFS struct {
	fs.FS
	*NoGo
}

func (n *NoGo) walkFN(path string, isDir bool) (bool, error) {
	if path != "." {
		// If the rule is a negation rule, still proceed.
		if n.MatchPathNoStat(path, isDir) {
			if isDir {
				return false, fs.SkipDir
			}
			return false, nil
		}
	}

	if !isDir {
		name := filepath.Base(path)
		for _, ignoreFileName := range n.ignoreFileNames {
			if name != ignoreFileName {
				continue
			}

			err := n.AddFile(path)
			if err != nil {
				return false, err
			}
			break
		}

	}

	return true, nil
}

// AferoWalk walks the file tree rooted at root, calling walkFn for each file or
// directory in the tree, including root. All errors that arise visiting files
// and directories are filtered by walkFn. The files are walked in lexical
// order, which makes the output deterministic but means that for very
// large directories Walk can be inefficient.
// Walk does not follow symbolic links.
//
// This implementation skips all folders and files according to the ignore
// files found in the file-tree.
//
// All options you pass, are applied to the internal NoGo instance.
func AferoWalk(ignoreFileNames []string, fsys afero.Fs, fn filepath.WalkFunc, options ...Option) error {
	iofs := afero.NewIOFS(fsys)
	n := New(ignoreFileNames, WithFS(iofs), WithoutMatchParents())
	n.Apply(options...)

	ifs := &ignoreFS{
		NoGo: n,
		FS:   iofs,
	}

	return afero.Walk(fsys, ".", func(path string, info fs.FileInfo, err error) error {
		if err != nil {
			return err
		}

		ok, err := ifs.walkFN(path, info.IsDir())
		if err != nil {
			return err
		}

		if ok {
			return fn(path, info, err)
		}

		return nil
	})
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
func WalkDir(ignoreFileNames []string, fsys fs.FS, root string, fn fs.WalkDirFunc, options ...Option) error {
	n := New(ignoreFileNames, WithFS(fsys), WithoutMatchParents())
	n.Apply(options...)

	ifs := &ignoreFS{
		NoGo: n,
		FS:   fsys,
	}

	return fs.WalkDir(ifs, root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		ok, err := ifs.walkFN(path, d.IsDir())
		if err != nil {
			return err
		}

		if ok {
			return fn(path, d, err)
		}

		return nil
	})
}

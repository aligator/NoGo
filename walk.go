// File walk.go is based on the fs.WalkDir implementation.

// Copyright 2020 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package nogo

import (
	"errors"
	"io/fs"
	"path"
	"path/filepath"
)

type ignoreFS struct {
	fs.FS
	*NoGo
	ignoreFileName string
}

// walkDir recursively descends path, calling walkDirFn.
func walkDir(fsys *ignoreFS, name string, d fs.DirEntry, walkDirFn fs.WalkDirFunc) error {
	if d.IsDir() {
		// Add the ignore files when touching a new folder.
		// That way we do not need to read all ignore files in advance.
		err := fsys.AddFile(filepath.Join(name, fsys.ignoreFileName))
		if err != nil && !errors.Is(err, fs.ErrNotExist) {
			return err
		}
	}

	if name != "." {
		match := fsys.MatchPath(name)
		if match.OnlyFolder && !d.IsDir() {
			match.Matches = false
		}

		// If the rule is a negation rule, still proceed.
		if match.Matches && !match.Rule.Negate {
			return nil
		}
	}

	if err := walkDirFn(name, d, nil); err != nil || !d.IsDir() {
		if err == fs.SkipDir && d.IsDir() {
			// Successfully skipped directory.
			err = nil
		}
		return err
	}

	dirs, err := fs.ReadDir(fsys, name)
	if err != nil {
		// Second call, to report ReadDir error.
		err = walkDirFn(name, d, err)
		if err != nil {
			return err
		}
	}

	for _, d1 := range dirs {
		name1 := path.Join(name, d1.Name())
		if err := walkDir(fsys, name1, d1, walkDirFn); err != nil {
			if err == fs.SkipDir {
				break
			}
			return err
		}
	}
	return nil
}

// WalkDir walks the file tree rooted at root, calling fn for each file or
// directory in the tree, including root.
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
func WalkDir(fsys fs.FS, ignoreFileName string, root string, fn fs.WalkDirFunc) error {
	n := New(WithFS(fsys))
	// If .gitignore is used, assume that also the .git folder should be ignored.
	if ignoreFileName == ".gitignore" {
		WithIgnoreDotGit()(n)
	}
	ifs := &ignoreFS{
		NoGo:           n,
		FS:             fsys,
		ignoreFileName: ignoreFileName,
	}

	info, err := fs.Stat(ifs, root)
	if err != nil {
		err = fn(root, nil, err)
	} else {
		err = walkDir(ifs, root, &statDirEntry{info}, fn)
	}
	if err == fs.SkipDir {
		return nil
	}
	return err
}

type statDirEntry struct {
	info fs.FileInfo
}

func (d *statDirEntry) Name() string               { return d.info.Name() }
func (d *statDirEntry) IsDir() bool                { return d.info.IsDir() }
func (d *statDirEntry) Type() fs.FileMode          { return d.info.Mode().Type() }
func (d *statDirEntry) Info() (fs.FileInfo, error) { return d.info, nil }

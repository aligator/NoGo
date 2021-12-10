package nogo

import (
	"io/fs"
	"path/filepath"
)

func (n NoGo) skipDir(path string, d interface{ IsDir() bool }) bool {
	if path != "." {
		if d == nil {
			return false
		}

		if match, _ := n.MatchWithoutParents(path, d.IsDir()); match {
			if d.IsDir() {
				return true
			}
		}
	}

	return false
}

// WalkDirFunc can be used for fs.WalkDir.
// It only calls the passed fn for folders and directories which are not ignored.
// It assumes that NoGo is instantiated with nogo.ForFS.
func (n NoGo) WalkDirFunc(fn fs.WalkDirFunc) fs.WalkDirFunc {
	return func(path string, d fs.DirEntry, err error) error {
		skip := n.skipDir(path, d)
		if skip {
			return fs.SkipDir
		}

		return fn(path, d, err)
	}
}

// WalkFunc can be used for fs.Walk.
// It only calls the passed fn for folders and directories which are not ignored.
// It assumes that NoGo is instantiated with nogo.ForFS.
func (n NoGo) WalkFunc(fn filepath.WalkFunc) filepath.WalkFunc {
	return func(path string, info fs.FileInfo, err error) error {
		skip := n.skipDir(path, info)
		if skip {
			return fs.SkipDir
		}

		return fn(path, info, err)
	}
}

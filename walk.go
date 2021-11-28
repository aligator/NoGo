package nogo

import (
	"io/fs"
	"path/filepath"
)

func (n *NoGo) walkFN(fsys fs.FS, ignoreFileNames []string, path string, isDir bool) (bool, error) {
	if path != "." {
		if match, _ := n.MatchWithoutParents(path, isDir); match {
			if isDir {
				return false, fs.SkipDir
			}
			return false, nil
		}
	}

	if !isDir {
		name := filepath.Base(path)
		for _, ignoreFileName := range ignoreFileNames {
			if name != ignoreFileName {
				continue
			}

			// Look for new ignore files.
			err := n.AddFile(fsys, path)
			if err != nil {
				return false, err
			}
			break
		}

	}

	return true, nil
}

func (n *NoGo) ForWalkDir(fsys fs.FS, root string, ignoreFilenames []string, fn fs.WalkDirFunc) (fs.FS, string, fs.WalkDirFunc) {
	return fsys, root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		ok, err := n.walkFN(fsys, ignoreFilenames, path, d.IsDir())
		if err != nil {
			return err
		}

		if ok {
			return fn(path, d, err)
		}

		return nil
	}
}

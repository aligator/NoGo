package nogo

import (
	"io/fs"
	"path/filepath"
)

// WalkFN can be used in any Walk function.
// With it you can also write an afero compatible WalkFunc:
//  err = afero.Walk(baseFS, ".", func(path string, info fs.FileInfo, err error) error {
//		if ok, err := n.WalkFN(afero.NewIOFS(baseFS), []string{".gitignore"}, path, info.IsDir(), err); !ok {
//			return err
//		}
//
//		fmt.Println(path, info.Name())
//		return nil
//	})
func (n *NoGo) WalkFN(fsys fs.FS, ignoreFileNames []string, path string, isDir bool, err error) (bool, error) {
	if err != nil {
		return false, err
	}

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
		ok, err := n.WalkFN(fsys, ignoreFilenames, path, d.IsDir(), err)
		if err != nil {
			return err
		}

		if ok {
			return fn(path, d, err)
		}

		return nil
	}
}

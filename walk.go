package nogo

import (
	"io/fs"
)

// WalkFunc can be used in any Walk function to automatically ignore ignored files.
// It is similar to ForWalkDir but with it you can write a WalkFunc for any other (than fs.WalkDir) Walk function.
// It returns true if everything is ok and false if the path is ignored and should be skipped.
//
//
// You have to call AddFromFS with the same fs before running the walk!
//
// The Walk function you use must support the fs.SkipDir error (or you have to skip that manually)
//
// Example for afero:
//  if err := n.AddFromFS(walkFS, ".gitignore"); err != nil {
//		panic(err)
//	}
//
//  err = afero.Walk(baseFS, ".", func(path string, info fs.FileInfo, err error) error {
//		if ok, err := n.WalkFunc(afero.NewIOFS(baseFS), path, info.IsDir(), err); !ok {
//			return err
//		}
//
//		fmt.Println(path, info.Name())
//		return nil
//	})
func (n NoGo) WalkFunc(fsys fs.FS, path string, isDir bool, err error) (bool, error) {
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

	return true, nil
}

// ForWalkDir can be used to set all parameters of fs.WalkDir.
// It only calls the passed WalkDirFunc for files and directories
// which are not ignored.
//
// You have to call AddFromFS with the same fs before running the walk!
//
// If you need something similar for any other Walk function (e.g. afero.Walk)
// You can use WalkFunc for that.
//
// Example:
//  if err := n.AddFromFS(walkFS, ".gitignore"); err != nil {
//		panic(err)
//	}
//
//  n := nogo.New(nogo.DotGitRule)
//  err = fs.WalkDir(n.ForWalkDir(walkFS, ".", func(path string, d fs.DirEntry, err error) error {
//		if err != nil {
//			return err
//		}
//		fmt.Println(path, d.Name())
//		return nil
//	}))
func (n NoGo) ForWalkDir(fsys fs.FS, root string, fn fs.WalkDirFunc) (fs.FS, string, fs.WalkDirFunc) {
	return fsys, root, func(path string, d fs.DirEntry, err error) error {
		ok, err := n.WalkFunc(fsys, path, d.IsDir(), err)
		if err != nil {
			return err
		}

		if ok {
			return fn(path, d, err)
		}

		return nil
	}
}

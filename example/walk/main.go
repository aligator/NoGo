package main

import (
	"fmt"
	"github.com/aligator/nogo"
	"io/fs"
	"os"
)

func main() {
	wd, err := os.Getwd()
	if err != nil {
		panic(err)
	}

	walkFS := os.DirFS(wd)
	n := nogo.New(nogo.DotGitRule)

	// Important: The NoGo instance (n) is NOT IMMUTABLE!
	// While walking the tree it automatically loads all found .gitignore files
	// and parses the rules of it.
	//
	// -> After running the WalkDir, 'n' contains the same
	// rules as if you had used CompileAll().
	err = fs.WalkDir(n.ForWalkDir(walkFS, ".", ".gitignore", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		fmt.Println(path, d.Name())
		return nil
	}))

	if err != nil {
		panic(err)
	}
}

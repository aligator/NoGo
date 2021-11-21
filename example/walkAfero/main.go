package main

import (
	"fmt"
	"github.com/spf13/afero"
	"io/fs"
	"nogo"
	"os"
)

func main() {
	wd, err := os.Getwd()
	if err != nil {
		panic(err)
	}

	baseFS := afero.NewBasePathFs(afero.NewOsFs(), wd)
	err = nogo.AferoWalk(baseFS, ".gitignore", func(path string, info fs.FileInfo, err error) error {
		if err != nil {
			return err
		}
		fmt.Println(path, info.Name())
		return nil
	}, nogo.WithIgnoreDotGit())

	if err != nil {
		panic(err)
	}
}

package main

import (
	"fmt"
	"io/fs"
	"nogo"
	"os"

	"github.com/spf13/afero"
)

func main() {
	wd, err := os.Getwd()
	if err != nil {
		panic(err)
	}

	baseFS := afero.NewBasePathFs(afero.NewOsFs(), wd)
	err = nogo.AferoWalk([]string{".gitignore"}, baseFS, func(path string, info fs.FileInfo, err error) error {
		if err != nil {
			return err
		}
		fmt.Println(path, info.Name())
		return nil
	}, nogo.WithRules(nogo.GitIgnoreRule...))

	if err != nil {
		panic(err)
	}
}

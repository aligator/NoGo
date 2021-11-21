package main

import (
	"flag"
	"fmt"
	"nogo"
	"os"
	"strings"
)

func main() {
	flag.Parse()

	wd, err := os.Getwd()
	if err != nil {
		panic(err)
	}

	wdfs := os.DirFS(wd)

	n := nogo.New(nogo.WithIgnoreDotGit(), nogo.WithFS(wdfs))
	if err := n.AddAll(".gitignore"); err != nil {
		panic(err)
	}

	toSearch := flag.Arg(0)
	toSearch = strings.TrimPrefix(toSearch, "./")
	if toSearch == "" {
		toSearch = "."
	}

	f, err := wdfs.Open(toSearch)
	if err != nil {
		panic(err)
	}

	info, err := f.Stat()
	if err != nil {
		panic(err)
	}

	if info.Name() == ".git" {
		return
	}

	err = f.Close()
	if err != nil {
		panic(err)
	}

	if match := n.MatchPath(toSearch); match.Matches && !match.Negate && !(match.OnlyFolder && !info.IsDir()) {
		fmt.Printf("./%v\n", toSearch)
	}
}

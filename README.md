# NoGo [![test](https://github.com/aligator/nogo/actions/workflows/test.yaml/badge.svg)](https://github.com/aligator/nogo/actions/workflows/test.yaml) [![CodeQL](https://github.com/aligator/nogo/actions/workflows/codeql-analysis.yml/badge.svg)](https://github.com/aligator/nogo/actions/workflows/codeql-analysis.yml)
A .gitignore parser for Go.

## Features
* parsing .gitignore files
* loading file trees with several .gitignore files
* fs.WalkDir and afero.Walk wrappers which ignore all ignored files
* full compatibility with git
* customizable ignore filename (instead of .gitignore)

As far as I could test it, it handles .gitignore files the same way as git.  
If you find an inconsistency with git, create a new Issue.  
The goal is to provide the exact same .gitignore handling.

## Usage
By default, the working directory is used as root. You can modify that by adding `nogo.WithFS(anyFSImplementation)`

Preconfigured for standard '.gitignore' files:
```go
n := nogo.NewGitignore()
match, err := n.MatchPath(toSearch)
if err != nil {
    panic(err)
}
fmt.Println(match)
```

Plain:
```go
n := nogo.New([]string{".nogoignore"})
match, err := n.MatchPath(toSearch)
if err != nil {
    panic(err)
}
fmt.Println(match)
```

There are also some alternative Match* methods which return also the causing 
rule and/or accept a `isDir` boolean and therefore skip an extra `Stat` call which may be more performant. 

## Options
All functions (`nogo.New`, `nogo.NewGitignore`, `nogo.WalkDir`, `nogo.WalkAfero`) accept options
as the last parameters, which can further modify NoGo. Please read the GoDoc for more information.
* `WithFS` (if not used, the working directory is used)
* `WithRules` (you can also pass new rules after creation by using any of the `Add*` methods)
* `WithoutMatchParents` (more performant, but may introduce false results if used in a wrong way)

There exists a predefined rule to ignore the `.git` folder automatically: (`nogo.NewGitignore` uses it by default)
```go
n := nogo.New([]string{".nogoignore"}, WithRules(GitIgnoreRule...))
```

## Walk
NoGo provides custom walk functions which ignore all files which are ignored by any ignore file.
It exists in two flavours as I noticed that the fs.WalkDir function
is not that easy to use with afero.NewIOFS especially on windows.  
See [walk](example/walk/main.go) and [walkAfero](example/walk/main.go) for examples.

## Dependencies
NoGo is built in pure Go with the exception of
* [afero](https://github.com/spf13/afero) which is only needed by the `AferoWalk` function and to run the tests.
* [testify](https://github.com/stretchr/testify) which is only needed to run the tests.

package nogo

import (
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"testing"

	"github.com/spf13/afero"
)

func NewTestFS(t *testing.T) fs.FS {
	memfs := afero.NewMemMapFs()

	var testFS = map[string]string{
		".gitignore":                                       "globallyIgnoredFile\naPartiallyIgnoredFolder/**\n!aPartiallyIgnoredFolder/.gitignore\naFolder/ignoredFile",
		"aFile":                                            "",
		"aFolder/ignoredFile":                              "",
		"aFolder/notIgnored":                               "",
		"aFolder/locallyIgnoredFile":                       "",
		"aFolder/.gitignore":                               "/locallyIgnoredFile\n/ignoredSubFolder",
		"aFolder/ignoredSubFolder/aFile":                   "",
		"aFolder/ignoredSubFolder/anotherFile":             "",
		"aPartiallyIgnoredFolder/.gitignore":               "!unignoredFile",
		"aPartiallyIgnoredFolder/unignoredFile":            "",
		"aPartiallyIgnoredFolder/ignoredFile":              "",
		"aPartiallyIgnoredFolder/ignoredFolder/.gitignore": "notParsed as it is in an ignored folder",
	}

	for path, file := range testFS {
		folder := filepath.Dir(path)
		assert.NoError(t, memfs.MkdirAll(folder, os.ModeDir))
		f, err := memfs.Create(path)
		assert.NoError(t, err)
		_, err = f.WriteString(file)
		assert.NoError(t, err)
	}

	return afero.NewIOFS(memfs)
}

var (
	TestFSGroups = []group{
		{
			prefix: "",
			rules: []Rule{
				{
					Regexp:  regexp.MustCompile("^(.*/)?globallyIgnoredFile$"),
					Pattern: "globallyIgnoredFile",
				},
				{
					Regexp:  regexp.MustCompile("^aPartiallyIgnoredFolder/.*$"),
					Pattern: "aPartiallyIgnoredFolder/**",
				},
				{
					Regexp:  regexp.MustCompile(`^aPartiallyIgnoredFolder/\.gitignore$`),
					Pattern: "!aPartiallyIgnoredFolder/.gitignore",
					Negate:  true,
				},
				{
					Regexp:  regexp.MustCompile(`^aFolder/ignoredFile$`),
					Pattern: "aFolder/ignoredFile",
				},
			},
		},
		{
			prefix: "aFolder",
			rules: []Rule{
				{
					Regexp:  regexp.MustCompile("^aFolder/locallyIgnoredFile$"),
					Prefix:  "aFolder",
					Pattern: "/locallyIgnoredFile",
				},
				{
					Regexp:  regexp.MustCompile("^aFolder/ignoredSubFolder$"),
					Prefix:  "aFolder",
					Pattern: "/ignoredSubFolder",
				},
			},
		},
		{
			prefix: "aPartiallyIgnoredFolder",
			rules: []Rule{
				{
					Regexp:  regexp.MustCompile("^aPartiallyIgnoredFolder(/.*|.*)/unignoredFile$"),
					Prefix:  "aPartiallyIgnoredFolder",
					Pattern: "!unignoredFile",
					Negate:  true,
				},
			},
		},
	}
)

func TestCompile(t *testing.T) {
	type args struct {
		prefix  string
		pattern string
	}
	type matches struct {
		name    string
		matches bool
		input   string
	}
	tests := []struct {
		name           string
		args           args
		wantSkip       bool
		wantRegexp     string
		wantNegate     bool
		wantOnlyFolder bool
		wantErr        bool
		wantMatches    []matches
	}{
		{
			name: "a specific file in the current folder (using a prefixed '/')",
			args: args{
				prefix:  "a/folder",
				pattern: "/aFile",
			},
			wantRegexp: "^a/folder/aFile$",
			wantMatches: []matches{
				{
					name:    "the file itself",
					matches: true,
					input:   "a/folder/aFile",
				},
				{
					name:    "another file",
					matches: false,
					input:   "a/folder/anotherFile",
				},
				{
					name:    "a file in a sub folder",
					matches: false,
					input:   "a/folder/sub/aFile",
				},
				{
					name:    "the file with a suffix",
					matches: false,
					input:   "a/folder/aFile.go",
				},
			},
		},
		{
			name: "a specific file in a sub folder (using a '/' in the middle)",
			args: args{
				prefix:  "a/folder",
				pattern: "sub/aFile",
			},
			wantRegexp: "^a/folder/sub/aFile$",
			wantMatches: []matches{
				{
					name:    "the file in the root",
					matches: false,
					input:   "a/folder/aFile",
				},
				{
					name:    "another file in the same sub folder",
					matches: false,
					input:   "a/folder/sub/anotherFile",
				},
				{
					name:    "a file in a sub folder",
					matches: true,
					input:   "a/folder/sub/aFile",
				},
				{
					name:    "the file with a suffix",
					matches: false,
					input:   "a/folder/sub/aFile.go",
				},
			},
		},
		{
			name: "a specific folder in a sub folder (using a '/' in the middle and at the end)",
			args: args{
				prefix:  "a/folder",
				pattern: "sub/aFolder/",
			},
			wantOnlyFolder: true,
			wantRegexp:     "^a/folder/sub/aFolder$",
			wantMatches: []matches{
				{
					name:    "the specific folder",
					matches: true,
					input:   "a/folder/sub/aFolder",
				},
				{
					name:    "a file inside of the folder",
					matches: false,
					input:   "a/folder/sub/aFolder/aFile",
				},
			},
		},
		{
			name: "a file anywhere below",
			args: args{
				prefix:  "a/folder",
				pattern: "aFile",
			},
			wantRegexp: "^a/folder(/.*|.*)/aFile$",
			wantMatches: []matches{
				{
					name:    "the file in the root",
					matches: true,
					input:   "a/folder/aFile",
				},
				{
					name:    "the file in a sub folder",
					matches: true,
					input:   "a/folder/sub/aFile",
				},
				{
					name:    "the file in a sub folder with a postfix",
					matches: false,
					input:   "a/folder/sub/aFile.go",
				},
			},
		},
		{
			name: "a file anywhere below with empty prefix",
			args: args{
				prefix:  "",
				pattern: "aFile",
			},
			wantRegexp: "^(.*/)?aFile$",
			wantMatches: []matches{
				{
					name:    "the file in the root with slash",
					matches: true,
					input:   "/aFile",
				},
				{
					name:    "the file in the root without slash",
					matches: true,
					input:   "aFile",
				},
				{
					name:    "the file in a sub folder",
					matches: true,
					input:   "sub/aFile",
				},
				{
					name:    "the file in a sub folder with a postfix",
					matches: false,
					input:   "sub/aFile.go",
				},
			},
		},
		{
			name: "single star to allow any suffix of the file",
			args: args{
				prefix:  "a/folder",
				pattern: "/aFile.*",
			},
			wantRegexp: "^a/folder/aFile\\.[^/]*$",
			wantMatches: []matches{
				{
					name:    "a matching suffix",
					matches: true,
					input:   "a/folder/aFile.anything",
				},
				{
					name:    "a not matching filename",
					matches: false,
					input:   "a/folder/wrongFile.anything",
				},
				{
					name:    "file without the suffix",
					matches: true,
					input:   "a/folder/aFile.",
				},
			},
		},
		{
			name: "single star in the middle of a folder name",
			args: args{
				prefix:  "a/folder",
				pattern: "/aFolder*IsHere/nogo.go",
			},
			wantRegexp: "^a/folder/aFolder[^/]*IsHere/nogo\\.go$",
			wantMatches: []matches{
				{
					name:    "with something in the middle",
					matches: true,
					input:   "a/folder/aFolderWTFIsHere/nogo.go",
				},
				{
					name:    "without something in the middle",
					matches: true,
					input:   "a/folder/aFolderIsHere/nogo.go",
				},
			},
		},
		{
			name: "question mark at the end",
			args: args{
				prefix:  "a/folder",
				pattern: "/aFolder/nogo.js?",
			},
			wantRegexp: "^a/folder/aFolder/nogo\\.js[^/]?$",
			wantMatches: []matches{
				{
					name:    "with one char at the end",
					matches: true,
					input:   "a/folder/aFolder/nogo.jsx",
				},
				{
					name:    "without something at the end",
					matches: true,
					input:   "a/folder/aFolder/nogo.js",
				},
				{
					name:    "with too many chars at the end",
					matches: false,
					input:   "a/folder/aFolder/nogo.jsxy",
				},
			},
		},
		{
			name: "question mark in the middle",
			args: args{
				prefix:  "a/folder",
				pattern: "/aFolder?yay/nogo.go",
			},
			wantRegexp: "^a/folder/aFolder[^/]?yay/nogo\\.go$",
			wantMatches: []matches{
				{
					name:    "with one char",
					matches: true,
					input:   "a/folder/aFolder-yay/nogo.go",
				},
				{
					name:    "without something",
					matches: true,
					input:   "a/folder/aFolder-yay/nogo.go",
				},
				{
					name:    "with too many chars",
					matches: false,
					input:   "a/folder/aFolder--yay/nogo.go",
				},
				{
					name:    "with a slash",
					matches: false,
					input:   "a/folder/aFolder/yay/nogo.go",
				},
			},
		},
		{
			name: "several question mark in the middle",
			args: args{
				prefix:  "a/folder",
				pattern: "/aFolder???yay/nogo.go",
			},
			wantRegexp: "^a/folder/aFolder[^/]?[^/]?[^/]?yay/nogo\\.go$",
			wantMatches: []matches{
				{
					name:    "with one char",
					matches: true,
					input:   "a/folder/aFolder-yay/nogo.go",
				},
				{
					name:    "with two chars",
					matches: true,
					input:   "a/folder/aFolder--yay/nogo.go",
				},
				{
					name:    "with three chars",
					matches: true,
					input:   "a/folder/aFolderWTFyay/nogo.go",
				},
				{
					name:    "with too many chars",
					matches: false,
					input:   "a/folder/aFolderWTF-yay/nogo.go",
				},
				{
					name:    "with a slash",
					matches: false,
					input:   "a/folder/aFolder-/-yay/nogo.go",
				},
			},
		},
		{
			name: "some special regexp chars in the pattern",
			args: args{
				prefix:  "a/folder",
				pattern: "/aFol^der/n{o}go.go",
			},
			wantRegexp: "^a/folder/aFol\\^der/n\\{o\\}go\\.go$",
			wantMatches: []matches{
				{
					name:    "with these characters in the input",
					matches: true,
					input:   "a/folder/aFol^der/n{o}go.go",
				},
			},
		},
		{
			name: "escaped stars and question marks",
			args: args{
				prefix:  "a/folder",
				pattern: "/aF\\?\\?older/no\\*go.go",
			},
			wantRegexp: "^a/folder/aF\\?\\?older/no\\*go\\.go$",
			wantMatches: []matches{
				{
					name:    "with these characters in the input",
					matches: true,
					input:   "a/folder/aF??older/no*go.go",
				},
			},
		},
		{
			name: "fnmatch matching only special characters",
			args: args{
				prefix:  "a/folder",
				pattern: "/aFolder/nogo.[jt]s",
			},
			wantRegexp: "^a/folder/aFolder/nogo\\.[jt]s$",
			wantMatches: []matches{
				{
					name:    "with one of these characters",
					matches: true,
					input:   "a/folder/aFolder/nogo.js",
				},
				{
					name:    "with wrong character",
					matches: false,
					input:   "a/folder/aFolder/nogo.zs",
				},
				{
					name:    "with too less characters",
					matches: false,
					input:   "a/folder/aFolder/nogo.s",
				},
				{
					name:    "with too many of these characters",
					matches: false,
					input:   "a/folder/aFolder/nogo.jts",
				},
			},
		},
		{
			name: "fnmatch matching range special characters",
			args: args{
				prefix:  "a/folder",
				pattern: "/aFolder/nogo.[a-z]s",
			},
			wantRegexp: "^a/folder/aFolder/nogo\\.[a-z]s$",
			wantMatches: []matches{
				{
					name:    "with one of these characters",
					matches: true,
					input:   "a/folder/aFolder/nogo.as",
				},
				{
					name:    "with wrong character",
					matches: false,
					input:   "a/folder/aFolder/nogo.Ts",
				},
				{
					name:    "with too less characters",
					matches: false,
					input:   "a/folder/aFolder/nogo.s",
				},
				{
					name:    "with too many of these characters",
					matches: false,
					input:   "a/folder/aFolder/nogo.abs",
				},
			},
		},
		{
			name: "fnmatch matching negated range special characters",
			args: args{
				prefix:  "a/folder",
				pattern: "/aFolder/nogo.[!a-z]s",
			},
			wantRegexp: "^a/folder/aFolder/nogo\\.[^a-z]s$",
			wantMatches: []matches{
				{
					name:    "with one of these characters",
					matches: false,
					input:   "a/folder/aFolder/nogo.as",
				},
				{
					name:    "with another character",
					matches: true,
					input:   "a/folder/aFolder/nogo.Ts",
				},
				{
					name:    "with too less characters",
					matches: false,
					input:   "a/folder/aFolder/nogo.s",
				},
				{
					name:    "with too many of these characters",
					matches: false,
					input:   "a/folder/aFolder/nogo.ABs",
				},
			},
		},
		{
			name: "fnmatch matching escaped [ and ]",
			args: args{
				prefix:  "a/folder",
				pattern: "/aFolder/nogo.\\[!a-z\\]s",
			},
			wantRegexp: "^a/folder/aFolder/nogo\\.\\[!a-z\\]s$",
			wantMatches: []matches{
				{
					name:    "with these characters in the input",
					matches: true,
					input:   "a/folder/aFolder/nogo.[!a-z]s",
				},
			},
		},
		{
			name: "ignore empty pattern",
			args: args{
				prefix:  "a/folder",
				pattern: "",
			},
			wantRegexp: "",
			wantSkip:   true,
		},
		{
			name: "ignore with # prefix",
			args: args{
				prefix:  "a/folder",
				pattern: "# anything",
			},
			wantSkip: true,
		},
		{
			name: "do not ignore escaped #-prefix and use that # as part of the file name",
			args: args{
				prefix:  "a/folder",
				pattern: "\\#aFile",
			},
			wantRegexp: "^a/folder(/.*|.*)/#aFile$",
			wantMatches: []matches{
				{
					name:    "exact file",
					matches: true,
					input:   "a/folder/#aFile",
				},
			},
		},
		{
			name: "Strip off suffix spaces",
			args: args{
				prefix:  "a/folder",
				pattern: "aFile/isHere   ",
			},
			wantRegexp: "^a/folder/aFile/isHere$",
			wantMatches: []matches{
				{
					name:    "exact file",
					matches: true,
					input:   "a/folder/aFile/isHere",
				},
				{
					name:    "do not match the spaces",
					matches: false,
					input:   "a/folder/aFile/isHere   ",
				},
			},
		},
		{
			name: "Do not strip off suffix spaces if the last was escaped",
			args: args{
				prefix:  "a/folder",
				pattern: "aFile/isHere  \\ ",
			},
			wantRegexp: "^a/folder/aFile/isHere   $",
			wantMatches: []matches{
				{
					name:    "exact file",
					matches: true,
					input:   "a/folder/aFile/isHere   ",
				},
				{
					name:    "do not match without the spaces",
					matches: false,
					input:   "a/folder/aFile/isHere",
				},
			},
		},
		{
			name: "negate match - Note, the regexp matching is not negated, but a flag is set.",
			args: args{
				prefix:  "a/folder",
				pattern: "!/aFile",
			},
			wantRegexp: "^a/folder/aFile$",
			wantNegate: true,
			wantMatches: []matches{
				{
					name:    "the file itself",
					matches: true,
					input:   "a/folder/aFile",
				},
				{
					name:    "another file",
					matches: false,
					input:   "a/folder/anotherFile",
				},
				{
					name:    "a file in a sub folder",
					matches: false,
					input:   "a/folder/sub/aFile",
				},
				{
					name:    "the file with a suffix",
					matches: false,
					input:   "a/folder/aFile.go",
				},
			},
		},
		{
			name: "dot in prefix",
			args: args{
				prefix:  ".idea",
				pattern: "/workspace.xml",
			},
			wantRegexp: "^\\.idea/workspace\\.xml$",
			wantMatches: []matches{
				{
					name:    "the file itself",
					matches: true,
					input:   ".idea/workspace.xml",
				},
			},
		},
		{
			name: "empty prefix",
			args: args{
				prefix:  "",
				pattern: "/.git",
			},
			wantRegexp: "^\\.git$",
			wantMatches: []matches{
				{
					name:    "the file itself",
					matches: true,
					input:   ".git",
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.args.pattern+"|"+tt.name, func(t *testing.T) {
			gotSkip, gotRule, err := Compile(tt.args.prefix, tt.args.pattern)

			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}

			assert.Equal(t, tt.wantRegexp, gotRule.Regexp.String())
			assert.Equal(t, tt.wantNegate, gotRule.Negate)
			assert.Equal(t, tt.wantOnlyFolder, gotRule.OnlyFolder)
			assert.Equal(t, tt.wantSkip, gotSkip)
			if gotSkip {
				return
			}

			for _, match := range tt.wantMatches {
				t.Run(match.input+"|"+match.name, func(t *testing.T) {
					gotMatches := gotRule.Regexp.MatchString(match.input)
					assert.Equal(t, match.matches, gotMatches)
				})
			}
		})
	}
}

func TestNoGo_AddAll(t *testing.T) {
	type fields struct {
		fs              fs.FS
		groups          []group
		ignoreFileNames []string
		matchNoParents  bool
	}
	tests := []struct {
		name       string
		fields     fields
		wantErr    bool
		wantGroups []group
	}{
		{
			name: "ignore files in NewTestFS() are parsed correctly",
			fields: fields{
				fs:              NewTestFS(t),
				ignoreFileNames: []string{".gitignore"},
			},
			wantErr:    false,
			wantGroups: TestFSGroups,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			n := &NoGo{
				fs:              tt.fields.fs,
				groups:          tt.fields.groups,
				ignoreFileNames: tt.fields.ignoreFileNames,
				matchNoParents:  tt.fields.matchNoParents,
			}

			err := n.AddAll()
			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}

			assert.EqualValues(t, tt.wantGroups, n.groups)
		})
	}
}

func TestNoGo_MatchPathBecause(t *testing.T) {
	type fields struct {
		fs              fs.FS
		groups          []group
		ignoreFileNames []string
		matchNoParents  bool
	}
	defaultConfig := func() fields {
		return fields{
			fs:              NewTestFS(t),
			groups:          TestFSGroups,
			ignoreFileNames: []string{".gitignore"},
			matchNoParents:  false,
		}
	}

	tests := []struct {
		name        string
		fields      fields
		path        string
		wantMatch   bool
		wantBecause Result
		wantErr     bool
	}{
		{
			name:        "a not ignored aFile",
			fields:      defaultConfig(),
			path:        "aFile",
			wantMatch:   false,
			wantBecause: Result{},
			wantErr:     false,
		},
		{
			name:      "an ignored aFile",
			fields:    defaultConfig(),
			path:      "aFolder/ignoredFile",
			wantMatch: true,
			wantBecause: Result{
				Rule: Rule{
					Regexp:  regexp.MustCompile("^aFolder/ignoredFile$"),
					Pattern: "aFolder/ignoredFile",
				},
				Found:       true,
				ParentMatch: false,
			},
			wantErr: false,
		},
		{
			name:      "an unignored aFile",
			fields:    defaultConfig(),
			path:      "aPartiallyIgnoredFolder/unignoredFile",
			wantMatch: false,
			wantBecause: Result{
				Rule: Rule{
					Regexp:  regexp.MustCompile("^aPartiallyIgnoredFolder(/.*|.*)/unignoredFile$"),
					Prefix:  "aPartiallyIgnoredFolder",
					Pattern: "!unignoredFile",
					Negate:  true,
				},
				Found:       true,
				ParentMatch: false,
			},
			wantErr: false,
		},
		// TODO: more tests
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			n := &NoGo{
				fs:              tt.fields.fs,
				groups:          tt.fields.groups,
				ignoreFileNames: tt.fields.ignoreFileNames,
				matchNoParents:  tt.fields.matchNoParents,
			}
			gotMatch, gotBecause, err := n.MatchPathBecause(tt.path)
			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}

			assert.Equal(t, tt.wantMatch, gotMatch)
			assert.EqualValues(t, tt.wantBecause, gotBecause)
		})
	}
}

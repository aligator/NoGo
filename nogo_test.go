package nogo

import (
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"io/fs"
	"regexp"
	"testing"
	"testing/fstest"
)

var (
	TestFSGroups = []group{
		{
			prefix: "",
			rules: []Rule{
				{
					Regexp:  []*regexp.Regexp{regexp.MustCompile("^(.*/)?globallyIgnored$")},
					Pattern: "globallyIgnored",
				},
				{
					Regexp:  []*regexp.Regexp{regexp.MustCompile("^aPartiallyIgnoredFolder/.*$")},
					Pattern: "aPartiallyIgnoredFolder/**",
				},
				{
					Regexp:  []*regexp.Regexp{regexp.MustCompile(`^aPartiallyIgnoredFolder/\.gitignore$`)},
					Pattern: "!aPartiallyIgnoredFolder/.gitignore",
					Negate:  true,
				},
				{
					Regexp:  []*regexp.Regexp{regexp.MustCompile(`^aFolder/ignoredFile$`)},
					Pattern: "aFolder/ignoredFile",
				},
				{
					Regexp:     []*regexp.Regexp{regexp.MustCompile(`^(.*/)?ignoredFolder$`)},
					Pattern:    "ignoredFolder/",
					OnlyFolder: true,
				},
				{
					Regexp:     []*regexp.Regexp{regexp.MustCompile(`^(.*/)?ignoredFolder-notAFolder$`)},
					Pattern:    "ignoredFolder-notAFolder/",
					OnlyFolder: true,
				},
			},
		},
		{
			prefix: "aFolder",
			rules: []Rule{
				{
					Regexp:  []*regexp.Regexp{regexp.MustCompile("^aFolder/locallyIgnoredFile$")},
					Prefix:  "aFolder",
					Pattern: "/locallyIgnoredFile",
				},
				{
					Regexp:  []*regexp.Regexp{regexp.MustCompile("^aFolder/ignoredSubFolder$")},
					Prefix:  "aFolder",
					Pattern: "/ignoredSubFolder",
				},
			},
		},
		{
			prefix: "aPartiallyIgnoredFolder",
			rules: []Rule{
				{
					Regexp:  []*regexp.Regexp{regexp.MustCompile("^aPartiallyIgnoredFolder(/.*)?/unignoredFile$")},
					Prefix:  "aPartiallyIgnoredFolder",
					Pattern: "!unignoredFile",
					Negate:  true,
				},
			},
		},
		{
			prefix: "glob-tests",
			rules: []Rule{
				{
					Regexp:  []*regexp.Regexp{regexp.MustCompile("^glob-tests/file[^/]*withStar$")},
					Prefix:  "glob-tests",
					Pattern: "/file*withStar",
				},
				{
					Regexp:  []*regexp.Regexp{regexp.MustCompile("^glob-tests/question[^/]?mark[^/]?[^/]?file[^/]?[^/]?[^/]?$")},
					Prefix:  "glob-tests",
					Pattern: "/question?mark??file???",
				},
				{
					Regexp: []*regexp.Regexp{
						regexp.MustCompile("^glob-tests/file[^/]with[^/]ranges$"),
						regexp.MustCompile("^glob-tests/file[a-z]with[^0-9]ranges$"),
					},
					Prefix:  "glob-tests",
					Pattern: "/file[a-z]with[!0-9]ranges",
				},
				{
					Regexp:  []*regexp.Regexp{regexp.MustCompile("^glob-tests/file[^/]*withDoubleStar$")},
					Prefix:  "glob-tests",
					Pattern: "/file**withDoubleStar", // Actually this resolves to a single star as the double star only has special meaning at the beginning or end of a filename.
				},
				{
					Regexp:  []*regexp.Regexp{regexp.MustCompile("^glob-tests(/.*)?/foo$")},
					Prefix:  "glob-tests",
					Pattern: "**/foo",
				},
				{
					Regexp:  []*regexp.Regexp{regexp.MustCompile("^glob-tests/any/.*$")},
					Prefix:  "glob-tests",
					Pattern: "any/**",
				},
				{
					Regexp:  []*regexp.Regexp{regexp.MustCompile("^glob-tests/something.*/more$")},
					Prefix:  "glob-tests",
					Pattern: "something/**/more",
				},
			},
		},
	}
)

var TestFSData = map[string]struct {
	data      string
	ignoredBy *Result
	isDir     bool
}{
	".gitignore":               {"globallyIgnored\naPartiallyIgnoredFolder/**\n!aPartiallyIgnoredFolder/.gitignore\naFolder/ignoredFile\nignoredFolder/\nignoredFolder-notAFolder/", nil, false},
	"ignoredFolder":            {"", &Result{Rule: TestFSGroups[0].rules[4], Found: true, ParentMatch: false}, true},
	"ignoredFolder-notAFolder": {"", nil, false},
	"globallyIgnored":          {"", &Result{Rule: TestFSGroups[0].rules[0], Found: true, ParentMatch: false}, false},
	"aFile":                    {"", nil, false},
	"aFolder/ignoredFile":      {"", &Result{Rule: TestFSGroups[0].rules[3], Found: true, ParentMatch: false}, false},
	// aFolder/ignoredFolder is actually no folder -> not ignored
	"aFolder/ignoredFolder":                                        {"", nil, false},
	"aFolder/notIgnored":                                           {"", nil, false},
	"aFolder/locallyIgnoredFile":                                   {"", &Result{Rule: TestFSGroups[1].rules[0], Found: true, ParentMatch: false}, false},
	"aFolder/.gitignore":                                           {"/locallyIgnoredFile\n/ignoredSubFolder", nil, false},
	"aFolder/ignoredSubFolder/aFile":                               {"", &Result{Rule: TestFSGroups[1].rules[1], Found: true, ParentMatch: true}, false},
	"aFolder/ignoredSubFolder/anotherFile":                         {"", &Result{Rule: TestFSGroups[1].rules[1], Found: true, ParentMatch: true}, false},
	"aPartiallyIgnoredFolder/.gitignore":                           {"!unignoredFile", &Result{Rule: TestFSGroups[0].rules[2], Found: true, ParentMatch: false}, false},
	"aPartiallyIgnoredFolder/unignoredFile":                        {"", &Result{Rule: TestFSGroups[2].rules[0], Found: true, ParentMatch: false}, false},
	"aPartiallyIgnoredFolder/ignoredFile":                          {"", &Result{Rule: TestFSGroups[0].rules[1], Found: true, ParentMatch: false}, false},
	"aPartiallyIgnoredFolder/ignoredFolder/.gitignore":             {"notParsed as it is in an ignored folder", &Result{Rule: TestFSGroups[0].rules[1], Found: true, ParentMatch: false}, false},
	"aFolder/anotherFolder/globallyIgnored":                        {"", &Result{Rule: TestFSGroups[0].rules[0], Found: true, ParentMatch: false}, false},
	"aFolder/anotherFolder/globallyIgnored/aFileInGloballyIgnored": {"", &Result{Rule: TestFSGroups[0].rules[0], Found: true, ParentMatch: true}, false},

	"glob-tests/.gitignore": {"/file*withStar\n/question?mark??file???\n/file[a-z]with[!0-9]ranges\n/file**withDoubleStar\n**/foo\nany/**\nsomething/**/more", nil, false},
	// star
	"glob-tests/file42withStar":  {"", &Result{Rule: TestFSGroups[3].rules[0], Found: true, ParentMatch: false}, false},
	"glob-tests/filewithStar":    {"", &Result{Rule: TestFSGroups[3].rules[0], Found: true, ParentMatch: false}, false},
	"glob-tests/file4/2withStar": {"", nil, false},

	// question mark
	"glob-tests/questionmarkfile":       {"", &Result{Rule: TestFSGroups[3].rules[1], Found: true, ParentMatch: false}, false},
	"glob-tests/question0mark42file123": {"", &Result{Rule: TestFSGroups[3].rules[1], Found: true, ParentMatch: false}, false},
	"glob-tests/questionämarköfileü":    {"", &Result{Rule: TestFSGroups[3].rules[1], Found: true, ParentMatch: false}, false},
	"glob-tests/question/markfile":      {"", nil, false},

	// ranges
	"glob-tests/filefwith-ranges": {"", &Result{Rule: TestFSGroups[3].rules[2], Found: true, ParentMatch: false}, false},
	"glob-tests/filewithranges":   {"", nil, false},
	"glob-tests/fileAwithAranges": {"", nil, false},
	"glob-tests/fileawith5ranges": {"", nil, false},
	"glob-tests/filefwith/ranges": {"", nil, false},

	// double star  // Actually this resolves to a single star as the double star only has special meaning at the beginning or end of a filename.
	"glob-tests/file42withDoubleStar":  {"", &Result{Rule: TestFSGroups[3].rules[3], Found: true, ParentMatch: false}, false},
	"glob-tests/filewithDoubleStar":    {"", &Result{Rule: TestFSGroups[3].rules[3], Found: true, ParentMatch: false}, false},
	"glob-tests/file4/2withDoubleStar": {"", nil, false},

	// **/foo
	"glob-tests/foo":      {"", &Result{Rule: TestFSGroups[3].rules[4], Found: true, ParentMatch: false}, false},
	"glob-tests/bar/foo":  {"", &Result{Rule: TestFSGroups[3].rules[4], Found: true, ParentMatch: false}, false},
	"glob-tests/bar/ffoo": {"", nil, false},
	"glob-tests/barfoo":   {"", nil, false},
	"glob-tests/foo/bar":  {"", &Result{Rule: TestFSGroups[3].rules[4], Found: true, ParentMatch: true}, false},

	// any/**
	"glob-tests/any":         {"", nil, false},
	"glob-tests/any/foo/bar": {"", &Result{Rule: TestFSGroups[3].rules[5], Found: true, ParentMatch: false}, false},
	"glob-tests/any/foo":     {"", &Result{Rule: TestFSGroups[3].rules[5], Found: true, ParentMatch: false}, false},
	"glob-tests/anyfoo/bar":  {"", nil, false},

	// something/**/more
	"glob-tests/something/more":                     {"", &Result{Rule: TestFSGroups[3].rules[6], Found: true, ParentMatch: false}, false},
	"glob-tests/something/much/much/more":           {"", &Result{Rule: TestFSGroups[3].rules[6], Found: true, ParentMatch: false}, false},
	"glob-tests/something/much/much/more/andMOOORE": {"", &Result{Rule: TestFSGroups[3].rules[6], Found: true, ParentMatch: true}, false},
	"glob-tests/something":                          {"", nil, false},
	"glob-tests/somethingmore":                      {"", nil, false},
}

func NewTestFS() fs.FS {
	memfs := fstest.MapFS{}

	for path, file := range TestFSData {
		if file.isDir {
			memfs[path] = &fstest.MapFile{Mode: fs.ModeDir}
		} else {
			memfs[path] = &fstest.MapFile{
				Data: []byte(file.data),
			}
		}
	}

	return memfs
}

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
		wantRegexp     []string
		wantNegate     bool
		wantOnlyFolder bool
		wantErr        assert.ErrorAssertionFunc
		wantMatches    []matches
	}{
		{
			name: "a specific file in the current folder (using a prefixed '/')",
			args: args{
				prefix:  "a/folder",
				pattern: "/aFile",
			},
			wantRegexp: []string{"^a/folder/aFile$"},
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
			wantErr: assert.NoError,
		},
		{
			name: "a specific file in a sub folder (using a '/' in the middle)",
			args: args{
				prefix:  "a/folder",
				pattern: "sub/aFile",
			},
			wantRegexp: []string{"^a/folder/sub/aFile$"},
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
			wantErr: assert.NoError,
		},
		{
			name: "a specific folder in a sub folder (using a '/' in the middle and at the end)",
			args: args{
				prefix:  "a/folder",
				pattern: "sub/aFolder/",
			},
			wantOnlyFolder: true,
			wantRegexp:     []string{"^a/folder/sub/aFolder$"},
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
			wantErr: assert.NoError,
		},
		{
			name: "a file anywhere below",
			args: args{
				prefix:  "a/folder",
				pattern: "aFile",
			},
			wantRegexp: []string{"^a/folder(/.*)?/aFile$"},
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
			wantErr: assert.NoError,
		},
		{
			name: "a file anywhere below with empty prefix",
			args: args{
				prefix:  "",
				pattern: "aFile",
			},
			wantRegexp: []string{"^(.*/)?aFile$"},
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
			wantErr: assert.NoError,
		},
		{
			name: "single star to allow any suffix of the file",
			args: args{
				prefix:  "a/folder",
				pattern: "/aFile.*",
			},
			wantRegexp: []string{"^a/folder/aFile\\.[^/]*$"},
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
			wantErr: assert.NoError,
		},
		{
			name: "single star in the middle of a folder name",
			args: args{
				prefix:  "a/folder",
				pattern: "/aFolder*IsHere/nogo.go",
			},
			wantRegexp: []string{"^a/folder/aFolder[^/]*IsHere/nogo\\.go$"},
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
			wantErr: assert.NoError,
		},
		{
			name: "question mark at the end",
			args: args{
				prefix:  "a/folder",
				pattern: "/aFolder/nogo.js?",
			},
			wantRegexp: []string{"^a/folder/aFolder/nogo\\.js[^/]?$"},
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
			wantErr: assert.NoError,
		},
		{
			name: "question mark in the middle",
			args: args{
				prefix:  "a/folder",
				pattern: "/aFolder?yay/nogo.go",
			},
			wantRegexp: []string{"^a/folder/aFolder[^/]?yay/nogo\\.go$"},
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
			wantErr: assert.NoError,
		},
		{
			name: "several question mark in the middle",
			args: args{
				prefix:  "a/folder",
				pattern: "/aFolder???yay/nogo.go",
			},
			wantRegexp: []string{"^a/folder/aFolder[^/]?[^/]?[^/]?yay/nogo\\.go$"},
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
			wantErr: assert.NoError,
		},
		{
			name: "some special regexp chars in the pattern",
			args: args{
				prefix:  "a/folder",
				pattern: "/aFol^der/n{o}go.go",
			},
			wantRegexp: []string{"^a/folder/aFol\\^der/n\\{o\\}go\\.go$"},
			wantMatches: []matches{
				{
					name:    "with these characters in the input",
					matches: true,
					input:   "a/folder/aFol^der/n{o}go.go",
				},
			},
			wantErr: assert.NoError,
		},
		{
			name: "escaped stars and question marks",
			args: args{
				prefix:  "a/folder",
				pattern: "/aF\\?\\?older/no\\*go.go",
			},
			wantRegexp: []string{"^a/folder/aF\\?\\?older/no\\*go\\.go$"},
			wantMatches: []matches{
				{
					name:    "with these characters in the input",
					matches: true,
					input:   "a/folder/aF??older/no*go.go",
				},
			},
			wantErr: assert.NoError,
		},
		{
			name: "fnmatch matching only special characters",
			args: args{
				prefix:  "a/folder",
				pattern: "/aFolder/nogo.[jt]s",
			},
			wantRegexp: []string{"^a/folder/aFolder/nogo\\.[^/]s$", "^a/folder/aFolder/nogo\\.[jt]s$"},
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
			wantErr: assert.NoError,
		},
		{
			name: "fnmatch matching range special characters",
			args: args{
				prefix:  "a/folder",
				pattern: "/aFolder/nogo.[a-z]s",
			},
			wantRegexp: []string{"^a/folder/aFolder/nogo\\.[^/]s$", "^a/folder/aFolder/nogo\\.[a-z]s$"},
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
			wantErr: assert.NoError,
		},
		{
			name: "fnmatch matching negated range special characters",
			args: args{
				prefix:  "a/folder",
				pattern: "/aFolder/nogo.[!a-z]s",
			},
			wantRegexp: []string{"^a/folder/aFolder/nogo\\.[^/]s$", "^a/folder/aFolder/nogo\\.[^a-z]s$"},
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
			wantErr: assert.NoError,
		},
		{
			name: "fnmatch matching escaped [ and ]",
			args: args{
				prefix:  "a/folder",
				pattern: `/aFolder/nogo.\[!a-z\]s`,
			},
			wantRegexp: []string{`^a/folder/aFolder/nogo\.\[!a-z\]s$`},
			wantMatches: []matches{
				{
					name:    "with these characters in the input",
					matches: true,
					input:   "a/folder/aFolder/nogo.[!a-z]s",
				},
			},
			wantErr: assert.NoError,
		},
		{
			name: "ignore empty pattern",
			args: args{
				prefix:  "a/folder",
				pattern: "",
			},
			wantRegexp: nil,
			wantSkip:   true,
			wantErr:    assert.NoError,
		},
		{
			name: "ignore with # prefix",
			args: args{
				prefix:  "a/folder",
				pattern: "# anything",
			},
			wantSkip: true,
			wantErr:  assert.NoError,
		},
		{
			name: "do not ignore escaped #-prefix and use that # as part of the file name",
			args: args{
				prefix:  "a/folder",
				pattern: "\\#aFile",
			},
			wantRegexp: []string{"^a/folder(/.*)?/#aFile$"},
			wantMatches: []matches{
				{
					name:    "exact file",
					matches: true,
					input:   "a/folder/#aFile",
				},
			},
			wantErr: assert.NoError,
		},
		{
			name: "Strip off suffix spaces",
			args: args{
				prefix:  "a/folder",
				pattern: "aFile/isHere   ",
			},
			wantRegexp: []string{"^a/folder/aFile/isHere$"},
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
			wantErr: assert.NoError,
		},
		{
			name: "Do not strip off suffix spaces if the last was escaped",
			args: args{
				prefix:  "a/folder",
				pattern: "aFile/isHere  \\ ",
			},
			wantRegexp: []string{"^a/folder/aFile/isHere   $"},
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
			wantErr: assert.NoError,
		},
		{
			name: "negate match - Note, the regexp matching is not negated, but a flag is set.",
			args: args{
				prefix:  "a/folder",
				pattern: "!/aFile",
			},
			wantRegexp: []string{"^a/folder/aFile$"},
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
			wantErr: assert.NoError,
		},
		{
			name: "dot in prefix",
			args: args{
				prefix:  ".idea",
				pattern: "/workspace.xml",
			},
			wantRegexp: []string{"^\\.idea/workspace\\.xml$"},
			wantMatches: []matches{
				{
					name:    "the file itself",
					matches: true,
					input:   ".idea/workspace.xml",
				},
			},
			wantErr: assert.NoError,
		},
		{
			name: "empty prefix",
			args: args{
				prefix:  "",
				pattern: "/.git",
			},
			wantRegexp: []string{"^\\.git$"},
			wantMatches: []matches{
				{
					name:    "the file itself",
					matches: true,
					input:   ".git",
				},
			},
			wantErr: assert.NoError,
		},
		{
			name: "open [ in pattern",
			args: args{
				prefix:  "",
				pattern: "[lool",
			},
			wantErr: assert.Error,
		},
		{
			name: "escaped open [ in pattern",
			args: args{
				prefix:  "",
				pattern: `\[lool`,
			},
			wantRegexp: []string{`^(.*/)?\[lool$`},
			wantErr:    assert.NoError,
		},
	}
	for _, tt := range tests {
		t.Run(tt.args.pattern+"|"+tt.name, func(t *testing.T) {
			gotSkip, gotRule, err := Compile(tt.args.prefix, tt.args.pattern)

			if tt.wantErr(t, err) {
				return
			}

			require.Equal(t, len(tt.wantRegexp), len(gotRule.Regexp))

			for i := range tt.wantRegexp {
				assert.EqualValues(t, tt.wantRegexp[i], gotRule.Regexp[i].String())
			}

			assert.Equal(t, tt.wantNegate, gotRule.Negate)
			assert.Equal(t, tt.wantOnlyFolder, gotRule.OnlyFolder)
			assert.Equal(t, tt.wantSkip, gotSkip)
			if gotSkip {
				return
			}

			for _, match := range tt.wantMatches {
				t.Run(match.input+"|"+match.name, func(t *testing.T) {
					var gotMatches bool
					for _, reg := range gotRule.Regexp {
						gotMatches = reg.MatchString(match.input)
						if !gotMatches {
							break
						}
					}
					assert.Equal(t, match.matches, gotMatches)
				})
			}
		})
	}
}

func TestNoGo_AddAll(t *testing.T) {
	type fields struct {
		fs             fs.FS
		groups         []group
		ignoreFileName string
		matchNoParents bool
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
				fs:             NewTestFS(),
				ignoreFileName: ".gitignore",
			},
			wantErr:    false,
			wantGroups: TestFSGroups,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			n := &NoGo{
				groups: tt.fields.groups,
			}
			err := n.addFromFS(tt.fields.fs, tt.fields.ignoreFileName)

			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}

			assert.EqualValues(t, tt.wantGroups, n.groups)
		})
	}
}

func TestNoGo_MatchBecause(t *testing.T) {
	for path, tt := range TestFSData {
		t.Run(path, func(t *testing.T) {
			n := &NoGo{
				groups: TestFSGroups,
			}
			gotMatch, gotBecause := n.MatchBecause(path, tt.isDir)

			if gotBecause.Negate {
				assert.Equal(t, tt.ignoredBy == nil, gotMatch)
			} else {
				assert.Equal(t, tt.ignoredBy != nil, gotMatch)
			}

			if tt.ignoredBy != nil {
				assert.EqualValues(t, *tt.ignoredBy, gotBecause)
			}
		})
	}
}

func TestNoGo_MatchWithoutParents(t *testing.T) {
	for path, tt := range TestFSData {
		t.Run(path, func(t *testing.T) {
			n := &NoGo{
				groups: TestFSGroups,
			}
			gotMatch, gotBecause := n.MatchWithoutParents(path, tt.isDir)

			// The same test as TestNoGo_MatchBecause, but ignore all parent matches.
			if tt.ignoredBy != nil && tt.ignoredBy.ParentMatch {
				tt.ignoredBy = nil
			}

			if gotBecause.Negate {
				assert.Equal(t, tt.ignoredBy == nil, gotMatch)
			} else {
				assert.Equal(t, tt.ignoredBy != nil, gotMatch)
			}

			if tt.ignoredBy != nil {
				assert.EqualValues(t, *tt.ignoredBy, gotBecause)
			}
		})
	}

	t.Run("file which would be matched by a parent with onlyFolder", func(t *testing.T) {
		n := &NoGo{
			groups: []group{
				{
					prefix: "",
					rules: []Rule{
						{
							Regexp: []*regexp.Regexp{regexp.MustCompile(`^(.*/)?anIgnoredFolder$`)},
							Prefix: "",
						},
					},
				},
				{
					prefix: "anIgnoredFolder",
					rules: []Rule{
						{
							Regexp: []*regexp.Regexp{regexp.MustCompile(`^anIgnoredFolder/aFile$`)},
							Prefix: "anIgnoredFolder",
							// Even if it is a file in this example, apply the rule only on folders:
							OnlyFolder: true,
						},
						{
							Regexp: []*regexp.Regexp{regexp.MustCompile(`^anIgnoredFolder/anotherFile$`)},
							Prefix: "anIgnoredFolder",
						},
					},
				},
			},
		}

		// Should be matched by the normal match: (as the parent folder is not checked for OnlyFolder):
		gotMatch, gotBecause := n.MatchBecause("anIgnoredFolder/aFile", false)
		assert.True(t, gotMatch)
		assert.True(t, gotBecause.Resolve(false))
		assert.EqualValues(t, Result{
			Rule:        n.groups[0].rules[0],
			Found:       true,
			ParentMatch: true,
		}, gotBecause)

		// But it should not be matched by MatchWithoutParents: (as the parent folder is never
		// checked and therefore the file is not ignored for being inside an ignored folder)
		// Also as it is a file it doesn't match the OnlyFolder rule:
		gotMatch, gotBecause = n.MatchWithoutParents("anIgnoredFolder/aFile", false)
		assert.False(t, gotMatch)
		assert.False(t, gotBecause.Resolve(false))
		assert.EqualValues(t, Result{}, gotBecause)

		// Should be matched by the normal match:
		gotMatch, gotBecause = n.MatchBecause("anIgnoredFolder/anotherFile", false)
		assert.True(t, gotMatch)
		assert.True(t, gotBecause.Resolve(false))
		assert.EqualValues(t, Result{
			Rule:        n.groups[1].rules[1],
			Found:       true,
			ParentMatch: false,
		}, gotBecause)

		// And it should also match with MatchWithoutParents as the file is matched inside the folder directly:
		gotMatch, gotBecause = n.MatchWithoutParents("anIgnoredFolder/anotherFile", false)
		assert.True(t, gotMatch)
		assert.True(t, gotBecause.Resolve(false))
		assert.EqualValues(t, Result{
			Rule:        n.groups[1].rules[1],
			Found:       true,
			ParentMatch: false,
		}, gotBecause)
	})
}

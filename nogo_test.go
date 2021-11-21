package nogo

import (
	"testing"
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
			wantRegexp: "^a/folder/aFile/?$",
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
			wantRegexp: "^a/folder/sub/aFile/?$",
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
			wantRegexp:     "^a/folder/sub/aFolder/$",
			wantMatches: []matches{
				{
					name:    "the specific folder",
					matches: true,
					input:   "a/folder/sub/aFolder/",
				},
				{
					name:    "a file with the same name (currently it is expected that the input ends with a / if it is a folder)",
					matches: false,
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
			wantRegexp: "^a/folder.*/aFile/?$",
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
				{
					name:    "a folder with the same name",
					matches: true,
					input:   "a/folder/sub/aFile/",
				},
			},
		},
		{
			name: "single star to allow any suffix of the file",
			args: args{
				prefix:  "a/folder",
				pattern: "/aFile.*",
			},
			wantRegexp: "^a/folder/aFile\\.[^/]*/?$",
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
			wantRegexp: "^a/folder/aFolder[^/]*IsHere/nogo\\.go/?$",
		},
		{
			name: "question mark at the end",
			args: args{
				prefix:  "a/folder",
				pattern: "/aFolder/nogo.js?",
			},
			wantRegexp: "^a/folder/aFolder/nogo\\.js[^/]?/?$",
		},
		{
			name: "question mark in the middle",
			args: args{
				prefix:  "a/folder",
				pattern: "/aFolder?yay/nogo.go",
			},
			wantRegexp: "^a/folder/aFolder[^/]?yay/nogo\\.go/?$",
		},
		{
			name: "several question mark in the middle",
			args: args{
				prefix:  "a/folder",
				pattern: "/aFolder???yay/nogo.go",
			},
			wantRegexp: "^a/folder/aFolder[^/]?[^/]?[^/]?yay/nogo\\.go/?$",
		},
		{
			name: "some special regexp chars in the pattern",
			args: args{
				prefix:  "a/folder",
				pattern: "/aFolder/n{o}go.go",
			},
			wantRegexp: "^a/folder/aFolder/n\\{o\\}go\\.go/?$",
		},
		{
			name: "escaped stars and question marks",
			args: args{
				prefix:  "a/folder",
				pattern: "/aF\\?\\?older/no\\*go.go",
			},
			wantRegexp: "^a/folder/aF\\?\\?older/no\\*go\\.go/?$",
		},
		{
			name: "fnmatch matching only special characters",
			args: args{
				prefix:  "a/folder",
				pattern: "/aFolder/nogo.[jt]s",
			},
			wantRegexp: "^a/folder/aFolder/nogo\\.[jt]s/?$",
		},
		{
			name: "fnmatch matching range special characters",
			args: args{
				prefix:  "a/folder",
				pattern: "/aFolder/nogo.[j-t]s",
			},
			wantRegexp: "^a/folder/aFolder/nogo\\.[j-t]s/?$",
		},
		{
			name: "fnmatch matching negated range special characters",
			args: args{
				prefix:  "a/folder",
				pattern: "/aFolder/nogo.[!j-t]s",
			},
			wantRegexp: "^a/folder/aFolder/nogo\\.[^j-t]s/?$",
		},
		{
			name: "fnmatch matching escaped [ and ]",
			args: args{
				prefix:  "a/folder",
				pattern: "/aFolder/nogo.\\[!j-t\\]s",
			},
			wantRegexp: "^a/folder/aFolder/nogo\\.\\[!j-t\\]s/?$",
		},
	}
	for _, tt := range tests {
		t.Run(tt.args.pattern+"|"+tt.name, func(t *testing.T) {
			gotSkip, gotRule, err := Compile(tt.args.prefix, tt.args.pattern)
			if (err != nil) != tt.wantErr {
				t.Errorf("Compile() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if gotRule.Regexp.String() != tt.wantRegexp {
				t.Errorf("Compile() gotRule.Regexp.String() = %v, want %v", gotRule.Regexp.String(), tt.wantRegexp)
			}
			if gotRule.Negate != tt.wantNegate {
				t.Errorf("Compile() gotRule.Negate = %v, want %v", gotRule.Negate, tt.wantNegate)
			}
			if gotRule.OnlyFolder != tt.wantOnlyFolder {
				t.Errorf("Compile() gotRule.OnlyFolder = %v, want %v", gotRule.OnlyFolder, tt.wantOnlyFolder)
			}

			if gotSkip != tt.wantSkip {
				t.Errorf("Compile() gotSkip = %v, want %v", gotSkip, tt.wantSkip)
				return
			}

			for _, match := range tt.wantMatches {
				t.Run(match.input+"|"+match.name, func(t *testing.T) {
					gotMatches := gotRule.MatchString(match.input)
					if gotMatches != match.matches {
						t.Errorf("gotRule.MatchString(\"%v\") = %v, want %v", match.input, gotMatches, match.matches)
					}
				})
			}
		})
	}
}

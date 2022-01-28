package nogo

import (
	"errors"
	"fmt"
	"io/fs"
	"path/filepath"
	"regexp"
	"testing"
	"testing/fstest"

	"github.com/stretchr/testify/assert"
)

func TestNoGo_WalkFunc(t *testing.T) {
	type fields struct {
		groups []group
	}
	type args struct {
		fsys           fs.FS
		ignoreFileName string
		path           string
		isDir          bool
		err            error
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    bool
		wantErr assert.ErrorAssertionFunc
	}{
		{
			name: "not ignored file",
			fields: fields{
				groups: TestFSGroups,
			},
			args: args{
				fsys:           NewTestFS(),
				ignoreFileName: ".gitignore",
				path:           "aFile",
			},
			want:    true,
			wantErr: assert.NoError,
		},
		{
			name: "error is set",
			args: args{
				fsys:           NewTestFS(),
				ignoreFileName: ".gitignore",
				path:           "aFile",
				err:            errors.New("an error"),
			},
			want:    false,
			wantErr: assert.Error,
		},
		{
			name: "ignored folder",
			fields: fields{
				groups: TestFSGroups,
			},
			args: args{
				fsys:           NewTestFS(),
				ignoreFileName: ".gitignore",
				path:           "ignoredFolder",
				isDir:          true,
			},
			want: false,
			wantErr: func(t assert.TestingT, err error, i ...interface{}) bool {
				return assert.ErrorIs(t, err, fs.SkipDir, i...)
			},
		},
		{
			name: "ignore file should be read if folder gets loaded",
			fields: fields{
				groups: nil,
			},
			args: args{
				fsys:           NewTestFS(),
				ignoreFileName: ".gitignore",
				path:           "",
				isDir:          true,
			},
			want:    true,
			wantErr: assert.NoError,
		},
		{
			name: "ignore file should be ignored if it is already ignored by a previous ignore file",
			fields: fields{
				groups: []group{
					{
						prefix: "",
						rules: []Rule{
							{
								Regexp: []*regexp.Regexp{regexp.MustCompile(`\.gitignore`)},
							},
						},
					},
				},
			},
			args: args{
				fsys:           NewTestFS(),
				ignoreFileName: ".gitignore",
				path:           "",
				isDir:          true,
			},
			// But still return ok as the folder itself is not ignored.
			want:    true,
			wantErr: assert.NoError,
		},
		{
			name: "ignore file should be ignored if it is already ignored by a previous ignore file",
			fields: fields{
				groups: []group{
					{
						prefix: "",
						rules: []Rule{
							{
								Regexp: []*regexp.Regexp{regexp.MustCompile(`\.gitignore`)},
							},
						},
					},
				},
			},
			args: args{
				fsys:           NewTestFS(),
				ignoreFileName: ".gitignore",
				path:           "",
				isDir:          true,
			},
			// But still return ok as the folder itself is not ignored.
			want:    true,
			wantErr: assert.NoError,
		},
		{
			name: "ignore file which doesn't exist should be ignored",
			fields: fields{
				groups: []group{
					{
						prefix: "",
						rules: []Rule{
							{
								Regexp: []*regexp.Regexp{regexp.MustCompile(`\.gitignore`)},
							},
						},
					},
				},
			},
			args: args{
				fsys:           NewTestFS(),
				ignoreFileName: "noIgnoreFile",
				path:           "",
				isDir:          true,
			},
			// But still return ok as the folder itself is not ignored.
			want:    true,
			wantErr: assert.NoError,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			n := &NoGo{
				groups: tt.fields.groups,
			}

			assert.NoError(t, n.AddFromFS(tt.args.fsys, tt.args.ignoreFileName))

			got, err := n.WalkFunc(tt.args.fsys, tt.args.path, tt.args.isDir, tt.args.err)
			if !tt.wantErr(t, err, fmt.Sprintf("WalkFunc(%v, %v, %v, %v, %v)", tt.args.fsys, tt.args.ignoreFileName, tt.args.path, tt.args.isDir, tt.args.err)) {
				return
			}
			assert.Equalf(t, tt.want, got, "WalkFunc(%v, %v, %v, %v, %v)", tt.args.fsys, tt.args.ignoreFileName, tt.args.path, tt.args.isDir, tt.args.err)
		})
	}
}

var ErrShouldNotBeReached = errors.New("file should not be reached")

// ForbiddenFS is a fstest.MapFS but allows to define
// files which should not be loaded.
// Note: Define folders explicitely in the map if you want to forbid them.
type ForbiddenFS struct {
	fstest.MapFS
	NotExpected map[string]struct{}
}

func (ofs ForbiddenFS) Open(name string) (fs.File, error) {
	file, err := ofs.MapFS.Open(name)
	if err != nil {
		return nil, err
	}

	if _, found := ofs.NotExpected[name]; found {
		return nil, ErrShouldNotBeReached
	}

	return ForbiddenFile{
		File:        file,
		notExpected: ofs.NotExpected,
		folder:      name,
	}, nil
}

func (ofs ForbiddenFS) ReadDir(name string) ([]fs.DirEntry, error) {
	entries, err := ofs.MapFS.ReadDir(name)
	if err != nil {
		return nil, err
	}

	if _, found := ofs.NotExpected[name]; found {
		return nil, ErrShouldNotBeReached
	}

	for i := range entries {
		entries[i] = ForbiddenDirEntry{
			DirEntry:    entries[i],
			notExpected: ofs.NotExpected,
			folder:      name,
		}
	}

	return entries, nil
}

func (ofs ForbiddenFS) Stat(name string) (fs.FileInfo, error) {
	fileInfo, err := ofs.MapFS.Stat(name)
	if err != nil {
		return nil, err
	}

	if _, found := ofs.NotExpected[name]; found {
		return nil, ErrShouldNotBeReached
	}

	return fileInfo, nil
}

type ForbiddenFile struct {
	fs.File
	notExpected map[string]struct{}
	folder      string
}

func (ofs ForbiddenFile) Stat() (fs.FileInfo, error) {
	fileInfo, err := ofs.File.Stat()
	if err != nil {
		return nil, err
	}

	if _, found := ofs.notExpected[filepath.Join(ofs.folder, fileInfo.Name())]; found {
		return nil, ErrShouldNotBeReached
	}

	return fileInfo, nil
}

type ForbiddenDirEntry struct {
	fs.DirEntry
	notExpected map[string]struct{}
	folder      string
}

func (ofs ForbiddenDirEntry) Info() (fs.FileInfo, error) {
	fileInfo, err := ofs.DirEntry.Info()
	if err != nil {
		return nil, err
	}

	if _, found := ofs.notExpected[filepath.Join(ofs.folder, fileInfo.Name())]; found {
		return nil, ErrShouldNotBeReached
	}

	return fileInfo, nil
}

func TestNoGo_AddFromFS_ignored_nested_files(t *testing.T) {
	// This tests a bug where AddFromFS did walk the whole tree because
	// the nogo-instance was not mutated with found .gitingore files.
	// (due to missing *-receivers in the walk methods)

	type args struct {
		fsys fs.FS
	}
	tests := []struct {
		name    string
		args    args
		wantErr assert.ErrorAssertionFunc
	}{
		{
			name: "nested ignored files should not be loaded to speed up ignore-file loading",
			args: args{
				fsys: ForbiddenFS{
					NotExpected: map[string]struct{}{
						"ignoredFolder/.gitignore":  {},
						"ignoredFolder/sub":         {},
						"notIgnored/subignored/lol": {},
					},
					MapFS: fstest.MapFS{
						"ignoredFolder": &fstest.MapFile{
							Mode: fs.ModeDir,
						},
						"ignoredFolder/sub": &fstest.MapFile{
							Mode: fs.ModeDir,
						},
						"ignoredFolder/sub/something": &fstest.MapFile{},
						"ignoredFolder/.gitignore":    &fstest.MapFile{},
						"aFile":                       &fstest.MapFile{},
						".gitignore": &fstest.MapFile{
							Data: []byte("ignoredFolder"),
						},
						"notIgnored/sub": &fstest.MapFile{},
						"notIgnored/.gitignore": &fstest.MapFile{
							Data: []byte("subignored"),
						},
						"notIgnored/subignored/lol": &fstest.MapFile{},
					},
				},
			},
			wantErr: assert.NoError,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			n := &NoGo{}

			assert.NoError(t, n.AddFromFS(tt.args.fsys, ".gitignore"))
		})
	}
}

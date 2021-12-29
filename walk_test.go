package nogo

import (
	"errors"
	"fmt"
	"io/fs"
	"regexp"
	"testing"

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

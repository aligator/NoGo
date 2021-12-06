package nogo

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestResult_Resolve(t *testing.T) {
	type fields struct {
		Rule        Rule
		Found       bool
		ParentMatch bool
	}
	type args struct {
		isDir bool
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   bool
	}{
		{
			name: "file",
			fields: fields{
				Rule: Rule{
					Negate:     false,
					OnlyFolder: false,
				},
				Found:       true,
				ParentMatch: false,
			},
			args: args{
				isDir: false,
			},
			want: true,
		},
		{
			name: "folder",
			fields: fields{
				Rule: Rule{
					Negate:     false,
					OnlyFolder: false,
				},
				Found:       true,
				ParentMatch: false,
			},
			args: args{
				isDir: true,
			},
			want: true,
		},
		{
			name: "file - onlyFolder",
			fields: fields{
				Rule: Rule{
					Negate:     false,
					OnlyFolder: true,
				},
				Found:       true,
				ParentMatch: false,
			},
			args: args{
				isDir: false,
			},
			want: false,
		},
		{
			name: "folder - onlyFolder",
			fields: fields{
				Rule: Rule{
					Negate:     false,
					OnlyFolder: true,
				},
				Found:       true,
				ParentMatch: false,
			},
			args: args{
				isDir: true,
			},
			want: true,
		},
		{
			name: "file - negated",
			fields: fields{
				Rule: Rule{
					Negate:     true,
					OnlyFolder: false,
				},
				Found:       true,
				ParentMatch: false,
			},
			args: args{
				isDir: false,
			},
			want: false,
		},
		{
			name: "folder - negated",
			fields: fields{
				Rule: Rule{
					Negate:     true,
					OnlyFolder: false,
				},
				Found:       true,
				ParentMatch: false,
			},
			args: args{
				isDir: true,
			},
			want: false,
		},
		{
			name: "file - onlyFolder negated",
			fields: fields{
				Rule: Rule{
					Negate:     true,
					OnlyFolder: true,
				},
				Found:       true,
				ParentMatch: false,
			},
			args: args{
				isDir: false,
			},
			want: false,
		},
		{
			name: "folder - onlyFolder negated",
			fields: fields{
				Rule: Rule{
					Negate:     true,
					OnlyFolder: true,
				},
				Found:       true,
				ParentMatch: false,
			},
			args: args{
				isDir: true,
			},
			want: false,
		},
		{
			name: "file - onlyFolder not-parentMatch",
			fields: fields{
				Rule: Rule{
					Negate:     false,
					OnlyFolder: true,
				},
				Found:       true,
				ParentMatch: false,
			},
			args: args{
				isDir: false,
			},
			want: false,
		},
		{
			name: "file - onlyFolder parentMatch",
			fields: fields{
				Rule: Rule{
					Negate:     false,
					OnlyFolder: true,
				},
				Found:       true,
				ParentMatch: true,
			},
			args: args{
				isDir: false,
			},
			want: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := Result{
				Rule:        tt.fields.Rule,
				Found:       tt.fields.Found,
				ParentMatch: tt.fields.ParentMatch,
			}
			assert.Equalf(t, tt.want, r.Resolve(tt.args.isDir), "Resolve(%v)", tt.args.isDir)
		})
	}
}

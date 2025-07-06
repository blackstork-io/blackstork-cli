package resolver

import (
	"context"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"testing"

	"github.com/stretchr/testify/require"
)

type mockFile struct {
	path    string
	content string
	isDir   bool
}

func mockFileDir(t *testing.T, files []mockFile) string {
	t.Helper()
	tmpDir := t.TempDir()
	for _, file := range files {
		if file.isDir {
			err := os.MkdirAll(filepath.Join(tmpDir, file.path), 0o755)
			require.NoError(t, err)
			continue
		}
		err := os.MkdirAll(filepath.Dir(filepath.Join(tmpDir, file.path)), 0o755)
		require.NoError(t, err)
		err = os.WriteFile(filepath.Join(tmpDir, file.path), []byte(file.content), 0o644)
		require.NoError(t, err)
	}
	return tmpDir
}

func TestLocalSource_Lookup(t *testing.T) {
	type fields struct {
		Path string
	}
	type args struct {
		ctx  context.Context
		name Name
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    []Version
		wantErr bool
	}{
		{
			name: "no path",
			fields: fields{
				Path: "",
			},
			args: args{
				ctx:  context.Background(),
				name: Name{"blackstork", "sqlite"},
			},
			want:    nil,
			wantErr: true,
		},
		{
			name: "no version",
			fields: fields{
				Path: t.TempDir(),
			},
			args: args{
				ctx:  context.Background(),
				name: Name{"blackstork", "sqlite"},
			},
			want:    nil,
			wantErr: false,
		},
		{
			name: "one version",
			fields: fields{
				Path: mockFileDir(t, []mockFile{
					{path: "blackstork/sqlite@1.0.0", content: "plugin"},
				}),
			},
			args: args{
				ctx:  context.Background(),
				name: Name{"blackstork", "sqlite"},
			},
			want:    []Version{mustVersion(t, "1.0.0")},
			wantErr: false,
		},
		{
			name: "one version with exe",
			fields: fields{
				Path: mockFileDir(t, []mockFile{
					{path: "blackstork/sqlite@1.0.0.exe", content: "plugin"},
				}),
			},
			args: args{
				ctx:  context.Background(),
				name: Name{"blackstork", "sqlite"},
			},
			want:    []Version{mustVersion(t, "1.0.0")},
			wantErr: false,
		},
		{
			name: "multiple version",
			fields: fields{
				Path: mockFileDir(t, []mockFile{
					{path: "blackstork/sqlite@1.0.0", content: "plugin"},
					{path: "blackstork/sqlite@1.0.1", content: "plugin"},
				}),
			},
			args: args{
				ctx:  context.Background(),
				name: Name{"blackstork", "sqlite"},
			},
			want:    []Version{mustVersion(t, "1.0.0"), mustVersion(t, "1.0.1")},
			wantErr: false,
		},
		{
			name: "multiple version with exe",
			fields: fields{
				Path: mockFileDir(t, []mockFile{
					{path: "blackstork/sqlite@1.0.0.exe", content: "plugin"},
					{path: "blackstork/sqlite@1.0.1.exe", content: "plugin"},
				}),
			},
			args: args{
				ctx:  context.Background(),
				name: Name{"blackstork", "sqlite"},
			},
			want:    []Version{mustVersion(t, "1.0.0"), mustVersion(t, "1.0.1")},
			wantErr: false,
		},
		{
			name: "skip invalid version",
			fields: fields{
				Path: mockFileDir(t, []mockFile{
					{path: "blackstork/sqlite@1.0.0", content: "plugin"},
					{path: "blackstork/sqlite@invalid", content: "plugin"},
					{path: "blackstork/@", content: "plugin"},
					{path: "blackstork/@1.0.0", content: "plugin"},
					{path: "blackstork/sqlite@", content: "plugin"},
					{path: "blackstork/@@", content: "plugin"},
				}),
			},
			args: args{
				ctx:  context.Background(),
				name: Name{"blackstork", "sqlite"},
			},
			want:    []Version{mustVersion(t, "1.0.0")},
			wantErr: false,
		},
		{
			name: "skip non-matching name",
			fields: fields{
				Path: mockFileDir(t, []mockFile{
					{path: "blackstork/sqlite@1.0.0", content: "plugin"},
					{path: "blackstork/other@1.0.1", content: "plugin"},
				}),
			},
			args: args{
				ctx:  context.Background(),
				name: Name{"blackstork", "sqlite"},
			},
			want:    []Version{mustVersion(t, "1.0.0")},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			source := NewLocal(tt.fields.Path, nil, nil)
			got, err := source.Lookup(tt.args.ctx, tt.args.name)
			if (err != nil) != tt.wantErr {
				t.Errorf("LocalSource.Lookup() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("LocalSource.Lookup() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestLocalSource_Resolve(t *testing.T) {
	type fields struct {
		Path string
	}
	type args struct {
		ctx       context.Context
		name      Name
		version   Version
		checksums []Checksum
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    *ResolvedPlugin
		wantErr bool
	}{
		{
			name: "no path",
			fields: fields{
				Path: "",
			},
			args: args{
				ctx:       context.Background(),
				name:      Name{"blackstork", "sqlite"},
				version:   mustVersion(t, "1.0.0"),
				checksums: nil,
			},
			want:    nil,
			wantErr: true,
		},
		{
			name: "no version",
			fields: fields{
				Path: t.TempDir(),
			},
			args: args{
				ctx:       context.Background(),
				name:      Name{"blackstork", "sqlite"},
				version:   mustVersion(t, "1.0.0"),
				checksums: nil,
			},
			want:    nil,
			wantErr: true,
		},
		{
			name: "just binary",
			fields: fields{
				Path: mockFileDir(t, []mockFile{
					{path: "blackstork/sqlite@1.0.0", content: "plugin"},
				}),
			},
			args: args{
				ctx:       context.Background(),
				name:      Name{"blackstork", "sqlite"},
				version:   mustVersion(t, "1.0.0"),
				checksums: nil,
			},
			want: &ResolvedPlugin{
				BinaryPath: filepath.Join("blackstork", "sqlite@1.0.0"),
				Checksums:  []Checksum{mustChecksum(t, "binary:"+runtime.GOOS+":"+runtime.GOARCH+":XmieKwFnK/M5luddXjcv9gxTbOFZmhRY6GfNj0vvUWA=")},
			},
			wantErr: false,
		},
		{
			name: "just binary with exe",
			fields: fields{
				Path: mockFileDir(t, []mockFile{
					{path: "blackstork/sqlite@1.0.0.exe", content: "plugin"},
				}),
			},
			args: args{
				ctx:       context.Background(),
				name:      Name{"blackstork", "sqlite"},
				version:   mustVersion(t, "1.0.0"),
				checksums: nil,
			},
			want: &ResolvedPlugin{
				BinaryPath: filepath.Join("blackstork", "sqlite@1.0.0.exe"),
				Checksums:  []Checksum{mustChecksum(t, "binary:"+runtime.GOOS+":"+runtime.GOARCH+":XmieKwFnK/M5luddXjcv9gxTbOFZmhRY6GfNj0vvUWA=")},
			},
			wantErr: false,
		},
		{
			name: "binary and checksum",
			fields: fields{
				Path: mockFileDir(t, []mockFile{
					{path: "blackstork/sqlite@1.0.0", content: "plugin"},
					{
						path: "blackstork/sqlite@1.0.0_checksums.txt",
						content: "archive:darwin:amd64:XmieKwFnK/M5luddXjcv9gxTbOFZmhRY6GfNj0vvUWA=\n" +
							"archive:linux:amd64:YmieKwFnK/M5luddXjcv9gxTbOFZmhRY6GfNj0vvUWA=\n" +
							"archive:windows:amd64:ZmieKwFnK/M5luddXjcv9gxTbOFZmhRY6GfNj0vvUWA=\n" +
							"binary:" + runtime.GOOS + ":" + runtime.GOARCH + ":XmieKwFnK/M5luddXjcv9gxTbOFZmhRY6GfNj0vvUWA=",
					},
				}),
			},
			args: args{
				ctx:       context.Background(),
				name:      Name{"blackstork", "sqlite"},
				version:   mustVersion(t, "1.0.0"),
				checksums: nil,
			},
			want: &ResolvedPlugin{
				BinaryPath: filepath.Join("blackstork", "sqlite@1.0.0"),
				Checksums: []Checksum{
					mustChecksum(t, "archive:darwin:amd64:XmieKwFnK/M5luddXjcv9gxTbOFZmhRY6GfNj0vvUWA="),
					mustChecksum(t, "archive:linux:amd64:YmieKwFnK/M5luddXjcv9gxTbOFZmhRY6GfNj0vvUWA="),
					mustChecksum(t, "archive:windows:amd64:ZmieKwFnK/M5luddXjcv9gxTbOFZmhRY6GfNj0vvUWA="),
					mustChecksum(t, "binary:"+runtime.GOOS+":"+runtime.GOARCH+":XmieKwFnK/M5luddXjcv9gxTbOFZmhRY6GfNj0vvUWA="),
				},
			},
			wantErr: false,
		},
		{
			name: "binary checksum does not match with input",
			fields: fields{
				Path: mockFileDir(t, []mockFile{
					{path: "blackstork/sqlite@1.0.0", content: "plugin"},
				}),
			},
			args: args{
				ctx:       context.Background(),
				name:      Name{"blackstork", "sqlite"},
				version:   mustVersion(t, "1.0.0"),
				checksums: []Checksum{mustChecksum(t, "binary:"+runtime.GOOS+":"+runtime.GOARCH+":XmieKwFnK/M5luddXjcv9gxTbOFZmhRY6GfNj1vvUWA=")},
			},
			want:    nil,
			wantErr: true,
		},
		{
			name: "binary checksum does match with input",
			fields: fields{
				Path: mockFileDir(t, []mockFile{
					{path: "blackstork/sqlite@1.0.0", content: "plugin"},
				}),
			},
			args: args{
				ctx:       context.Background(),
				name:      Name{"blackstork", "sqlite"},
				version:   mustVersion(t, "1.0.0"),
				checksums: []Checksum{mustChecksum(t, "binary:"+runtime.GOOS+":"+runtime.GOARCH+":XmieKwFnK/M5luddXjcv9gxTbOFZmhRY6GfNj0vvUWA=")},
			},
			want: &ResolvedPlugin{
				BinaryPath: filepath.Join("blackstork", "sqlite@1.0.0"),
				Checksums:  []Checksum{mustChecksum(t, "binary:"+runtime.GOOS+":"+runtime.GOARCH+":XmieKwFnK/M5luddXjcv9gxTbOFZmhRY6GfNj0vvUWA=")},
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			source := NewLocal(tt.fields.Path, nil, nil)
			got, err := source.Resolve(tt.args.ctx, tt.args.name, tt.args.version, tt.args.checksums)
			if (err != nil) != tt.wantErr {
				t.Errorf("LocalSource.Resolve() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.want != nil {
				tt.want.BinaryPath = filepath.Join(tt.fields.Path, tt.want.BinaryPath)
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("LocalSource.Resolve() = %v, want %v", got, tt.want)
			}
		})
	}
}

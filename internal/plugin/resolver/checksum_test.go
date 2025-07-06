package resolver

import (
	"bytes"
	"encoding/base64"
	"io"
	"reflect"
	"slices"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func mustBase64(t *testing.T, s string) []byte {
	sum, err := base64.StdEncoding.DecodeString(s)
	require.NoError(t, err)
	return sum
}

func TestPluginChecksum_UnmarshalJSON(t *testing.T) {
	str := `"archive:darwin:arm64:lgNgp5LO81yt1boBsiaNsJCzLWD9r5ovW+el5k/dDZ8="`
	var dst Checksum
	err := dst.UnmarshalJSON([]byte(str))
	require.NoError(t, err)
	require.Equal(t, "archive", dst.Object)
	require.Equal(t, "darwin", dst.OS)
	require.Equal(t, "arm64", dst.Arch)
	require.Equal(t, "lgNgp5LO81yt1boBsiaNsJCzLWD9r5ovW+el5k/dDZ8=", base64.StdEncoding.EncodeToString(dst.Sum))
}

func TestPluginChecksum_MarshalJSON(t *testing.T) {
	sum, _ := base64.StdEncoding.DecodeString("lgNgp5LO81yt1boBsiaNsJCzLWD9r5ovW+el5k/dDZ8=")
	src := Checksum{
		Object: "archive",
		OS:     "darwin",
		Arch:   "arm64",
		Sum:    sum,
	}
	data, err := src.MarshalJSON()
	require.NoError(t, err)
	require.Equal(t, `"archive:darwin:arm64:lgNgp5LO81yt1boBsiaNsJCzLWD9r5ovW+el5k/dDZ8="`, string(data))
}

func Test_encodeChecksums(t *testing.T) {
	type args struct {
		checksums []Checksum
	}
	tests := []struct {
		name    string
		args    args
		wantW   string
		wantErr bool
	}{
		{
			name: "empty",
			args: args{
				checksums: nil,
			},
			wantW:   "",
			wantErr: false,
		},
		{
			name: "single",
			args: args{
				checksums: []Checksum{
					{
						Object: "archive",
						OS:     "darwin",
						Arch:   "arm64",
						Sum:    mustBase64(t, "lgNgp5LO81yt1boBsiaNsJCzLWD9r5ovW+el5k/dDZ8="),
					},
				},
			},
			wantW:   "archive:darwin:arm64:lgNgp5LO81yt1boBsiaNsJCzLWD9r5ovW+el5k/dDZ8=\n",
			wantErr: false,
		},
		{
			name: "multiple",
			args: args{
				checksums: []Checksum{
					{
						Object: "archive",
						OS:     "darwin",
						Arch:   "arm64",
						Sum:    mustBase64(t, "lgNgp5LO81yt1boBsiaNsJCzLWD9r5ovW+el5k/dDZ8="),
					},
					{
						Object: "binary",
						OS:     "linux",
						Arch:   "amd64",
						Sum:    mustBase64(t, "lgNgp5LO81yt1boBsiaNsJCzLWD9r5ovW+el5k/dDZ8="),
					},
				},
			},
			wantW: "archive:darwin:arm64:lgNgp5LO81yt1boBsiaNsJCzLWD9r5ovW+el5k/dDZ8=\n" +
				"binary:linux:amd64:lgNgp5LO81yt1boBsiaNsJCzLWD9r5ovW+el5k/dDZ8=\n",
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := &bytes.Buffer{}
			if err := encodeChecksums(w, tt.args.checksums); (err != nil) != tt.wantErr {
				t.Errorf("encodeChecksums() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if gotW := w.String(); gotW != tt.wantW {
				t.Errorf("encodeChecksums() = %v, want %v", gotW, tt.wantW)
			}
		})
	}
}

func Test_decodeChecksums(t *testing.T) {
	type args struct {
		r io.Reader
	}
	tests := []struct {
		name    string
		args    args
		want    []Checksum
		wantErr bool
	}{
		{
			name: "empty",
			args: args{
				r: bytes.NewBufferString(""),
			},
			want:    nil,
			wantErr: false,
		},
		{
			name: "single",
			args: args{
				r: bytes.NewBufferString("archive:darwin:arm64:lgNgp5LO81yt1boBsiaNsJCzLWD9r5ovW+el5k/dDZ8=\n"),
			},
			want: []Checksum{
				{
					Object: "archive",
					OS:     "darwin",
					Arch:   "arm64",
					Sum:    mustBase64(t, "lgNgp5LO81yt1boBsiaNsJCzLWD9r5ovW+el5k/dDZ8="),
				},
			},
			wantErr: false,
		},
		{
			name: "multiple",
			args: args{
				r: bytes.NewBufferString("archive:darwin:arm64:lgNgp5LO81yt1boBsiaNsJCzLWD9r5ovW+el5k/dDZ8=\n" +
					"binary:linux:amd64:lgNgp5LO81yt1boBsiaNsJCzLWD9r5ovW+el5k/dDZ8=\n"),
			},
			want: []Checksum{
				{
					Object: "archive",
					OS:     "darwin",
					Arch:   "arm64",
					Sum:    mustBase64(t, "lgNgp5LO81yt1boBsiaNsJCzLWD9r5ovW+el5k/dDZ8="),
				},
				{
					Object: "binary",
					OS:     "linux",
					Arch:   "amd64",
					Sum:    mustBase64(t, "lgNgp5LO81yt1boBsiaNsJCzLWD9r5ovW+el5k/dDZ8="),
				},
			},
			wantErr: false,
		},
		{
			name: "invalid",
			args: args{
				r: bytes.NewBufferString("invalid\n"),
			},
			want:    nil,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := decodeChecksums(tt.args.r)
			if (err != nil) != tt.wantErr {
				t.Errorf("decodeChecksums() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("decodeChecksums() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestChecksum_Compare(t *testing.T) {
	strList := []string{
		"archive:darwin:arm64:lgNgp5LO81yt1boBsiaNsJCzLWD9r5ovW+el5k/dDZ8=",
		"binary:darwin:arm64:lgNgp5LO81yt1boBsiaNsJCzLWD9r5ovW+el5k/dDZ8=",
		"binary:windows:amd64:lgNgp5LO81yt1boBsiaNsJCzLWD9r5ovW+el5k/dDZ8=",
		"binary:darwin:amd64:lgNgp5LO81yt1boBsiaNsJCzLWD9r5ovW+el5k/dDZ8=",
		"binary:linux:arm64:lgNgp5LO81yt1boBsiaNsJCzLWD9r5ovW+el5k/dDZ8=",
		"archive:windows:amd64:lgNgp5LO81yt1boBsiaNsJCzLWD9r5ovW+el5k/dDZ8=",
		"archive:linux:arm64:lgNgp5LO81yt1boBsiaNsJCzLWD9r5ovW+el5k/dDZ8=",
		"archive:darwin:amd64:lgNgp5LO81yt1boBsiaNsJCzLWD9r5ovW+el5k/dDZ8=",
		"archive:linux:amd64:lgNgp5LO81yt1boBsiaNsJCzLWD9r5ovW+el5k/dDZ8=",
		"binary:linux:amd64:lgNgp5LO81yt1boBsiaNsJCzLWD9r5ovW+el5k/dDZ8=",
		"archive:windows:arm64:lgNgp5LO81yt1boBsiaNsJCzLWD9r5ovW+el5k/dDZ8=",
		"binary:windows:arm64:lgNgp5LO81yt1boBsiaNsJCzLWD9r5ovW+el5k/dDZ8=",
	}
	checksums := make([]Checksum, len(strList))
	for i, str := range strList {
		checksums[i].UnmarshalText([]byte(str))
	}
	slices.SortFunc(checksums, func(a, b Checksum) int {
		return a.Compare(b)
	})

	gotStrList := make([]string, len(checksums))
	for i, c := range checksums {
		gotStrList[i] = c.String()
	}

	assert.Exactly(t, []string{
		"archive:darwin:amd64:lgNgp5LO81yt1boBsiaNsJCzLWD9r5ovW+el5k/dDZ8=",
		"archive:darwin:arm64:lgNgp5LO81yt1boBsiaNsJCzLWD9r5ovW+el5k/dDZ8=",
		"archive:linux:amd64:lgNgp5LO81yt1boBsiaNsJCzLWD9r5ovW+el5k/dDZ8=",
		"archive:linux:arm64:lgNgp5LO81yt1boBsiaNsJCzLWD9r5ovW+el5k/dDZ8=",
		"archive:windows:amd64:lgNgp5LO81yt1boBsiaNsJCzLWD9r5ovW+el5k/dDZ8=",
		"archive:windows:arm64:lgNgp5LO81yt1boBsiaNsJCzLWD9r5ovW+el5k/dDZ8=",
		"binary:darwin:amd64:lgNgp5LO81yt1boBsiaNsJCzLWD9r5ovW+el5k/dDZ8=",
		"binary:darwin:arm64:lgNgp5LO81yt1boBsiaNsJCzLWD9r5ovW+el5k/dDZ8=",
		"binary:linux:amd64:lgNgp5LO81yt1boBsiaNsJCzLWD9r5ovW+el5k/dDZ8=",
		"binary:linux:arm64:lgNgp5LO81yt1boBsiaNsJCzLWD9r5ovW+el5k/dDZ8=",
		"binary:windows:amd64:lgNgp5LO81yt1boBsiaNsJCzLWD9r5ovW+el5k/dDZ8=",
		"binary:windows:arm64:lgNgp5LO81yt1boBsiaNsJCzLWD9r5ovW+el5k/dDZ8=",
	}, gotStrList)
}

func mustChecksum(t *testing.T, str string) Checksum {
	t.Helper()
	var c Checksum
	err := c.UnmarshalText([]byte(str))
	require.NoError(t, err)
	return c
}

func TestChecksum_Match(t *testing.T) {
	tests := []struct {
		name string
		sum  Checksum
		args []Checksum
		want bool
	}{
		{
			name: "match",
			sum:  mustChecksum(t, "archive:darwin:arm64:lgNgp5LO81yt1boBsiaNsJCzLWD9r5ovW+el5k/dDZ8="),
			args: []Checksum{
				mustChecksum(t, "binary:linux:amd64:lgNgp5LO81yt1boBsiaNsJCzLWD9r5ovW+el5k/dDZ8="),
				mustChecksum(t, "archive:darwin:arm64:lgNgp5LO81yt1boBsiaNsJCzLWD9r5ovW+el5k/dDZ8="),
				mustChecksum(t, "binary:darwin:arm64:lgNgp5LO81yt1boBsiaNsJCzLWD9r5ovW+el5k/dDZ8="),
			},
			want: true,
		},
		{
			name: "empty",
			sum:  mustChecksum(t, "archive:darwin:arm64:lgNgp5LO81yt1boBsiaNsJCzLWD9r5ovW+el5k/dDZ8="),
			args: nil,
			want: false,
		},
		{
			name: "mismatch",
			sum:  mustChecksum(t, "archive:darwin:arm64:lgNgp5LO81yt1boBsiaNsJCzLWD9r5ovW+el5k/dDZ8="),
			args: []Checksum{
				mustChecksum(t, "binary:linux:amd64:lgNgp5LO81yt1boBsiaNsJCzLWD9r5ovW+el5k/dDZ8="),
				mustChecksum(t, "binary:darwin:arm64:lgNgp5LO81yt1boBsiaNsJCzLWD9r5ovW+el5k/dDZ8="),
			},
			want: false,
		},
		{
			name: "mismatch_os",
			sum:  mustChecksum(t, "archive:darwin:arm64:lgNgp5LO81yt1boBsiaNsJCzLWD9r5ovW+el5k/dDZ8="),
			args: []Checksum{
				mustChecksum(t, "archive:linux:amd64:lgNgp5LO81yt1boBsiaNsJCzLWD9r5ovW+el5k/dDZ8="),
				mustChecksum(t, "binary:darwin:arm64:lgNgp5LO81yt1boBsiaNsJCzLWD9r5ovW+el5k/dDZ8="),
			},
			want: false,
		},
		{
			name: "mismatch_arch",
			sum:  mustChecksum(t, "archive:darwin:arm64:lgNgp5LO81yt1boBsiaNsJCzLWD9r5ovW+el5k/dDZ8="),
			args: []Checksum{
				mustChecksum(t, "archive:darwin:amd64:lgNgp5LO81yt1boBsiaNsJCzLWD9r5ovW+el5k/dDZ8="),
				mustChecksum(t, "binary:darwin:arm64:lgNgp5LO81yt1boBsiaNsJCzLWD9r5ovW+el5k/dDZ8="),
			},
			want: false,
		},
		{
			name: "mismatch_sum",
			sum:  mustChecksum(t, "archive:darwin:arm64:lgNgp5LO81yt1boBsiaNsJCzLWD9r5ovW+el5k/dDZ8="),
			args: []Checksum{
				mustChecksum(t, "archive:darwin:arm64:lgNgp5LO81yt1boBsiaNsJCzLWD9r5ovW+el5k/dDa8="),
				mustChecksum(t, "binary:darwin:arm64:lgNgp5LO81yt1boBsiaNsJCzLWD9r5ovW+el5k/dDZ8="),
			},
			want: false,
		},
		{
			name: "empty_sum",
			sum:  mustChecksum(t, ":::"),
			args: []Checksum{
				mustChecksum(t, "archive:darwin:arm64:lgNgp5LO81yt1boBsiaNsJCzLWD9r5ovW+el5k/dDZ9="),
				mustChecksum(t, "binary:darwin:arm64:lgNgp5LO81yt1boBsiaNsJCzLWD9r5ovW+el5k/dDZ8="),
			},
			want: false,
		},
		{
			name: "empty_args",
			sum:  mustChecksum(t, "archive:darwin:arm64:lgNgp5LO81yt1boBsiaNsJCzLWD9r5ovW+el5k/dDZ9="),
			args: nil,
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.sum.Match(tt.args); got != tt.want {
				t.Errorf("Checksum.Match() = %v, want %v", got, tt.want)
			}
		})
	}
}

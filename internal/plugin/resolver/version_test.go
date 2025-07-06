package resolver

import (
	"reflect"
	"testing"

	"github.com/Masterminds/semver/v3"
	"github.com/stretchr/testify/require"
)

func mustVersion(t *testing.T, str string) Version {
	t.Helper()
	v, err := semver.NewVersion(str)
	require.NoError(t, err)
	return Version{v}
}

func TestParseConstraintMap(t *testing.T) {
	type args struct {
		src map[string]string
	}
	tests := []struct {
		name    string
		args    args
		want    ConstraintMap
		wantErr bool
	}{
		{
			name: "nil",
			args: args{
				src: nil,
			},
			want:    nil,
			wantErr: false,
		},
		{
			name: "empty",
			args: args{
				src: map[string]string{},
			},
			want:    ConstraintMap{},
			wantErr: false,
		},
		{
			name: "single",
			args: args{
				src: map[string]string{
					"ns/name": "1.0.0",
				},
			},
			want: ConstraintMap{
				Name{"ns", "name"}: mustConstraint(t, "1.0.0"),
			},
			wantErr: false,
		},
		{
			name: "with_v_prefix",
			args: args{
				src: map[string]string{
					"ns/name": "v1.0.0",
				},
			},
			want: ConstraintMap{
				Name{"ns", "name"}: mustConstraint(t, "v1.0.0"),
			},
			wantErr: false,
		},
		{
			name: "multiple",
			args: args{
				src: map[string]string{
					"ns/name1": "1.0.0",
					"ns/name2": "2.0.0",
					"ns/name3": "3.0.0",
				},
			},
			want: ConstraintMap{
				Name{"ns", "name1"}: mustConstraint(t, "1.0.0"),
				Name{"ns", "name2"}: mustConstraint(t, "2.0.0"),
				Name{"ns", "name3"}: mustConstraint(t, "3.0.0"),
			},
			wantErr: false,
		},
		{
			name: "invalid_name",
			args: args{
				src: map[string]string{
					"ns": "1.0.0",
				},
			},
			want:    nil,
			wantErr: true,
		},
		{
			name: "invalid_version",
			args: args{
				src: map[string]string{
					"ns/name": "",
				},
			},
			want:    nil,
			wantErr: true,
		},
		{
			name: "invalid_version_constraint",
			args: args{
				src: map[string]string{
					"ns/name": "1.0.0+",
				},
			},
			want:    nil,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseConstraintMap(tt.args.src)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseVersionMap() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("parseVersionMap() = %v, want %v", got, tt.want)
			}
		})
	}
}

func mustConstraint(t *testing.T, str string) *semver.Constraints {
	t.Helper()
	c, err := semver.NewConstraint(str)
	require.NoError(t, err)
	return c
}

func TestVersion_UnmarshalJSON(t *testing.T) {
	v := new(Version)
	err := v.UnmarshalJSON([]byte(`"1.0.0"`))
	require.NoError(t, err)
	require.Equal(t, mustVersion(t, "1.0.0"), *v)
	v = new(Version)
	err = v.UnmarshalJSON([]byte(`"v1.0.0"`))
	require.Error(t, err)
	err = v.UnmarshalJSON([]byte(`"1.0"`))
	require.Error(t, err)
	err = v.UnmarshalJSON([]byte(`"1"`))
	require.Error(t, err)
}

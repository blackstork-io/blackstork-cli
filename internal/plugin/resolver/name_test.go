package resolver

import (
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
)

func mustName(t *testing.T, str string) Name {
	t.Helper()
	name, err := ParseName(str)
	assert.NoError(t, err)
	return name
}

func TestName_String(t *testing.T) {
	name := Name{"namespace", "short"}
	assert.Equal(t, "namespace/short", name.String())
	assert.Equal(t, "namespace", name.Namespace())
	assert.Equal(t, "short", name.Short())
}

func TestName_Compare(t *testing.T) {
	tests := []struct {
		name  string
		name1 Name
		name2 Name
		want  int
	}{
		{
			name:  "equal",
			name1: Name{"namespace", "short"},
			name2: Name{"namespace", "short"},
			want:  0,
		},
		{
			name:  "namespace",
			name1: Name{"namespace1", "short"},
			name2: Name{"namespace2", "short"},
			want:  -1,
		},
		{
			name:  "short",
			name1: Name{"namespace", "short1"},
			name2: Name{"namespace", "short2"},
			want:  -1,
		},
		{
			name:  "both",
			name1: Name{"namespace1", "short1"},
			name2: Name{"namespace2", "short2"},
			want:  -1,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.name1.Compare(tt.name2); got != tt.want {
				t.Errorf("Name.Compare() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestName_JSON(t *testing.T) {
	name := Name{"ns", "name"}
	data, err := name.MarshalJSON()
	assert.NoError(t, err)
	assert.Equal(t, `"ns/name"`, string(data))

	var name2 Name
	err = name2.UnmarshalJSON(data)
	assert.NoError(t, err)
	assert.Equal(t, name, name2)
}

func TestParseName(t *testing.T) {
	type args struct {
		name string
	}
	tests := []struct {
		name    string
		args    args
		want    Name
		wantErr bool
	}{
		{
			name: "simple",
			args: args{
				name: "namespace/short",
			},
			want:    Name{"namespace", "short"},
			wantErr: false,
		},
		{
			name: "empty",
			args: args{
				name: "",
			},
			want:    Name{},
			wantErr: true,
		},
		{
			name: "no_slash",
			args: args{
				name: "namespace",
			},
			want:    Name{},
			wantErr: true,
		},
		{
			name: "no_short",
			args: args{
				name: "namespace/",
			},
			want:    Name{},
			wantErr: true,
		},
		{
			name: "no_namespace",
			args: args{
				name: "/short",
			},
			want:    Name{},
			wantErr: true,
		},
		{
			name: "only_slash",
			args: args{
				name: "/",
			},
			want:    Name{},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseName(tt.args.name)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseName() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ParseName() = %v, want %v", got, tt.want)
			}
		})
	}
}

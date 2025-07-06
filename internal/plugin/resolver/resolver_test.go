package resolver

import (
	"context"
	"testing"

	"github.com/hashicorp/hcl/v2"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestResolver_Install(t *testing.T) {
	source := newMockSource(t)
	resolver, diags := NewResolver(map[string]string{
		"blackstork/sqlite": ">= 1.0 < 2.0",
	}, WithSources(source))
	require.Len(t, diags, 0)
	require.NotNil(t, resolver)
	source.EXPECT().Lookup(mock.Anything, Name{"blackstork", "sqlite"}).Return([]Version{
		mustVersion(t, "1.0.0"),
		mustVersion(t, "1.0.1"),
		mustVersion(t, "1.0.2"),
	}, nil)
	source.EXPECT().Resolve(mock.Anything, Name{"blackstork", "sqlite"}, mustVersion(t, "1.0.2"), []Checksum(nil)).Return(&ResolvedPlugin{
		BinaryPath: "/blackstork/sqlite/1.0.2",
		Checksums: []Checksum{
			mustChecksum(t, "archive:linux:amd64:lgNgp5LO81yt1boBsiaNsJCzLWD9r5ovW+el5k/dDZ8="),
			mustChecksum(t, "binary:linux:amd64:lgNgp5LO81yt1boBsiaNsJCzLWD9r5ovW+el5k/dDZ8="),
		},
	}, nil)
	lockFile, diags := resolver.Install(context.Background(), &LockFile{}, false)
	require.Len(t, diags, 0)
	require.Equal(t, &LockFile{
		Plugins: []PluginLock{
			{
				Name:    Name{"blackstork", "sqlite"},
				Version: mustVersion(t, "1.0.2"),
				Checksums: []Checksum{
					mustChecksum(t, "archive:linux:amd64:lgNgp5LO81yt1boBsiaNsJCzLWD9r5ovW+el5k/dDZ8="),
					mustChecksum(t, "binary:linux:amd64:lgNgp5LO81yt1boBsiaNsJCzLWD9r5ovW+el5k/dDZ8="),
				},
			},
		},
	}, lockFile)
	source.AssertExpectations(t)
	// again with checksums
	source.EXPECT().Lookup(mock.Anything, Name{"blackstork", "sqlite"}).Return([]Version{
		mustVersion(t, "1.0.0"),
		mustVersion(t, "1.0.1"),
		mustVersion(t, "1.0.2"),
	}, nil)
	source.EXPECT().Resolve(mock.Anything, Name{"blackstork", "sqlite"}, mustVersion(t, "1.0.2"), lockFile.Plugins[0].Checksums).Return(&ResolvedPlugin{
		BinaryPath: "/blackstork/sqlite/1.0.2",
		Checksums: []Checksum{
			mustChecksum(t, "archive:linux:amd64:lgNgp5LO81yt1boBsiaNsJCzLWD9r5ovW+el5k/dDZ8="),
			mustChecksum(t, "binary:linux:amd64:lgNgp5LO81yt1boBsiaNsJCzLWD9r5ovW+el5k/dDZ8="),
		},
	}, nil)
	lockFile, diags = resolver.Install(context.Background(), lockFile, false)
	require.Len(t, diags, 0)
	require.Equal(t, &LockFile{
		Plugins: []PluginLock{
			{
				Name:    Name{"blackstork", "sqlite"},
				Version: mustVersion(t, "1.0.2"),
				Checksums: []Checksum{
					mustChecksum(t, "archive:linux:amd64:lgNgp5LO81yt1boBsiaNsJCzLWD9r5ovW+el5k/dDZ8="),
					mustChecksum(t, "binary:linux:amd64:lgNgp5LO81yt1boBsiaNsJCzLWD9r5ovW+el5k/dDZ8="),
				},
			},
		},
	}, lockFile)
	source.AssertExpectations(t)
}

func TestResolver_InstallError(t *testing.T) {
	source := newMockSource(t)
	resolver, diags := NewResolver(map[string]string{
		"blackstork/sqlite": ">= 1.0 < 2.0",
	}, WithSources(source))
	require.Len(t, diags, 0)
	require.NotNil(t, resolver)
	// missing plugin
	source.EXPECT().Lookup(mock.Anything, Name{"blackstork", "sqlite"}).Return([]Version{}, nil)
	lockFile, diags := resolver.Install(context.Background(), &LockFile{}, false)
	require.Len(t, diags, 1)
	require.Nil(t, lockFile)
	source.AssertExpectations(t)
}

func TestResolver_Resolve(t *testing.T) {
	source := newMockSource(t)
	resolver, diags := NewResolver(map[string]string{
		"blackstork/sqlite": ">= 1.0 < 2.0",
	}, WithSources(source))
	require.Len(t, diags, 0)
	require.NotNil(t, resolver)
	source.EXPECT().Resolve(mock.Anything, Name{"blackstork", "sqlite"}, mustVersion(t, "1.0.2"), []Checksum{
		mustChecksum(t, "archive:linux:amd64:lgNgp5LO81yt1boBsiaNsJCzLWD9r5ovW+el5k/dDZ8="),
		mustChecksum(t, "binary:linux:amd64:lgNgp5LO81yt1boBsiaNsJCzLWD9r5ovW+el5k/dDZ8="),
	}).Return(&ResolvedPlugin{
		BinaryPath: "/blackstork/sqlite@1.0.2",
		Checksums: []Checksum{
			mustChecksum(t, "archive:linux:amd64:lgNgp5LO81yt1boBsiaNsJCzLWD9r5ovW+el5k/dDZ8="),
			mustChecksum(t, "binary:linux:amd64:lgNgp5LO81yt1boBsiaNsJCzLWD9r5ovW+el5k/dDZ8="),
		},
	}, nil)
	binMap, diags := resolver.Resolve(context.Background(), &LockFile{
		Plugins: []PluginLock{
			{
				Name:    Name{"blackstork", "sqlite"},
				Version: mustVersion(t, "1.0.2"),
				Checksums: []Checksum{
					mustChecksum(t, "archive:linux:amd64:lgNgp5LO81yt1boBsiaNsJCzLWD9r5ovW+el5k/dDZ8="),
					mustChecksum(t, "binary:linux:amd64:lgNgp5LO81yt1boBsiaNsJCzLWD9r5ovW+el5k/dDZ8="),
				},
			},
		},
	})
	require.Len(t, diags, 0)
	require.Equal(t, map[string]string{
		"blackstork/sqlite": "/blackstork/sqlite@1.0.2",
	}, binMap)
	source.AssertExpectations(t)
}

func TestResolver_ResolveBadLockFile(t *testing.T) {
	source := newMockSource(t)
	resolver, diags := NewResolver(map[string]string{
		"blackstork/sqlite": ">= 1.0 < 2.0",
	}, WithSources(source))
	require.Len(t, diags, 0)
	require.NotNil(t, resolver)
	// missing plugin
	binMap, diags := resolver.Resolve(context.Background(), &LockFile{})
	require.Len(t, diags, 1)
	require.Nil(t, binMap)
	source.AssertExpectations(t)
}

func TestResolver_ResolveMissmatchLockFile(t *testing.T) {
	source := newMockSource(t)
	resolver, diags := NewResolver(map[string]string{
		"blackstork/sqlite": ">= 1.0 < 2.0",
	}, WithSources(source))
	require.Len(t, diags, 0)
	require.NotNil(t, resolver)
	binMap, diags := resolver.Resolve(context.Background(), &LockFile{
		Plugins: []PluginLock{
			{
				Name:    Name{"blackstork", "sqlite"},
				Version: mustVersion(t, "3.0.0"),
				Checksums: []Checksum{
					mustChecksum(t, "archive:linux:amd64:lgNgp5LO81yt1boBsiaNsJCzLWD9r5ovW+el5k/dDZ8="),
					mustChecksum(t, "binary:linux:amd64:lgNgp5LO81yt1boBsiaNsJCzLWD9r5ovW+el5k/dDZ8="),
				},
			},
		},
	})
	require.Len(t, diags, 1)
	require.Nil(t, binMap)
}

func TestResolver_ResolveWarn(t *testing.T) {
	source := newMockSource(t)
	resolver, diags := NewResolver(map[string]string{}, WithSources(source))
	require.Len(t, diags, 0)
	require.NotNil(t, resolver)
	binMap, diags := resolver.Resolve(context.Background(), &LockFile{
		Plugins: []PluginLock{
			{
				Name:    Name{"blackstork", "sqlite"},
				Version: mustVersion(t, "1.0.2"),
				Checksums: []Checksum{
					mustChecksum(t, "archive:linux:amd64:lgNgp5LO81yt1boBsiaNsJCzLWD9r5ovW+el5k/dDZ8="),
					mustChecksum(t, "binary:linux:amd64:lgNgp5LO81yt1boBsiaNsJCzLWD9r5ovW+el5k/dDZ8="),
				},
			},
		},
	})
	require.Len(t, diags, 1)
	require.Equal(t, hcl.DiagWarning, diags[0].Severity)
	require.Empty(t, binMap)
}

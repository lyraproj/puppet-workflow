package puppetwf

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func Test_munged_1(t *testing.T) {
	require.Equal(t, `Foo::X010::Bar`, munged(`/foo/0.1.0/bar`))
}

func Test_munged_2(t *testing.T) {
	require.Equal(t, `Foo::VX::Bar`, munged(`/foo/v::x/bar`))
}

func Test_munged_3(t *testing.T) {
	require.Equal(t, `Foo::V0::Bar`, munged(`/foo/v::0/bar`))
}

func Test_munged_4(t *testing.T) {
	require.Equal(t, `Foo::ABC::Bar`, munged(`/foo/a.b.c/bar`))
}

func Test_munged_5(t *testing.T) {
	require.Equal(t, `Foo::Abc::Bar`, munged(`/foo/abc/bar`))
}

func Test_munged_6(t *testing.T) {
	require.Equal(t, `A::B::C`, munged(`/a/b/c/`))
}

package nuget

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRenderCsproj(t *testing.T) {
	t.Parallel()
	t.Run("single_framework", func(t *testing.T) {
		t.Parallel()
		got, err := renderCsproj(
			[]string{"netstandard2.0"},
			[]nugetDep{
				{name: "Google.Protobuf", version: "3.34.1"},
			},
		)
		require.NoError(t, err)
		want := `<Project Sdk="Microsoft.NET.Sdk">
  <PropertyGroup>
    <TargetFramework>netstandard2.0</TargetFramework>
  </PropertyGroup>
  <ItemGroup>
    <PackageReference Include="Google.Protobuf" Version="3.34.1" />
  </ItemGroup>
</Project>
`
		assert.Equal(t, want, got)
	})
	t.Run("multiple_deps", func(t *testing.T) {
		t.Parallel()
		got, err := renderCsproj(
			[]string{"netstandard2.0"},
			[]nugetDep{
				{name: "Google.Protobuf", version: "3.34.1"},
				{name: "Grpc.Net.Common", version: "2.76.0"},
			},
		)
		require.NoError(t, err)
		want := `<Project Sdk="Microsoft.NET.Sdk">
  <PropertyGroup>
    <TargetFramework>netstandard2.0</TargetFramework>
  </PropertyGroup>
  <ItemGroup>
    <PackageReference Include="Google.Protobuf" Version="3.34.1" />
    <PackageReference Include="Grpc.Net.Common" Version="2.76.0" />
  </ItemGroup>
</Project>
`
		assert.Equal(t, want, got)
	})
	t.Run("multiple_frameworks", func(t *testing.T) {
		t.Parallel()
		got, err := renderCsproj(
			[]string{"netstandard2.0", "net6.0"},
			[]nugetDep{
				{name: "Google.Protobuf", version: "3.34.1"},
			},
		)
		require.NoError(t, err)
		want := `<Project Sdk="Microsoft.NET.Sdk">
  <PropertyGroup>
    <TargetFrameworks>netstandard2.0;net6.0</TargetFrameworks>
  </PropertyGroup>
  <ItemGroup>
    <PackageReference Include="Google.Protobuf" Version="3.34.1" />
  </ItemGroup>
</Project>
`
		assert.Equal(t, want, got)
	})
}

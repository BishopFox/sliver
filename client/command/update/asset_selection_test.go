package update

import (
	"testing"

	"github.com/bishopfox/sliver/client/version"
)

func TestFindWindowsExecutableAssetForArchitecture(t *testing.T) {
	for _, tc := range []struct {
		prefix    string
		arch      string
		wrongArch string
	}{
		{prefix: "sliver-client", arch: "amd64", wrongArch: "arm64"},
		{prefix: "sliver-client", arch: "arm64", wrongArch: "386"},
		{prefix: "sliver-client", arch: "386", wrongArch: "amd64"},
		{prefix: "sliver-server", arch: "amd64", wrongArch: "arm64"},
		{prefix: "sliver-server", arch: "arm64", wrongArch: "amd64"},
	} {
		t.Run(tc.prefix+"/"+tc.arch, func(t *testing.T) {
			want := tc.prefix + "_windows-" + tc.arch + ".exe"
			otherPrefix := "sliver-server"
			if tc.prefix == otherPrefix {
				otherPrefix = "sliver-client"
			}
			assets := releaseAssets(
				want+".minisig",
				otherPrefix+"_windows-"+tc.arch+".exe",
				tc.prefix+"_windows-"+tc.wrongArch+".exe",
				want,
			)

			got := findAssetFor(tc.prefix, assetSuffixes("windows", tc.arch), assets)
			if got == nil || got.Name != want {
				t.Fatalf("findAssetFor(%s, windows/%s) = %v, want %q", tc.prefix, tc.arch, got, want)
			}
		})
	}
}

func TestFindNonWindowsReleaseAssets(t *testing.T) {
	for _, tc := range []struct {
		name   string
		prefix string
		goos   string
		arch   string
		want   string
	}{
		{name: "linux zip", prefix: "sliver-client", goos: "linux", arch: "amd64", want: "sliver-client_linux-amd64.zip"},
		{name: "linux extensionless", prefix: "sliver-server", goos: "linux", arch: "arm64", want: "sliver-server_linux-arm64"},
		{name: "darwin zip", prefix: "sliver-client", goos: "darwin", arch: "arm64", want: "sliver-client_macos-arm64.zip"},
		{name: "darwin extensionless", prefix: "sliver-server", goos: "darwin", arch: "amd64", want: "sliver-server_darwin-amd64"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			assets := releaseAssets(tc.want+".minisig", tc.want)
			got := findAssetFor(tc.prefix, assetSuffixes(tc.goos, tc.arch), assets)
			if got == nil || got.Name != tc.want {
				t.Fatalf("findAssetFor(%s, %s/%s) = %v, want %q", tc.prefix, tc.goos, tc.arch, got, tc.want)
			}
		})
	}
}

func releaseAssets(names ...string) []version.Asset {
	assets := make([]version.Asset, 0, len(names))
	for _, name := range names {
		assets = append(assets, version.Asset{
			Name:               name,
			BrowserDownloadURL: "https://example.invalid/releases/v1/" + name,
		})
	}
	return assets
}

package assets

import (
	"runtime"
	"strings"
	"testing"

	ver "github.com/bishopfox/sliver/server/version"
	utilAssets "github.com/bishopfox/sliver/util/assets"
)

func TestExpectedAssetVersionIncludesGarbleManifest(t *testing.T) {
	digest, ok := utilAssets.ExpectedGarbleSHA256(runtime.GOOS, runtime.GOARCH)
	if !ok {
		t.Skipf("no pinned Garble artifact for %s/%s", runtime.GOOS, runtime.GOARCH)
	}

	version := expectedAssetVersion()
	if !strings.HasPrefix(version, ver.GitCommit+":garble-") {
		t.Fatalf("expectedAssetVersion() = %q, want Git commit and Garble marker", version)
	}
	if !strings.HasSuffix(version, digest) {
		t.Fatalf("expectedAssetVersion() = %q, want pinned digest %q", version, digest)
	}
}

# Server 
./go-tests.sh CGO_ENABLED=1 go --tags osusergo,netgo,sqlite_omit_load_extension,server -ldflags "-s -w -X github.com/bishopfox/sliver/client/version.Version=v1.4.6 -X \"github.com/bishopfox/sliver/client/version.GoVersion=go version go1.16.2 linux/amd64\" -X github.com/bishopfox/sliver/client/version.CompiledAt=1616711046 -X github.com/bishopfox/sliver/client/version.GithubReleasesURL=https://api.github.com/repos/BishopFox/sliver/releases -X github.com/bishopfox/sliver/client/version.GitCommit=8a170119660642fef3ff542861bae8b5f2e5e5be -X github.com/bishopfox/sliver/client/version.GitDirty=Dirty"
Testing with build command:
-------------------------------------------------------
CGO_ENABLED=1 go test --tags osusergo,netgo,sqlite_omit_load_extension,server -ldflags "-s -w -X github.com/bishopfox/sliver/client/version.Version=v1.4.6 -X "github.com/bishopfox/sliver/client/version.GoVersion=go version go1.16.2 linux/amd64" -X github.com/bishopfox/sliver/client/version.CompiledAt=1616711046 -X github.com/bishopfox/sliver/client/version.GithubReleasesURL=https://api.github.com/repos/BishopFox/sliver/releases -X github.com/bishopfox/sliver/client/version.GitCommit=8a170119660642fef3ff542861bae8b5f2e5e5be -X github.com/bishopfox/sliver/client/version.GitDirty=Dirty"
ok  	github.com/bishopfox/sliver/util	(cached) [no tests to run]
ok  	github.com/bishopfox/sliver/util/encoders	(cached)
--- FAIL: TestRemoveContent (0.01s)
    website_test.go:171: row value misused
    website_test.go:175: Expected ErrRecordNotFound, but got open /home/user/.sliver/web/f08e4b1a-3a51-4cb1-bbba-83f5dc239d26: no such file or directory
FAIL
FAIL	github.com/bishopfox/sliver/server/website	0.041s
FAIL
-s -w -X github.com/bishopfox/sliver/client/version.Version=v1.4.6 -X "github.com/bishopfox/sliver/client/version.GoVersion=go version go1.16.2 linux/amd64" -X github.com/bishopfox/sliver/client/version.CompiledAt=1616711046 -X github.com/bishopfox/sliver/client/version.GithubReleasesURL=https://api.github.com/repos/BishopFox/sliver/releases -X github.com/bishopfox/sliver/client/version.GitCommit=8a170119660642fef3ff542861bae8b5f2e5e5be -X github.com/bishopfox/sliver/client/version.GitDirty=Dirty

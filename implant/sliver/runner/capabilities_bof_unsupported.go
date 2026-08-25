//go:build !sliver_lint && !((darwin && (amd64 || arm64)) || (linux && (386 || amd64 || arm64)) || (windows && (386 || amd64 || arm64)))

package runner

func implantCapabilities() uint64 {
	return 0
}

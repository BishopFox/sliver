package assets

import (
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type runner struct {
	logger      *logger
	httpClient  *http.Client
	outputDir   string
	workDir     string
	goIndex     int
	zigIndex    int
	garbleIndex int
	zigMirrors  []string
}

type config struct {
	verbose bool
}

// Run executes the asset generation flow.
func Run(args []string) error {
	cfg, showHelp, err := parseArgs(args)
	if err != nil {
		return err
	}
	if showHelp {
		fmt.Println(usage())
		return nil
	}

	repoRoot, err := findRepoRoot()
	if err != nil {
		return err
	}
	outputDir := filepath.Join(repoRoot, "server", "assets", "fs")
	if err := os.MkdirAll(outputDir, 0o755); err != nil {
		return fmt.Errorf("create output dir: %w", err)
	}

	workDir, err := os.MkdirTemp("", "sliver-assets-")
	if err != nil {
		return fmt.Errorf("create work dir: %w", err)
	}

	log := newLogger(cfg.verbose)
	r := &runner{
		logger:     log,
		httpClient: &http.Client{Timeout: 15 * time.Minute},
		outputDir:  outputDir,
		workDir:    workDir,
	}

	log.Logf("-----------------------------------------------------------------")
	log.Logf("%s (Output: %s)", workDir, outputDir)
	log.Logf("-----------------------------------------------------------------")

	defer func() {
		log.Logf("clean up: %s", workDir)
		_ = os.RemoveAll(workDir)
	}()

	if err := r.buildGoAssets(); err != nil {
		return err
	}
	if err := r.buildZigAssets(); err != nil {
		return err
	}
	if err := r.buildGarbleAssets(); err != nil {
		return err
	}

	log.Logf("")
	log.Logf("[*] All done")

	return nil
}

func parseArgs(args []string) (config, bool, error) {
	cfg := config{}
	showHelp := false

	for _, arg := range args {
		switch arg {
		case "-v", "--verbose":
			cfg.verbose = true
		case "-h", "--help":
			showHelp = true
		default:
			return config{}, false, fmt.Errorf("unknown argument: %s", arg)
		}
	}

	return cfg, showHelp, nil
}

func usage() string {
	return "Usage: assets [--verbose]"
}

func findRepoRoot() (string, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("get working directory: %w", err)
	}

	dir := cwd
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", errors.New("unable to locate go.mod from current directory")
		}
		dir = parent
	}
}

func ensureDir(path string) error {
	if path == "" {
		return errors.New("empty directory path")
	}
	return os.MkdirAll(path, 0o755)
}

func trimLeadingDot(path string) string {
	return strings.TrimPrefix(path, "./")
}

package cli

/*
	Sliver Implant Framework
	Copyright (C) 2022  Bishop Fox

	This program is free software: you can redistribute it and/or modify
	it under the terms of the GNU General Public License as published by
	the Free Software Foundation, either version 3 of the License, or
	(at your option) any later version.

	This program is distributed in the hope that it will be useful,
	but WITHOUT ANY WARRANTY; without even the implied warranty of
	MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
	GNU General Public License for more details.

	You should have received a copy of the GNU General Public License
	along with this program.  If not, see <https://www.gnu.org/licenses/>.
*/

import (
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"runtime"
	"runtime/debug"
	"strings"
	"sync"
	"syscall"

	clientAssets "github.com/bishopfox/sliver/client/assets"
	"github.com/bishopfox/sliver/client/transport"
	"github.com/bishopfox/sliver/client/version"
	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/bishopfox/sliver/server/assets"
	"github.com/bishopfox/sliver/server/builder"
	"github.com/bishopfox/sliver/server/c2"
	"github.com/bishopfox/sliver/server/generate"
	"github.com/bishopfox/sliver/server/log"
	"github.com/spf13/cobra"
)

var (
	builderLog = log.NamedLogger("cli", "builder")
)

const (
	enableTargetFlagStr  = "enable-target"
	disableTargetFlagStr = "disable-target"

	operatorConfigFlagStr    = "config"
	operatorConfigDirFlagStr = "config-dir"
	quietFlagStr             = "quiet"
	logLevelFlagStr          = "log-level"
)

func initBuilderCmd() *cobra.Command {
	builderCmd.Flags().StringP(nameFlagStr, "n", "", "Name of the builder (blank = hostname)")
	builderCmd.Flags().IntP(logLevelFlagStr, "L", 4, "Logging level: 1/fatal, 2/error, 3/warn, 4/info, 5/debug, 6/trace")
	builderCmd.Flags().StringP(operatorConfigFlagStr, "c", "", "operator config file path")
	builderCmd.Flags().StringP(operatorConfigDirFlagStr, "d", "", "operator config directory path")
	builderCmd.Flags().BoolP(quietFlagStr, "q", false, "do not write any content to stdout")

	// Artifact configuration options
	builderCmd.Flags().StringSlice(enableTargetFlagStr, []string{}, "force enable a target: format:goos/goarch")
	builderCmd.Flags().StringSlice(disableTargetFlagStr, []string{}, "force disable target arch: format:goos/goarch")

	return builderCmd
}

var builders []*builder.Builder

var builderCmd = &cobra.Command{
	Use:   "builder",
	Short: "Start the process as an external builder",
	Long:  ``,
	Run: func(cmd *cobra.Command, args []string) {
		configPath, err := cmd.Flags().GetString(operatorConfigFlagStr)
		if err != nil {
			builderLog.Errorf("Failed to parse --%s flag %s\n", operatorConfigFlagStr, err)
			return
		}

		configDir, err := cmd.Flags().GetString(operatorConfigDirFlagStr)
		if err != nil {
			builderLog.Errorf("Failed to parse --%s flag %s\n", operatorConfigDirFlagStr, err)
			return
		}
		if configPath == "" && configDir == "" {
			builderLog.Errorf("Missing --%s or --%s flags\n", operatorConfigFlagStr, operatorConfigDirFlagStr)
			return
		}

		quiet, err := cmd.Flags().GetBool(quietFlagStr)
		if err != nil {
			builderLog.Errorf("Failed to parse --%s flag %s\n", quietFlagStr, err)
		}
		if !quiet {
			log.RootLogger.AddHook(log.NewStdoutHook(log.RootLoggerName))
		}
		builderLog.Infof("Initializing Sliver external builder - %s", version.FullVersion())

		level, err := cmd.Flags().GetInt(logLevelFlagStr)
		if err != nil {
			builderLog.Errorf("Failed to parse --%s flag %s\n", logLevelFlagStr, err)
			return
		}
		log.RootLogger.SetLevel(log.LevelFrom(level))

		defer func() {
			if r := recover(); r != nil {
				builderLog.Printf("panic:\n%s", debug.Stack())
				builderLog.Fatalf("stacktrace from panic: \n" + string(debug.Stack()))
				os.Exit(99)
			}
		}()

		// setup assets
		assets.Setup(true, false)
		// setup default profiles for HTTP C2
		c2.SetupDefaultC2Profiles()
		config := configPath
		multipleBuilders := (configPath == "" && configDir != "")
		if multipleBuilders {
			config = configDir
		}
		startBuilders(cmd, config, multipleBuilders)
		// Handle SIGHUP to reload builders
		sigHup := make(chan os.Signal, 1)
		signal.Notify(sigHup, syscall.SIGHUP)
		// Handle interupt to stop all builders and exit
		sigInt := make(chan os.Signal, 1)
		signal.Notify(sigInt, os.Interrupt)
		for {
			select {
			case <-sigHup:
				builderLog.Info("Received SIGHUP, reloading builders")
				reloadBuilders(cmd, config, multipleBuilders)
			case <-sigInt:
				builderLog.Info("Received SIGINT, stopping all builders")
				for _, builderInst := range builders {
					builderInst.Stop()
				}
				return
			}
		}
	},
}

// Start all builders if multpile is true or a single builder otherwise.
func startBuilders(cmd *cobra.Command, config string, multpile bool) {
	// We're passing a mutex to each builder to prevent concurrent builds.
	// Concurrent build should be fine in theory, but may cause resource
	// exhaustion on the server.
	// For single builders, this should have no impact.
	mutex := &sync.Mutex{}
	// Single builder
	if !multpile {
		singleBuilder, err := createBuilder(cmd, config, mutex)
		if err != nil {
			builderLog.Errorf("Failed to create builder: %s", err)
			os.Exit(-1)
		}
		// Start single builder (blocking call)
		singleBuilder.Start()
	} else {
		// Multiple builders
		builderLog.Infof("Reading config dir: %s", config)
		err := filepath.Walk(config, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				builderLog.Errorf("Failed to walk config dir: %s", err)
				return err
			}
			if info.IsDir() {
				return nil
			}
			go func() {
				builderLog.Infof("Starting builder with config file: %s", path)
				builderInst, err := createBuilder(cmd, path, mutex)
				if err != nil {
					builderLog.Errorf("Failed to create builder: %s", err)
					return
				}
				builders = append(builders, builderInst)
				builderInst.Start()
			}()
			return nil
		})
		if err != nil {
			builderLog.Errorf("Failed to walk config dir: %s", err)
			return
		}
	}
}

func reloadBuilders(cmd *cobra.Command, config string, multiple bool) {
	builderLog.Infof("Reloading builders")
	for _, builderInst := range builders {
		builderInst.Stop()
	}
	builders = nil
	startBuilders(cmd, config, multiple)
}

func createBuilder(cmd *cobra.Command, configPath string, mutex *sync.Mutex) (*builder.Builder, error) {
	externalBuilder := parseBuilderConfigFlags(cmd)
	externalBuilder.Templates = []string{"sliver"}

	// load the client configuration from the filesystem
	config, err := clientAssets.ReadConfig(configPath)
	if err != nil {
		builderLog.Fatalf("Invalid config file: %s", err)
		return nil, err
	}
	if externalBuilder.Name == "" {
		builderLog.Infof("No builder name was specified, attempting to use hostname")
		externalBuilder.Name, err = os.Hostname()
		if err != nil {
			builderLog.Errorf("Failed to get hostname: %s", err)
			externalBuilder.Name = fmt.Sprintf("%s's %s builder", config.Operator, runtime.GOOS)
		}
	}
	builderLog.Infof("Hello my name is: %s", externalBuilder.Name)

	// connect to the server
	builderLog.Infof("Connecting to %s@%s:%d ...", config.Operator, config.LHost, config.LPort)
	rpc, ln, err := transport.MTLSConnect(config)
	if err != nil {
		builderLog.Errorf("Failed to connect to server %s@%s:%d: %s", config.Operator, config.LHost, config.LPort, err)
		return nil, err
	}

	return builder.NewBuilder(externalBuilder, mutex, rpc, ln), nil
}

func parseBuilderConfigFlags(cmd *cobra.Command) *clientpb.Builder {
	externalBuilder := &clientpb.Builder{GOOS: runtime.GOOS, GOARCH: runtime.GOARCH}

	externalBuilder.CrossCompilers = generate.GetCrossCompilers()
	builderLog.Infof("Found %d cross-compilers", len(externalBuilder.CrossCompilers))
	for _, crossCompiler := range externalBuilder.CrossCompilers {
		builderLog.Debugf("Found cross-compiler: cc = '%s' cxx = '%s'", crossCompiler.GetCCPath(), crossCompiler.GetCXXPath())
	}

	externalBuilder.Targets = generate.GetCompilerTargets()
	builderLog.Infof("This machine has %d compiler targets", len(externalBuilder.Targets))
	for _, target := range externalBuilder.Targets {
		builderLog.Infof("[compiler target] %v", target)
	}

	parseForceEnableTargets(cmd, externalBuilder)
	parseForceDisableTargets(cmd, externalBuilder)

	name, err := cmd.Flags().GetString(nameFlagStr)
	if err != nil {
		builderLog.Errorf("Failed to parse --%s flag %s\n", nameFlagStr, err)
	}
	if name != "" {
		externalBuilder.Name = name
	}

	return externalBuilder
}

func parseForceEnableTargets(cmd *cobra.Command, externalBuilder *clientpb.Builder) {
	enableTargets, err := cmd.Flags().GetStringSlice(enableTargetFlagStr)
	if err != nil {
		builderLog.Errorf("Failed to parse --%s flag %s\n", enableTargetFlagStr, err)
		return
	}

	for _, target := range enableTargets {
		parts1 := strings.Split(target, ":")
		if len(parts1) != 2 {
			builderLog.Errorf("Invalid target format: %s", target)
			continue
		}
		parts2 := strings.Split(parts1[1], "/")
		if len(parts2) != 2 {
			builderLog.Errorf("Invalid target format: %s", target)
			continue
		}
		format := parts1[0]
		goos := parts2[0]
		goarch := parts2[1]
		target := &clientpb.CompilerTarget{
			GOOS:   goos,
			GOARCH: goarch,
		}
		switch strings.ToLower(format) {

		case "executable", "exe", "exec", "pe":
			target.Format = clientpb.OutputFormat_EXECUTABLE

		case "shared-lib", "sharedlib", "dll", "so", "dylib":
			target.Format = clientpb.OutputFormat_SHARED_LIB

		case "service", "svc":
			target.Format = clientpb.OutputFormat_SERVICE

		case "shellcode", "shell", "sc":
			target.Format = clientpb.OutputFormat_SHELLCODE

		default:
			builderLog.Warnf("Invalid format '%s' defaulting to executable", format)
			target.Format = clientpb.OutputFormat_EXECUTABLE
		}

		builderLog.Infof("Force enable target %s:%s/%s", target.Format, goos, goarch)
		externalBuilder.Targets = append(externalBuilder.Targets, target)
	}
}

func parseForceDisableTargets(cmd *cobra.Command, externalBuilder *clientpb.Builder) {
	disableTargets, err := cmd.Flags().GetStringSlice(disableTargetFlagStr)
	if err != nil {
		builderLog.Errorf("Failed to parse --%s flag %s\n", disableTargetFlagStr, err)
		return
	}

	for _, target := range disableTargets {
		parts1 := strings.Split(target, ":")
		if len(parts1) != 2 {
			builderLog.Errorf("Invalid target format: %s", target)
			continue
		}
		parts2 := strings.Split(parts1[1], "/")
		if len(parts2) != 2 {
			builderLog.Errorf("Invalid target format: %s", target)
			continue
		}

		var format clientpb.OutputFormat
		switch strings.ToLower(parts1[0]) {

		case "executable", "exe", "exec", "pe":
			format = clientpb.OutputFormat_EXECUTABLE

		case "shared-lib", "sharedlib", "dll", "so", "dylib":
			format = clientpb.OutputFormat_SHARED_LIB

		case "service", "svc":
			format = clientpb.OutputFormat_SERVICE

		case "shellcode", "shell", "sc":
			format = clientpb.OutputFormat_SHELLCODE

		default:
			builderLog.Warnf("Invalid format '%s' defaulting to executable", parts1[0])
			format = clientpb.OutputFormat_EXECUTABLE
		}

		goos := parts2[0]
		goarch := parts2[1]

		builderLog.Infof("Force disable target %s:%s/%s", format, goos, goarch)
		for i, t := range externalBuilder.Targets {
			if t.GOOS == goos && t.GOARCH == goarch && t.Format == format {
				externalBuilder.Targets = append(externalBuilder.Targets[:i], externalBuilder.Targets[i+1:]...)
				break
			}
		}
	}
}

package watchtower

import (
	"errors"
	"fmt"

	consts "github.com/bishopfox/sliver/client/constants"
	"github.com/bishopfox/sliver/server/configs"
	"github.com/bishopfox/sliver/server/core"
	"github.com/bishopfox/sliver/server/db"
	"github.com/bishopfox/sliver/server/db/models"
	"github.com/bishopfox/sliver/server/log"
	"github.com/lesnuages/snitch/pkg/snitch"
)

var (
	initialized   = false
	watcher       *snitch.Snitch
	watchtowerLog = log.NamedLogger("watchtower", "samples")
)

func update(implantBuild *models.ImplantBuild) {
	if watcher != nil && initialized {
		watchtowerLog.Debugf("Monitoring implant %s (%s)", implantBuild.Name, implantBuild.MD5)
		watcher.Add(implantBuild.Name, implantBuild.MD5)
		watchtowerLog.Debugf("Implant added to the watch list")
	}
}

func handleBurnedImplant(result *snitch.ScanResult) {
	build, err := db.ImplantBuildByName(result.Sample.Name())
	if build != nil && err == nil {
		build.Burned = true
		db.Session().Save(build)
	}
	for _, session := range core.Sessions.All() {
		// Won't work for sessions that have been renamed
		if session.Name == result.Sample.Name() {
			core.EventBroker.Publish(core.Event{
				Session:   session,
				EventType: consts.WatchtowerEvent,
				Data:      []byte(fmt.Sprintf("%s - %v", result.Provider, result.LastSeen)),
			})
		}
	}
}

func addExistingImplants() error {
	builds, err := db.ImplantBuilds()
	if err != nil {
		return err
	}
	for _, build := range builds {
		if !build.Burned {
			update(build)
		}
	}
	return nil
}

func StartWatchTower(config *configs.ServerConfig) error {
	var scanners []snitch.Scanner
	if watcher != nil {
		return errors.New("monitoring already started")
	}
	if config.Watchtower == nil {
		return errors.New("no provider info")
	}
	if config.Watchtower.VTApiKey != "" {
		scanners = append(scanners, snitch.NewVTScanner(config.Watchtower.VTApiKey, snitch.VTMaxRequests, "Virus Total"))
	}
	if config.Watchtower.XForceApiKey != "" && config.Watchtower.XForceApiPassword != "" {
		scanners = append(scanners, snitch.NewXForceScanner(config.Watchtower.XForceApiKey, config.Watchtower.XForceApiPassword, snitch.XForceMaxRequests, "IBM X-Force"))
	}
	if len(scanners) == 0 {
		return errors.New("missing provider credentials")
	}
	watcher = snitch.WithHandleFlagged(handleBurnedImplant)
	// Add providers
	for _, s := range scanners {
		watcher.AddScanner(s)
	}
	// Start the loop
	watcher.Start()
	initialized = true
	err := addExistingImplants()
	if err != nil {
		return err
	}
	return nil
}

func AddImplantToWatchlist(implant *models.ImplantBuild) {
	update(implant)
}

func StopWatchTower() {
	if watcher != nil {
		watcher.Stop()
	}
	initialized = false
	watcher = nil
}

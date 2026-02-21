package cli

import (
	"errors"
	"fmt"
	"log"
	"os"
	"runtime/debug"
	"strings"

	"github.com/bishopfox/sliver/client/constants"
	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/bishopfox/sliver/server/assets"
	"github.com/bishopfox/sliver/server/c2"
	"github.com/bishopfox/sliver/server/certs"
	"github.com/bishopfox/sliver/server/configs"
	"github.com/bishopfox/sliver/server/console"
	"github.com/bishopfox/sliver/server/cryptography"
	"github.com/bishopfox/sliver/server/daemon"
	"github.com/bishopfox/sliver/server/db"
	"github.com/spf13/cobra"
)

var daemonCmd = &cobra.Command{
	Use:   "daemon",
	Short: "Force start server in daemon mode",
	Long:  ``,
	Run: func(cmd *cobra.Command, args []string) {
		force, err := cmd.Flags().GetBool(forceFlagStr)
		if err != nil {
			fmt.Printf("Failed to parse --%s flag %s\n", forceFlagStr, err)
			return
		}
		lhost, err := cmd.Flags().GetString(lhostFlagStr)
		if err != nil {
			fmt.Printf("Failed to parse --%s flag %s\n", lhostFlagStr, err)
			return
		}
		lport, err := cmd.Flags().GetUint16(lportFlagStr)
		if err != nil {
			fmt.Printf("Failed to parse --%s flag %s\n", lportFlagStr, err)
			return
		}

		tailscale, err := cmd.Flags().GetBool(tailscaleFlagStr)
		if err != nil {
			fmt.Printf("Failed to parse --%s flag %s\n", tailscaleFlagStr, err)
			return
		}

		appDir := assets.GetRootAppDir()
		logFile := initConsoleLogging(appDir)
		defer logFile.Close()

		defer func() {
			if r := recover(); r != nil {
				log.Printf("panic:\n%s", debug.Stack())
				fmt.Println("stacktrace from panic: \n" + string(debug.Stack()))
				os.Exit(99)
			}
		}()

		assets.Setup(force, false)
		c2.SetupDefaultC2Profiles()
		certs.SetupCAs()
		certs.SetupWGKeys()
		cryptography.AgeServerKeyPair()
		cryptography.MinisignServerPrivateKey()
		_, _ = configs.LoadCrackConfig()

		listenerJobs, err := db.ListenerJobs()
		if err != nil {
			fmt.Printf("[!] Failed to load persistent listener jobs: %s\n", err)
		}

		err = StartPersistentJobs(listenerJobs)
		if err != nil {
			fmt.Printf("[!] %s\n", err)
		}

		daemon.Start(lhost, uint16(lport), tailscale)
	},
}

func StartPersistentJobs(listenerJobs []*clientpb.ListenerJob) error {
	if len(listenerJobs) == 0 {
		return nil
	}

	var startupErrs []error
	for _, j := range listenerJobs {
		if j == nil {
			startupErrs = append(startupErrs, errors.New("persistent startup entry is nil"))
			continue
		}

		listenerJob, err := db.ListenerByJobID(j.JobID)
		if err != nil {
			startupErrs = append(startupErrs, fmt.Errorf("persistent %s listener (saved job id=%d): failed to load config: %w", j.Type, j.JobID, err))
			continue
		}

		startupContext := persistentJobStartupContext(j.Type, j.JobID, listenerJob)
		savedJobID := j.JobID
		jobID, err := startPersistentListenerJob(j.Type, listenerJob)
		if err != nil {
			startupErrs = append(startupErrs, fmt.Errorf("%s: failed to start: %w", startupContext, err))
			continue
		}

		j.JobID = jobID
		if err := db.UpdateListenerJobID(savedJobID, jobID); err != nil {
			startupErrs = append(startupErrs, fmt.Errorf("%s: started with new job id=%d but failed to update listener record: %w", startupContext, j.JobID, err))
		}
	}

	if len(startupErrs) == 0 {
		return nil
	}
	return fmt.Errorf("persistent listener startup errors (%d/%d failed): %w", len(startupErrs), len(listenerJobs), errors.Join(startupErrs...))
}

func startPersistentListenerJob(jobType string, listenerJob *clientpb.ListenerJob) (uint32, error) {
	switch jobType {
	case constants.HttpStr, constants.HttpsStr:
		if listenerJob.HTTPConf == nil {
			return 0, errors.New("missing HTTP listener configuration")
		}
		job, err := c2.StartHTTPListenerJob(listenerJob.HTTPConf)
		if err != nil {
			return 0, err
		}
		return uint32(job.ID), nil
	case constants.MtlsStr:
		if listenerJob.MTLSConf == nil {
			return 0, errors.New("missing mTLS listener configuration")
		}
		job, err := c2.StartMTLSListenerJob(listenerJob.MTLSConf)
		if err != nil {
			return 0, err
		}
		return uint32(job.ID), nil
	case constants.WGStr:
		if listenerJob.WGConf == nil {
			return 0, errors.New("missing WireGuard listener configuration")
		}
		job, err := c2.StartWGListenerJob(listenerJob.WGConf)
		if err != nil {
			return 0, err
		}
		return uint32(job.ID), nil
	case constants.DnsStr:
		if listenerJob.DNSConf == nil {
			return 0, errors.New("missing DNS listener configuration")
		}
		job, err := c2.StartDNSListenerJob(listenerJob.DNSConf)
		if err != nil {
			return 0, err
		}
		return uint32(job.ID), nil
	case constants.MultiplayerModeStr:
		if listenerJob.MultiConf == nil {
			return 0, errors.New("missing multiplayer listener configuration")
		}
		id, err := console.JobStartClientListener(listenerJob.MultiConf)
		if err != nil {
			return 0, err
		}
		return uint32(id), nil
	case constants.TCPListenerStr:
		if listenerJob.TCPConf == nil {
			return 0, errors.New("missing TCP stager listener configuration")
		}
		job, err := c2.StartTCPStagerListenerJob(listenerJob.TCPConf.Host, uint16(listenerJob.TCPConf.Port), listenerJob.TCPConf.ProfileName, listenerJob.TCPConf.Data)
		if err != nil {
			return 0, err
		}
		return uint32(job.ID), nil
	default:
		return 0, fmt.Errorf("unsupported listener type %q", jobType)
	}
}

func persistentJobStartupContext(jobType string, savedJobID uint32, listenerJob *clientpb.ListenerJob) string {
	prefix := fmt.Sprintf("persistent %s listener (saved job id=%d)", jobType, savedJobID)
	if listenerJob == nil {
		return prefix
	}

	switch jobType {
	case constants.HttpStr, constants.HttpsStr:
		if listenerJob.HTTPConf == nil {
			return prefix
		}
		return fmt.Sprintf("%s [%s domain=%q]", prefix, listenerBind(listenerJob.HTTPConf.Host, listenerJob.HTTPConf.Port), listenerJob.HTTPConf.Domain)
	case constants.MtlsStr:
		if listenerJob.MTLSConf == nil {
			return prefix
		}
		return fmt.Sprintf("%s [%s]", prefix, listenerBind(listenerJob.MTLSConf.Host, listenerJob.MTLSConf.Port))
	case constants.WGStr:
		if listenerJob.WGConf == nil {
			return prefix
		}
		return fmt.Sprintf("%s [wg_port=%d netstack_port=%d key_port=%d]", prefix, listenerJob.WGConf.Port, listenerJob.WGConf.NPort, listenerJob.WGConf.KeyPort)
	case constants.DnsStr:
		if listenerJob.DNSConf == nil {
			return prefix
		}
		return fmt.Sprintf("%s [%s domains=%s]", prefix, listenerBind(listenerJob.DNSConf.Host, listenerJob.DNSConf.Port), strings.Join(listenerJob.DNSConf.Domains, ","))
	case constants.MultiplayerModeStr:
		if listenerJob.MultiConf == nil {
			return prefix
		}
		return fmt.Sprintf("%s [%s]", prefix, listenerBind(listenerJob.MultiConf.Host, listenerJob.MultiConf.Port))
	case constants.TCPListenerStr:
		if listenerJob.TCPConf == nil {
			return prefix
		}
		return fmt.Sprintf("%s [%s profile=%q]", prefix, listenerBind(listenerJob.TCPConf.Host, listenerJob.TCPConf.Port), listenerJob.TCPConf.ProfileName)
	default:
		return prefix
	}
}

func listenerBind(host string, port uint32) string {
	host = strings.TrimSpace(host)
	if host == "" {
		host = "*"
	}
	return fmt.Sprintf("%s:%d", host, port)
}

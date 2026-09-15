package opfor

// aggressorIntegrationConfig contains the importer integrations and default
// policies which are intentionally shared with ScriptLoader child runtimes.
// Keeping them in one value makes that inheritance structural: adding a new
// integration here cannot silently omit it from the child profile clone.
type aggressorIntegrationConfig struct {
	beaconEncoder                     BeaconStringEncoder
	bofPackByteOrder                  BOFPackByteOrder
	eventDispatcher                   AggressorEventDispatcher
	aggressorBeaconTranscriptSink     AggressorBeaconTranscriptSink
	aggressorBeaconActionProvider     AggressorBeaconActionProvider
	aggressorBeaconExecutionProvider  AggressorBeaconExecutionProvider
	aggressorBOFExtractor             AggressorBOFExtractor
	aggressorArtifactProvider         AggressorArtifactProvider
	aggressorPayloadProvider          AggressorPayloadProvider
	aggressorListenerProvider         AggressorListenerProvider
	aggressorPayloadStoreProvider     AggressorPayloadStoreProvider
	aggressorSiteProvider             AggressorSiteProvider
	aggressorTeamServerRPCProvider    AggressorTeamServerRPCProvider
	aggressorSessionQueryProvider     AggressorSessionQueryProvider
	aggressorDataModelQueryProvider   AggressorDataModelQueryProvider
	aggressorDataStoreProvider        AggressorDataStoreProvider
	aggressorPreferenceProvider       AggressorPreferenceProvider
	aggressorCodeTransformProvider    AggressorCodeTransformProvider
	aggressorProcessInjectionProvider AggressorProcessInjectionProvider
	aggressorProfileProvider          AggressorProfileProvider
	aggressorVPNProvider              AggressorVPNProvider
	aggressorPEProvider               AggressorPEProvider
	aggressorClientServiceProvider    AggressorClientServiceProvider
	aggressorClientUIProvider         AggressorClientUIProvider
	aggressorDialogProvider           AggressorDialogProvider
	aggressorPromptProvider           AggressorPromptProvider
	aggressorBreakpointProvider       AggressorBreakpointProvider
}

// aggressorConfig groups every construction-time capability owned by the
// Aggressor compatibility layer. It is embedded in runtimeConfig for now so
// existing option constructors keep their narrow field assignments while the
// layer can be moved behind a profile boundary independently of the Sleep
// runtime configuration.
type aggressorConfig struct {
	aggressorIntegrationConfig
	aggressorCommands         map[AggressorCommandKind]AggressorCommandCatalog
	aggressorBeaconTechniques map[AggressorBeaconTechniqueKind]AggressorBeaconTechniqueCatalog
}

func defaultAggressorConfig() aggressorConfig {
	return aggressorConfig{
		aggressorIntegrationConfig: aggressorIntegrationConfig{
			beaconEncoder:    utf8BeaconStringEncoder{},
			bofPackByteOrder: BOFPackBigEndian,
			eventDispatcher:  synchronousAggressorEventDispatcher{},
		},
	}
}

// aggressorState is the runtime-owned counterpart to aggressorConfig. Keeping
// provider references, catalogs, UI identifiers, and dispatch policy together
// gives child-runtime cloning and future profile installation one authoritative
// state boundary without changing current promoted-field behavior.
type aggressorState struct {
	aggressorIntegrationConfig
	nextAggressorDialog       AggressorDialogID
	nextAggressorPrompt       AggressorPromptID
	aggressorCommands         *aggressorCommandState
	aggressorBeaconTechniques *aggressorBeaconTechniqueState
}

func newAggressorState(config aggressorConfig) aggressorState {
	return aggressorState{
		aggressorIntegrationConfig: config.aggressorIntegrationConfig,
		aggressorCommands:          newAggressorCommandState(config.aggressorCommands),
		aggressorBeaconTechniques:  newAggressorBeaconTechniqueState(config.aggressorBeaconTechniques),
	}
}

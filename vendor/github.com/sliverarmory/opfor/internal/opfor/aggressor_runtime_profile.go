package opfor

// aggressorRuntimeExtensionProfile recreates the construction-time Aggressor
// integrations and importer-owned base catalogs in ScriptLoader children.
// Provider instances are deliberately shared; all runtime-owned registries,
// callback layers, UI identifiers, and script overlays are rebuilt fresh.
type aggressorRuntimeExtensionProfile struct{}

func (aggressorRuntimeExtensionProfile) cloneForScriptLoader(parent *Runtime) Option {
	snapshot := parent.aggressorState.scriptLoaderConfig()
	return func(config *runtimeConfig) error {
		config.aggressorConfig = cloneAggressorConfig(snapshot)
		return nil
	}
}

func (state *aggressorState) scriptLoaderConfig() aggressorConfig {
	if state == nil {
		return aggressorConfig{}
	}
	config := aggressorConfig{
		aggressorIntegrationConfig: state.aggressorIntegrationConfig,
	}
	if state.aggressorCommands != nil {
		for _, kind := range []AggressorCommandKind{AggressorCommandBeacon, AggressorCommandSSH} {
			catalog := state.aggressorCommands.baseSnapshot(kind)
			if len(catalog.Commands) == 0 && len(catalog.Groups) == 0 {
				continue
			}
			if config.aggressorCommands == nil {
				config.aggressorCommands = make(map[AggressorCommandKind]AggressorCommandCatalog, 2)
			}
			config.aggressorCommands[kind] = catalog
		}
	}
	if state.aggressorBeaconTechniques != nil {
		for _, kind := range aggressorBeaconTechniqueKinds {
			catalog := state.aggressorBeaconTechniques.baseSnapshot(kind)
			if len(catalog.Techniques) == 0 {
				continue
			}
			if config.aggressorBeaconTechniques == nil {
				config.aggressorBeaconTechniques = make(map[AggressorBeaconTechniqueKind]AggressorBeaconTechniqueCatalog, len(aggressorBeaconTechniqueKinds))
			}
			config.aggressorBeaconTechniques[kind] = catalog
		}
	}
	return config
}

// cloneAggressorConfig preserves provider identity while detaching the catalog
// maps and slices. The extra copy makes a captured profile Option safe even if
// it is applied more than once internally.
func cloneAggressorConfig(config aggressorConfig) aggressorConfig {
	clone := aggressorConfig{
		aggressorIntegrationConfig: config.aggressorIntegrationConfig,
	}
	if len(config.aggressorCommands) != 0 {
		clone.aggressorCommands = make(map[AggressorCommandKind]AggressorCommandCatalog, len(config.aggressorCommands))
		for kind, catalog := range config.aggressorCommands {
			clone.aggressorCommands[kind] = cloneAggressorCommandCatalog(catalog)
		}
	}
	if len(config.aggressorBeaconTechniques) != 0 {
		clone.aggressorBeaconTechniques = make(map[AggressorBeaconTechniqueKind]AggressorBeaconTechniqueCatalog, len(config.aggressorBeaconTechniques))
		for kind, catalog := range config.aggressorBeaconTechniques {
			clone.aggressorBeaconTechniques[kind] = cloneAggressorBeaconTechniqueCatalog(catalog)
		}
	}
	return clone
}

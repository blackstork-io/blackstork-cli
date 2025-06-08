package definitions

// An interface for the blocks that are powered by the plugins (by content providers, data sources, publishers and formatters)
type ExecBlock interface {
	isExecBlock()

	GetSource() *ExecBlockDef
	// GetRunnerName() string
	// GetBlockName() string
	//
	// GetConfig() evaluation.Configuration
}

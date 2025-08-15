package commands

const (
	ConfigPathFlag         = "config"
	ConfigPathShortFlag    = "c"
	ConfigPathDefaultValue = ""
	ConfigPathUsage        = "Location of config file"

	ContinueGenerationFlag         = "continue"
	ContinueGenerationShortFlag    = "c"
	ContinueGenerationDefaultValue = false
	ContinueGenerationUsage        = "Continue generation from the last recorded row"

	ForceGenerationFlag             = "force"
	ForceGenerationShortFlag        = "f"
	ForceGenerationFlagDefaultValue = false
	ForceGenerationUsage            = "Force generation even if output file conflicts found and partition files limit reached" //nolint:lll

	TTYFlag      = "tty"
	TTYShortFlag = "t"
	TTYUsage     = "Activate TTY mode"

	NoTTYFlag         = "no-tty"
	NoTTYShortFlag    = "T"
	NoTTYDefaultValue = false
	NoTTYUsage        = "Deactivate TTY mode"

	DebugModeFlag         = "debug"
	DebugModeShortFlag    = "d"
	DebugModeDefaultValue = false
	DebugModeUsage        = "Enable debug mode"

	CPUProfileFlag         = "cpu-profile"
	CPUProfileShortFlag    = ""
	CPUProfileDefaultValue = ""
	CPUProfileUsage        = "Path to GoLang CPU profile file"

	MemoryProfileFlag         = "memory-profile"
	MemoryProfileShortFlag    = ""
	MemoryProfileDefaultValue = ""
	MemoryProfileUsage        = "Path to GoLang memory profile file"

	OpenAIAPIKeyFlag         = "api-key"
	OpenAIAPIKeyShortFlag    = "k"
	OpenAIAPIKeyDefaultValue = ""
	OpenAIAPIKeyUsage        = "Open AI API key"

	OpenAIBaseURLFlag         = "base-url"
	OpenAIBaseURLShortFlag    = "u"
	OpenAIBaseURLDefaultValue = ""
	OpenAIBaseURLUsage        = "Open AI base URL"

	OpenAIModelFlag         = "model"
	OpenAIModelShortFlag    = "m"
	OpenAIModelDefaultValue = ""
	OpenAIModelUsage        = "Open AI model"

	HTTPListenAddressFlag         = "listen-address"
	HTTPListenAddressShortFlag    = "a"
	HTTPListenAddressDefaultValue = ""
	HTTPListenAddressUsage        = "HTTP listen address"

	HTTPReadTimeoutFlag         = "read-timeout"
	HTTPReadTimeoutShortFlag    = "r"
	HTTPReadTimeoutDefaultValue = 0
	HTTPReadTimeoutUsage        = "HTTP read timeout"

	HTTPWriteTimeoutFlag         = "write-timeout"
	HTTPWriteTimeoutShortFlag    = "w"
	HTTPWriteTimeoutDefaultValue = 0
	HTTPWriteTimeoutUsage        = "HTTP write timeout"

	HTTPIdleTimeoutFlag         = "idle-timeout"
	HTTPIdleTimeoutShortFlag    = "i"
	HTTPIdleTimeoutDefaultValue = 0
	HTTPIdleTimeoutUsage        = "HTTP idle timeout"

	GenerationConfigSavePathFlag         = "save"
	GenerationConfigSavePathShortFlag    = "s"
	GenerationConfigSavePathDefaultValue = ""
	GenerationConfigSavePathUsage        = "Location to save generation config file"

	ExtraInputFlag         = "extra-input"
	ExtraInputShortFlag    = "e"
	ExtraInputDefaultValue = false
	ExtraInputUsage        = "Request clarifying information"

	ExtraFileFlag         = "extra-file"
	ExtraFileShortFlag    = "f"
	ExtraFileDefaultValue = ""
	ExtraSampleFileUsage  = "Location of file containing data samples"
	ExtraSQLFileUsage     = "Location of file containing SQL query"
)

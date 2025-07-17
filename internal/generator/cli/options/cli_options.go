package options

import (
	"os"

	"sdvg/internal/generator/cli/openai"
	"sdvg/internal/generator/cli/render"
	"sdvg/internal/generator/cli/streams"
	"sdvg/internal/generator/models"
	"sdvg/internal/generator/usecase"
)

// Option type is a value wrapper with a flag indicating whether its value has been modified.
type Option[T any] struct {
	Value   T
	Changed *bool
}

// SDVGOptions type is used to describe root command options.
type SDVGOptions struct {
	TTY           Option[bool]
	NoTTY         Option[bool]
	ConfigPath    string
	DebugMode     bool
	CPUProfile    string
	MemoryProfile string
	OpenAIAPIKey  string
	OpenAIBaseURL string
	OpenAIModel   string
}

type CliOptions struct {
	useCase     usecase.UseCase
	openAI      openai.Service
	renderer    render.Renderer
	in          *streams.In
	out         *streams.Out
	appConfig   *models.AppConfig
	sdvgOptions *SDVGOptions
	version     string
	useTTY      bool
}

func NewCliOptions(useCase usecase.UseCase, version string) *CliOptions {
	return &CliOptions{
		useCase:     useCase,
		version:     version,
		in:          streams.NewIn(os.Stdin),
		out:         streams.NewOut(os.Stdout),
		appConfig:   &models.AppConfig{},
		sdvgOptions: &SDVGOptions{},
	}
}

func (opts *CliOptions) UseCase() usecase.UseCase {
	return opts.useCase
}

func (opts *CliOptions) SetUseCase(useCase usecase.UseCase) {
	opts.useCase = useCase
}

func (opts *CliOptions) OpenAI() openai.Service {
	return opts.openAI
}

func (opts *CliOptions) SetOpenAI(openAI openai.Service) {
	opts.openAI = openAI
}

func (opts *CliOptions) Renderer() render.Renderer {
	return opts.renderer
}

func (opts *CliOptions) SetRenderer(renderer render.Renderer) {
	opts.renderer = renderer
}

func (opts *CliOptions) In() *streams.In {
	return opts.in
}

func (opts *CliOptions) SetIn(in *streams.In) {
	opts.in = in
}

func (opts *CliOptions) Out() *streams.Out {
	return opts.out
}

func (opts *CliOptions) SetOut(out *streams.Out) {
	opts.out = out
}

func (opts *CliOptions) AppConfig() *models.AppConfig {
	return opts.appConfig
}

func (opts *CliOptions) SetAppConfig(appConfig *models.AppConfig) {
	opts.appConfig = appConfig
}

func (opts *CliOptions) SDVGOpts() *SDVGOptions {
	return opts.sdvgOptions
}

func (opts *CliOptions) SetSDVGOpts(sdvgOpts *SDVGOptions) {
	opts.sdvgOptions = sdvgOpts
}

func (opts *CliOptions) UseTTY() bool {
	return opts.useTTY
}

func (opts *CliOptions) SetUseTTY(useTTY bool) {
	opts.useTTY = useTTY
}

func (opts *CliOptions) Version() string {
	return opts.version
}

func (opts *CliOptions) SetVersion(version string) {
	opts.version = version
}

func (opts *CliOptions) DebugMode() bool {
	return opts.SDVGOpts().DebugMode
}

func (opts *CliOptions) SetDebugMode(debugMode bool) {
	opts.SDVGOpts().DebugMode = debugMode
}

func (opts *CliOptions) CPUProfile() string {
	return opts.SDVGOpts().CPUProfile
}

func (opts *CliOptions) SetCPUProfile(profile string) {
	opts.SDVGOpts().CPUProfile = profile
}

func (opts *CliOptions) MemoryProfile() string {
	return opts.SDVGOpts().MemoryProfile
}

func (opts *CliOptions) SetMemoryProfile(profile string) {
	opts.SDVGOpts().MemoryProfile = profile
}

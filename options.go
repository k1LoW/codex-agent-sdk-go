package codex

// Options holds top-level configuration for the SDK.
type Options struct {
	CLIPath string
	Config  map[string]string
	Env     map[string]string
	Stderr  func(string)

	// Approval callbacks
	OnCommandApproval    OnCommandApprovalFunc
	OnFileChangeApproval OnFileChangeApprovalFunc
	OnPermissionApproval OnPermissionApprovalFunc
	OnToolCall           OnToolCallFunc
	OnUserInput          OnUserInputFunc
	OnElicitation        OnElicitationFunc
}

// Option configures a Codex instance or Query call.
type Option func(*Options)

// WithCLIPath sets the path to the codex binary.
func WithCLIPath(path string) Option {
	return func(o *Options) { o.CLIPath = path }
}

// WithConfig sets temporary Codex CLI config overrides.
func WithConfig(config map[string]string) Option {
	return func(o *Options) { o.Config = config }
}

// WithEnv sets environment variables for the CLI process.
func WithEnv(env map[string]string) Option {
	return func(o *Options) { o.Env = env }
}

// WithStderr sets a callback for stderr output from the CLI process.
func WithStderr(fn func(string)) Option {
	return func(o *Options) { o.Stderr = fn }
}

// WithOnCommandApproval sets the callback for command execution approval requests.
func WithOnCommandApproval(fn OnCommandApprovalFunc) Option {
	return func(o *Options) { o.OnCommandApproval = fn }
}

// WithOnFileChangeApproval sets the callback for file change approval requests.
func WithOnFileChangeApproval(fn OnFileChangeApprovalFunc) Option {
	return func(o *Options) { o.OnFileChangeApproval = fn }
}

// WithOnPermissionApproval sets the callback for permission approval requests.
func WithOnPermissionApproval(fn OnPermissionApprovalFunc) Option {
	return func(o *Options) { o.OnPermissionApproval = fn }
}

// WithOnToolCall sets the callback for dynamic tool call requests.
func WithOnToolCall(fn OnToolCallFunc) Option {
	return func(o *Options) { o.OnToolCall = fn }
}

// WithOnUserInput sets the callback for user input requests.
func WithOnUserInput(fn OnUserInputFunc) Option {
	return func(o *Options) { o.OnUserInput = fn }
}

// WithOnElicitation sets the callback for MCP elicitation requests.
func WithOnElicitation(fn OnElicitationFunc) Option {
	return func(o *Options) { o.OnElicitation = fn }
}

func applyOptions(opts []Option) *Options {
	o := &Options{}
	for _, opt := range opts {
		opt(o)
	}
	return o
}

// ThreadOptions holds configuration for thread/start.
type ThreadOptions struct {
	Model          string
	Sandbox        string
	ApprovalPolicy string
	CWD            string
	Ephemeral      *bool
}

// ThreadOption configures a thread.
type ThreadOption func(*ThreadOptions)

// WithModel sets the model for the thread.
func WithModel(model string) ThreadOption {
	return func(o *ThreadOptions) { o.Model = model }
}

// WithSandbox sets the sandbox mode for the thread.
func WithSandbox(mode string) ThreadOption {
	return func(o *ThreadOptions) { o.Sandbox = mode }
}

// WithApprovalPolicy sets the approval policy for the thread.
func WithApprovalPolicy(policy string) ThreadOption {
	return func(o *ThreadOptions) { o.ApprovalPolicy = policy }
}

// WithCWD sets the working directory for the thread.
func WithCWD(dir string) ThreadOption {
	return func(o *ThreadOptions) { o.CWD = dir }
}

// WithEphemeral sets the thread as ephemeral.
func WithEphemeral(v bool) ThreadOption {
	return func(o *ThreadOptions) { o.Ephemeral = &v }
}

func applyThreadOptions(opts []ThreadOption) *ThreadOptions {
	o := &ThreadOptions{}
	for _, opt := range opts {
		opt(o)
	}
	return o
}

func (o *ThreadOptions) toParams() map[string]any {
	params := map[string]any{}
	if o.Model != "" {
		params["model"] = o.Model
	}
	if o.Sandbox != "" {
		params["sandbox"] = o.Sandbox
	}
	if o.ApprovalPolicy != "" {
		params["approvalPolicy"] = o.ApprovalPolicy
	}
	if o.CWD != "" {
		params["cwd"] = o.CWD
	}
	if o.Ephemeral != nil {
		params["ephemeral"] = *o.Ephemeral
	}
	return params
}

// TurnOptions holds configuration for turn/start.
type TurnOptions struct {
	Model        string
	CWD          string
	Effort       string
	OutputSchema any
}

// TurnOption configures a single turn.
type TurnOption func(*TurnOptions)

// WithTurnModel sets the model for a turn.
func WithTurnModel(model string) TurnOption {
	return func(o *TurnOptions) { o.Model = model }
}

// WithTurnCWD sets the working directory for a turn.
func WithTurnCWD(dir string) TurnOption {
	return func(o *TurnOptions) { o.CWD = dir }
}

// WithEffort sets the reasoning effort for a turn.
func WithEffort(effort string) TurnOption {
	return func(o *TurnOptions) { o.Effort = effort }
}

// WithOutputSchema sets the JSON schema for structured output.
func WithOutputSchema(schema any) TurnOption {
	return func(o *TurnOptions) { o.OutputSchema = schema }
}

func applyTurnOptions(opts []TurnOption) *TurnOptions {
	o := &TurnOptions{}
	for _, opt := range opts {
		opt(o)
	}
	return o
}

func buildTurnStartParams(threadID string, input []UserInput, opts *TurnOptions) map[string]any {
	params := map[string]any{
		"threadId": threadID,
		"input":    input,
	}
	if opts.Model != "" {
		params["model"] = opts.Model
	}
	if opts.CWD != "" {
		params["cwd"] = opts.CWD
	}
	if opts.Effort != "" {
		params["effort"] = opts.Effort
	}
	if opts.OutputSchema != nil {
		params["outputSchema"] = opts.OutputSchema
	}
	return params
}

// QueryOption combines all option levels for the one-shot Query API.
type QueryOption func(*queryOptions)

type queryOptions struct {
	codex  []Option
	thread []ThreadOption
	turn   []TurnOption
}

// WithOptions wraps Option(s) for Query.
func WithOptions(opts ...Option) QueryOption {
	return func(o *queryOptions) { o.codex = append(o.codex, opts...) }
}

// WithThreadOptions wraps ThreadOption(s) for Query.
func WithThreadOptions(opts ...ThreadOption) QueryOption {
	return func(o *queryOptions) { o.thread = append(o.thread, opts...) }
}

// WithTurnOptions wraps TurnOption(s) for Query.
func WithTurnOptions(opts ...TurnOption) QueryOption {
	return func(o *queryOptions) { o.turn = append(o.turn, opts...) }
}

func applyQueryOptions(opts []QueryOption) *queryOptions {
	o := &queryOptions{}
	for _, opt := range opts {
		opt(o)
	}
	return o
}

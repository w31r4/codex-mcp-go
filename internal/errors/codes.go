package errors

// Code is a stable numeric error code.
//
// Notes:
// - JSON-RPC reserves the range -32768..-32000 for system/standard errors.
// - Server-defined errors commonly use the range -32099..-32000.
type Code int

const (
	// JSON-RPC standard error codes (reference).
	ParseError     Code = -32700
	InvalidRequest Code = -32600
	MethodNotFound Code = -32601
	InvalidParams  Code = -32602
	InternalError  Code = -32603

	// Server-defined error codes for codex-mcp-go.
	CodexNotFound        Code = -32001
	CodexTimeout         Code = -32002
	CodexExecutionFailed Code = -32003
	WorkdirNotFound      Code = -32004
	WorkdirNotDirectory  Code = -32005
	ImageNotFound        Code = -32006
	InvalidSandboxMode   Code = -32007
	ParameterProhibited  Code = -32008
	SessionNotFound      Code = -32009
	NoOutputTimeout      Code = -32010
	SessionLimitExceeded Code = -32011
	WorkdirBusy          Code = -32012
)

// Name returns a stable string identifier for the code.
func (c Code) Name() string {
	switch c {
	case ParseError:
		return "ParseError"
	case InvalidRequest:
		return "InvalidRequest"
	case MethodNotFound:
		return "MethodNotFound"
	case InvalidParams:
		return "InvalidParams"
	case InternalError:
		return "InternalError"
	case CodexNotFound:
		return "CodexNotFound"
	case CodexTimeout:
		return "CodexTimeout"
	case CodexExecutionFailed:
		return "CodexExecutionFailed"
	case WorkdirNotFound:
		return "WorkdirNotFound"
	case WorkdirNotDirectory:
		return "WorkdirNotDirectory"
	case ImageNotFound:
		return "ImageNotFound"
	case InvalidSandboxMode:
		return "InvalidSandboxMode"
	case ParameterProhibited:
		return "ParameterProhibited"
	case SessionNotFound:
		return "SessionNotFound"
	case NoOutputTimeout:
		return "NoOutputTimeout"
	case SessionLimitExceeded:
		return "SessionLimitExceeded"
	case WorkdirBusy:
		return "WorkdirBusy"
	default:
		return "UnknownError"
	}
}

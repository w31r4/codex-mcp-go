package config

import (
	"os"
	"strconv"
	"strings"
)

const (
	envServerName    = "CODEX_MCP_SERVER_NAME"
	envServerVersion = "CODEX_MCP_VERSION"

	envDefaultTimeout   = "CODEX_DEFAULT_TIMEOUT"
	envMaxTimeout       = "CODEX_MAX_TIMEOUT"
	envNoOutputTimeout  = "CODEX_NO_OUTPUT_TIMEOUT"
	envMaxBufferedLines = "CODEX_MAX_BUFFERED_LINES"
	envExecutablePath   = "CODEX_EXECUTABLE_PATH"

	envAllowedModels       = "CODEX_ALLOWED_MODELS"
	envAllowedProfiles     = "CODEX_ALLOWED_PROFILES"
	envDefaultSandbox      = "CODEX_DEFAULT_SANDBOX"
	envAllowedSandboxModes = "CODEX_ALLOWED_SANDBOX_MODES"
	envAllowedWorkDirs     = "CODEX_ALLOWED_WORK_DIRS"
	envDisableYolo         = "CODEX_DISABLE_YOLO"

	envLogLevel  = "CODEX_LOG_LEVEL"
	envLogFormat = "CODEX_LOG_FORMAT"
	envLogOutput = "CODEX_LOG_OUTPUT"
	envLogFile   = "CODEX_LOG_FILE"
)

func (c *Config) LoadFromEnv() {
	if c == nil {
		return
	}

	if v := strings.TrimSpace(os.Getenv(envServerName)); v != "" {
		c.Server.Name = v
	}
	if v := strings.TrimSpace(os.Getenv(envServerVersion)); v != "" {
		c.Server.Version = v
	}

	if v, ok := readIntEnv(envDefaultTimeout); ok {
		c.Codex.DefaultTimeoutSeconds = v
	}
	if v, ok := readIntEnv(envMaxTimeout); ok {
		c.Codex.MaxTimeoutSeconds = v
	}
	if v, ok := readIntEnv(envNoOutputTimeout); ok {
		c.Codex.DefaultNoOutputTimeoutSeconds = v
	}
	if v, ok := readIntEnv(envMaxBufferedLines); ok {
		c.Codex.MaxBufferedLines = v
	}
	if v := strings.TrimSpace(os.Getenv(envExecutablePath)); v != "" {
		c.Codex.ExecutablePath = v
	}

	if v, ok := readCSVEnv(envAllowedModels); ok {
		c.Security.AllowedModels = v
	}
	if v, ok := readCSVEnv(envAllowedProfiles); ok {
		c.Security.AllowedProfiles = v
	}
	if v := strings.TrimSpace(os.Getenv(envDefaultSandbox)); v != "" {
		c.Security.DefaultSandbox = v
	}
	if v, ok := readCSVEnv(envAllowedSandboxModes); ok {
		c.Security.AllowedSandboxModes = v
	}
	if v, ok := readCSVEnv(envAllowedWorkDirs); ok {
		c.Security.AllowedWorkDirs = v
	}
	if v := strings.TrimSpace(os.Getenv(envDisableYolo)); v != "" {
		if b, err := strconv.ParseBool(v); err == nil {
			c.Security.DisableYolo = b
		}
	}

	if v := strings.TrimSpace(os.Getenv(envLogLevel)); v != "" {
		c.Logging.Level = v
	}
	if v := strings.TrimSpace(os.Getenv(envLogFormat)); v != "" {
		c.Logging.Format = v
	}
	if v := strings.TrimSpace(os.Getenv(envLogOutput)); v != "" {
		c.Logging.Output = v
	}
	if v := strings.TrimSpace(os.Getenv(envLogFile)); v != "" {
		c.Logging.FilePath = v
	}
}

func readIntEnv(key string) (int, bool) {
	v := strings.TrimSpace(os.Getenv(key))
	if v == "" {
		return 0, false
	}
	i, err := strconv.Atoi(v)
	if err != nil {
		return 0, false
	}
	return i, true
}

func readCSVEnv(key string) ([]string, bool) {
	v := strings.TrimSpace(os.Getenv(key))
	if v == "" {
		return nil, false
	}
	parts := strings.Split(v, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		out = append(out, p)
	}
	return out, true
}

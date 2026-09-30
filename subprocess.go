package codex

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"sync"
	"time"
)

// subprocessTransport communicates with codex app-server via stdin/stdout.
type subprocessTransport struct {
	options *Options
	cliPath string

	cmd     *exec.Cmd
	stdin   io.WriteCloser
	decoder *json.Decoder
	stderr  io.ReadCloser

	writeMu  sync.Mutex
	closed   bool
	waitOnce sync.Once
	waitErr  error
}

func newSubprocessTransport(options *Options) *subprocessTransport {
	return &subprocessTransport{options: options}
}

func (t *subprocessTransport) Connect(ctx context.Context) error {
	cliPath, err := t.findCLI()
	if err != nil {
		return err
	}
	t.cliPath = cliPath

	args := t.buildArgs()
	t.cmd = exec.CommandContext(ctx, t.cliPath, args...) //nolint:gosec // cliPath is from findCLI (user-configured or well-known paths)

	// Environment
	env := os.Environ()
	env = append(env, "CODEX_INTERNAL_ORIGINATOR_OVERRIDE=codex_sdk_go")
	if t.options.Env != nil {
		for k, v := range t.options.Env {
			env = append(env, k+"="+v)
		}
	}
	t.cmd.Env = env

	// Pipes
	stdin, err := t.cmd.StdinPipe()
	if err != nil {
		return &CLIConnectionError{SDKError{Message: "failed to create stdin pipe", Cause: err}}
	}
	t.stdin = stdin

	stdout, err := t.cmd.StdoutPipe()
	if err != nil {
		return &CLIConnectionError{SDKError{Message: "failed to create stdout pipe", Cause: err}}
	}
	t.decoder = json.NewDecoder(stdout)

	if t.options.Stderr != nil {
		stderr, err := t.cmd.StderrPipe()
		if err != nil {
			return &CLIConnectionError{SDKError{Message: "failed to create stderr pipe", Cause: err}}
		}
		t.stderr = stderr
		go t.readStderr()
	}

	if err := t.cmd.Start(); err != nil {
		return &CLINotFoundError{SDKError: SDKError{Message: "failed to start codex app-server", Cause: err}, CLIPath: t.cliPath}
	}

	return nil
}

func (t *subprocessTransport) Write(data []byte) error {
	t.writeMu.Lock()
	defer t.writeMu.Unlock()

	if t.closed || t.stdin == nil {
		return &CLIConnectionError{SDKError{Message: "transport is closed"}}
	}
	_, err := t.stdin.Write(data)
	if err != nil {
		return &CLIConnectionError{SDKError{Message: "failed to write to stdin", Cause: err}}
	}
	return nil
}

func (t *subprocessTransport) ReadMessage() (map[string]any, error) {
	var msg map[string]any
	if err := t.decoder.Decode(&msg); err != nil {
		if errors.Is(err, io.EOF) {
			if waitErr := t.waitProcess(); waitErr != nil {
				if exitErr, ok := errors.AsType[*exec.ExitError](waitErr); ok {
					return nil, &ProcessError{
						SDKError: SDKError{Message: "CLI process failed"},
						ExitCode: exitErr.ExitCode(),
					}
				}
				return nil, &ProcessError{SDKError: SDKError{Message: "CLI process failed", Cause: waitErr}}
			}
			return nil, io.EOF
		}
		return nil, &JSONDecodeError{SDKError: SDKError{Message: "failed to decode JSON", Cause: err}}
	}
	return msg, nil
}

func (t *subprocessTransport) Close() error {
	t.writeMu.Lock()
	t.closed = true
	if t.stdin != nil {
		t.stdin.Close()
		t.stdin = nil
	}
	t.writeMu.Unlock()

	if t.cmd != nil && t.cmd.Process != nil {
		done := make(chan error, 1)
		go func() { done <- t.waitProcess() }()

		select {
		case err := <-done:
			return err
		case <-time.After(3 * time.Second):
			t.cmd.Process.Kill() //nolint:errcheck // best-effort: process may have already exited
			return <-done
		}
	}
	return nil
}

func (t *subprocessTransport) waitProcess() error {
	t.waitOnce.Do(func() {
		if t.cmd != nil {
			t.waitErr = t.cmd.Wait()
		}
	})
	return t.waitErr
}

func (t *subprocessTransport) readStderr() {
	if t.stderr == nil || t.options.Stderr == nil {
		return
	}
	buf := make([]byte, 4096)
	for {
		n, err := t.stderr.Read(buf)
		if n > 0 {
			for line := range strings.SplitSeq(strings.TrimRight(string(buf[:n]), "\n"), "\n") {
				if line != "" {
					t.options.Stderr(line)
				}
			}
		}
		if err != nil {
			return
		}
	}
}

func (t *subprocessTransport) findCLI() (string, error) {
	if t.options.CLIPath != "" {
		return t.options.CLIPath, nil
	}

	if p, err := exec.LookPath("codex"); err == nil {
		return p, nil
	}

	candidates := []string{"/usr/local/bin/codex"}
	if home, err := os.UserHomeDir(); err == nil {
		candidates = append(candidates,
			filepath.Join(home, ".local", "bin", "codex"),
			filepath.Join(home, "node_modules", ".bin", "codex"),
			filepath.Join(home, ".npm-global", "bin", "codex"),
			filepath.Join(home, ".yarn", "bin", "codex"),
		)
	}

	if runtime.GOOS == "windows" {
		for i, c := range candidates {
			candidates[i] = c + ".exe"
		}
	}

	for _, p := range candidates {
		if info, err := os.Stat(p); err == nil && !info.IsDir() {
			return p, nil
		}
	}

	return "", &CLINotFoundError{
		SDKError: SDKError{
			Message: "Codex CLI not found. Install with: npm install -g @openai/codex",
		},
	}
}

func (t *subprocessTransport) buildArgs() []string {
	args := []string{"app-server", "--listen", "stdio://"}
	if len(t.options.Config) == 0 {
		return args
	}

	keys := make([]string, 0, len(t.options.Config))
	for key := range t.options.Config {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	for _, key := range keys {
		args = append(args, "-c", key+"="+t.options.Config[key])
	}
	return args
}

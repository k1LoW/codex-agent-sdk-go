package codex

import (
	"reflect"
	"testing"
)

func TestSubprocessTransportBuildArgsWithConfig(t *testing.T) {
	transport := newSubprocessTransport(&Options{
		Config: map[string]string{
			"model":             "gpt-5.4",
			"sandbox_mode":      "workspace-write",
			"approval_policy":   "on-request",
			"model_provider.id": "openai",
		},
	})

	got := transport.buildArgs()
	want := []string{
		"app-server", "--listen", "stdio://",
		"-c", "approval_policy=on-request",
		"-c", "model=gpt-5.4",
		"-c", "model_provider.id=openai",
		"-c", "sandbox_mode=workspace-write",
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("buildArgs() = %#v, want %#v", got, want)
	}
}

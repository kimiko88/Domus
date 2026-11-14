package supervisor

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestEnsureContainerAppliesResourceAndSecurityOptions(t *testing.T) {
	dir := t.TempDir()
	argsFile := filepath.Join(dir, "ctr-args.log")
	ctrScript := filepath.Join(dir, "ctr")
	script := `#!/bin/sh
printf '%s\n' "$*" >> "$CTR_ARGS_FILE"
case "$*" in
  *"container info"*) exit 1 ;;
esac
exit 0
`
	if err := os.WriteFile(ctrScript, []byte(script), 0700); err != nil {
		t.Fatal(err)
	}
	t.Setenv("CTR_ARGS_FILE", argsFile)

	runtime := NewContainerdRuntime(filepath.Join(dir, "containerd.sock"))
	runtime.ctrBin = ctrScript
	err := runtime.EnsureContainer(context.Background(), "domus-app-test", ContainerConfig{
		Image: "example/app:1.0", NetworkMode: "host",
		MemoryLimitMB: 256, CPULimit: 1.5, ReadOnlyRoot: true,
	})
	if err != nil {
		t.Fatalf("EnsureContainer returned error: %v", err)
	}

	args, err := os.ReadFile(argsFile)
	if err != nil {
		t.Fatal(err)
	}
	got := string(args)
	for _, expected := range []string{
		"--memory-limit 268435456",
		"--cpu-period 100000",
		"--cpu-quota 150000",
		"--net-host",
		"--seccomp",
		"--read-only",
	} {
		if !strings.Contains(got, expected) {
			t.Errorf("ctr arguments missing %q; got:\n%s", expected, got)
		}
	}
}

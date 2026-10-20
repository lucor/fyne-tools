package mobile

import (
	"encoding/base64"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestBuildAPKDryRunSkipsAndroidDexMerge(t *testing.T) {
	previousBuildN := buildN
	buildN = true
	t.Cleanup(func() { buildN = previousBuildN })
	t.Setenv("ANDROID_HOME", "")

	extraDex := filepath.Join(t.TempDir(), "missing.dex")
	if _, err := buildAPK(io.Discard, nil, nil, nil, extraDex); err != nil {
		t.Fatalf("dry run should not require an Android SDK or read the extra DEX: %v", err)
	}
}

func TestMergeAndroidDexWithoutExtraReturnsEmbeddedDex(t *testing.T) {
	got, err := mergeAndroidDex("")
	if err != nil {
		t.Fatal(err)
	}
	want, err := base64.StdEncoding.DecodeString(dexStr)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != string(want) {
		t.Fatal("merged DEX differs from embedded DEX when no extra DEX is provided")
	}
}

func TestMergeAndroidDexUsesD8(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("fake d8 helper uses a shell script")
	}

	sdk := t.TempDir()
	buildTools := filepath.Join(sdk, "build-tools", "35.0.0")
	if err := os.MkdirAll(buildTools, 0o755); err != nil {
		t.Fatal(err)
	}
	extraDex := filepath.Join(t.TempDir(), "service.dex")
	if err := os.WriteFile(extraDex, []byte("service dex"), 0o600); err != nil {
		t.Fatal(err)
	}
	d8 := filepath.Join(buildTools, "d8")
	script := `#!/bin/sh
out=""
next=0
count=0
for arg in "$@"; do
  if [ "$next" = 1 ]; then out="$arg"; next=0; continue; fi
  if [ "$arg" = "--output" ]; then next=1; continue; fi
  case "$arg" in
    *.dex) test -f "$arg" || exit 21; count=$((count + 1));;
  esac
done
test "$count" -eq 2 || exit 22
printf merged-dex > "$out/classes.dex"
`
	if err := os.WriteFile(d8, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("ANDROID_HOME", sdk)

	got, err := mergeAndroidDex(extraDex)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "merged-dex" {
		t.Fatalf("got merged DEX %q, want %q", got, "merged-dex")
	}
}

func TestMergeAndroidDexReportsMissingD8(t *testing.T) {
	sdk := t.TempDir()
	if err := os.MkdirAll(filepath.Join(sdk, "build-tools"), 0o755); err != nil {
		t.Fatal(err)
	}
	extraDex := filepath.Join(t.TempDir(), "service.dex")
	if err := os.WriteFile(extraDex, []byte("service dex"), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("ANDROID_HOME", sdk)
	if _, err := mergeAndroidDex(extraDex); err == nil {
		t.Fatal("expected missing d8 error")
	}
}

func TestMergeAndroidDexRejectsMultipleD8Outputs(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("fake d8 helper uses a shell script")
	}

	sdk := t.TempDir()
	buildTools := filepath.Join(sdk, "build-tools", "35.0.0")
	if err := os.MkdirAll(buildTools, 0o755); err != nil {
		t.Fatal(err)
	}
	extraDex := filepath.Join(t.TempDir(), "service.dex")
	if err := os.WriteFile(extraDex, []byte("service dex"), 0o600); err != nil {
		t.Fatal(err)
	}
	d8 := filepath.Join(buildTools, "d8")
	script := `#!/bin/sh
out=""
next=0
for arg in "$@"; do
  if [ "$next" = 1 ]; then out="$arg"; next=0; continue; fi
  if [ "$arg" = "--output" ]; then next=1; fi
done
printf primary > "$out/classes.dex"
printf secondary > "$out/classes2.dex"
`
	if err := os.WriteFile(d8, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("ANDROID_HOME", sdk)

	if _, err := mergeAndroidDex(extraDex); err == nil || !strings.Contains(err.Error(), "multiple DEX files") {
		t.Fatalf("expected multiple DEX error, got %v", err)
	}
}

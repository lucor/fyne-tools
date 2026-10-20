package mobile

import (
	"encoding/base64"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"fyne.io/tools/cmd/fyne/internal/util"
)

func mergeAndroidDex(extraDex string) ([]byte, error) {
	baseDex, err := base64.StdEncoding.DecodeString(dexStr)
	if err != nil {
		return nil, fmt.Errorf("decode embedded Android DEX: %w", err)
	}
	if extraDex == "" {
		return baseDex, nil
	}
	if err := util.RequireAndroidSDK(); err != nil {
		return nil, err
	}

	buildTools := util.AndroidBuildToolsPath()
	d8 := filepath.Join(buildTools, "d8")
	if runtime.GOOS == "windows" {
		d8 += ".bat"
	}
	if _, err := os.Stat(d8); err != nil {
		return nil, fmt.Errorf("d8 not found in Android SDK build-tools (%s): %w", buildTools, err)
	}
	if _, err := os.Stat(extraDex); err != nil {
		return nil, fmt.Errorf("Android DEX file %q: %w", extraDex, err)
	}

	dir, err := os.MkdirTemp("", "fyne-dex-merge-*")
	if err != nil {
		return nil, err
	}
	defer os.RemoveAll(dir)

	basePath := filepath.Join(dir, fileClassesDex)
	if err := os.WriteFile(basePath, baseDex, 0o600); err != nil {
		return nil, err
	}
	outDir := filepath.Join(dir, "out")
	if err := os.Mkdir(outDir, 0o700); err != nil {
		return nil, err
	}
	args := []string{"--min-api", fmt.Sprint(minAndroidAPI), "--output", outDir, basePath, extraDex}
	var cmd *exec.Cmd
	if runtime.GOOS == "windows" {
		cmdArgs := append([]string{"/c", d8}, args...)
		cmd = exec.Command("cmd", cmdArgs...)
	} else {
		cmd = exec.Command(d8, args...)
	}
	if output, err := cmd.CombinedOutput(); err != nil {
		return nil, fmt.Errorf("merge Android DEX files with d8: %w: %s", err, output)
	}
	files, err := os.ReadDir(outDir)
	if err != nil {
		return nil, err
	}
	for _, file := range files {
		if strings.HasPrefix(file.Name(), "classes") && strings.HasSuffix(file.Name(), ".dex") && file.Name() != fileClassesDex {
			return nil, errors.New("d8 produced multiple DEX files; --android-dex requires a single DEX result")
		}
	}
	merged, err := os.ReadFile(filepath.Join(outDir, fileClassesDex))
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, errors.New("d8 did not produce classes.dex")
		}
		return nil, err
	}
	return merged, nil
}

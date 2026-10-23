package mobile

import (
	"image"
	"image/png"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"

	"fyne.io/tools/cmd/fyne/internal/metadata"
	"fyne.io/tools/cmd/fyne/internal/util"
)

func TestCompileAndroidResources(t *testing.T) {
	if err := util.RequireAndroidSDK(); err != nil {
		t.Skip("Android SDK required for resource compilation")
	}
	for _, tc := range []struct {
		name     string
		resource string
		wantErr  string
	}{
		{name: "generated resources only"},
		{name: "app resources", resource: "drawable/stop.xml"},
		{name: "conflicting resource", resource: "mipmap-xxxhdpi/ic_launcher.png", wantErr: "aapt2 link failed"},
		{name: "missing directory", resource: "missing", wantErr: "aapt2 compile failed"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			icon := filepath.Join(t.TempDir(), "icon.png")
			writeTestPNG(t, icon, 64)
			extra := ""
			if tc.resource != "" {
				extra = filepath.Join(t.TempDir(), "res")
				if tc.resource != "missing" {
					path := filepath.Join(extra, filepath.FromSlash(tc.resource))
					if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
						t.Fatal(err)
					}
					if filepath.Ext(path) == ".png" {
						writeTestPNG(t, path, 32)
					} else if err := os.WriteFile(path, []byte(`<shape xmlns:android="http://schemas.android.com/apk/res/android"><solid android:color="#ffffff"/></shape>`), 0o644); err != nil {
						t.Fatal(err)
					}
					if err := os.WriteFile(filepath.Join(extra, ".DS_Store"), []byte("ignored"), 0o644); err != nil {
						t.Fatal(err)
					}
				}
			}
			manifest := []byte(`<manifest xmlns:android="http://schemas.android.com/apk/res/android" package="dev.example.resources"><application android:icon="@mipmap/ic_launcher"/></manifest>`)
			arsc, resources, compiledManifest, err := compileAndroidResources(t.TempDir(), manifest, icon, "", "", "", nil, 29, 1, "1.0", extra)
			if tc.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tc.wantErr) {
					t.Fatalf("error = %v, want %q", err, tc.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			for _, path := range []string{arsc, compiledManifest} {
				if data, err := os.ReadFile(path); err != nil || len(data) == 0 {
					t.Fatalf("missing compiled output %q: %v", path, err)
				}
			}
			if extra != "" {
				if _, err := os.Stat(filepath.Join(resources, filepath.FromSlash(tc.resource))); err != nil {
					t.Fatalf("app resource was not packaged: %v", err)
				}
			}
		})
	}
}

func writeTestPNG(t *testing.T, path string, size int) {
	f, err := os.Create(path)
	assert.Nil(t, err)
	assert.Nil(t, png.Encode(f, image.NewRGBA(image.Rect(0, 0, size, size))))
	assert.Nil(t, f.Close())
}

func pngSize(t *testing.T, path string) int {
	f, err := os.Open(path)
	assert.Nil(t, err)
	defer f.Close()
	conf, err := png.DecodeConfig(f)
	assert.Nil(t, err)
	assert.Equal(t, conf.Width, conf.Height)
	return conf.Width
}

func Test_writeSplashResources(t *testing.T) {
	dir := t.TempDir()
	icon := filepath.Join(dir, "icon.png")
	custom := filepath.Join(dir, "custom.png")
	writeTestPNG(t, icon, 64)
	writeTestPNG(t, custom, 32)

	res := filepath.Join(dir, "res")
	err := writeSplashResources(res, &metadata.Splash{Background: "#015952"}, icon)
	assert.Nil(t, err)

	// the fallback icon is scaled to the fixed splash size
	assert.Equal(t, splashIconSize, pngSize(t, filepath.Join(res, "drawable-xxxhdpi", "splash_icon.png")))

	data, err := os.ReadFile(filepath.Join(res, "values", "colors.xml"))
	assert.Nil(t, err)
	assert.Contains(t, string(data), `<color name="splash_background">#015952</color>`)
	data, err = os.ReadFile(filepath.Join(res, "drawable", "splash_background.xml"))
	assert.Nil(t, err)
	assert.Contains(t, string(data), "@color/splash_background")
	assert.Contains(t, string(data), "@drawable/splash_icon")
	data, err = os.ReadFile(filepath.Join(res, "values", "styles.xml"))
	assert.Nil(t, err)
	assert.Contains(t, string(data), "android:windowBackground")
	data, err = os.ReadFile(filepath.Join(res, "values-v31", "styles.xml"))
	assert.Nil(t, err)
	assert.Contains(t, string(data), "android:windowSplashScreenAnimatedIcon")

	// a custom icon is preferred and the colour defaults to white
	res = filepath.Join(dir, "res2")
	assert.Nil(t, writeSplashResources(res, &metadata.Splash{Icon: custom}, icon))
	assert.Equal(t, splashIconSize, pngSize(t, filepath.Join(res, "drawable-xxxhdpi", "splash_icon.png")))
	data, err = os.ReadFile(filepath.Join(res, "values", "colors.xml"))
	assert.Nil(t, err)
	assert.Contains(t, string(data), "#FFFFFF")

	// a missing icon is an error
	assert.NotNil(t, writeSplashResources(filepath.Join(dir, "res3"), &metadata.Splash{}, filepath.Join(dir, "none.png")))
}

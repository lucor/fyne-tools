package commands

import (
	"flag"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/urfave/cli/v2"
)

func TestAndroidResourcesFlag(t *testing.T) {
	for _, command := range []*cli.Command{Package(), Release()} {
		t.Run(command.Name, func(t *testing.T) {
			flags := flag.NewFlagSet(command.Name, flag.ContinueOnError)
			var resources *cli.StringFlag
			for _, option := range command.Flags {
				if err := option.Apply(flags); err != nil {
					t.Fatal(err)
				}
				if option.Names()[0] == "android-resources" {
					resources = option.(*cli.StringFlag)
				}
			}
			if resources == nil {
				t.Fatal("Android resources flag is missing")
			}
			if err := flags.Parse([]string{"--android-resources", "app-res"}); err != nil {
				t.Fatal(err)
			}
			if *resources.Destination != "app-res" {
				t.Fatalf("resources path = %q", *resources.Destination)
			}
		})
	}
}

func Test_keyValueFlag_String(t *testing.T) {
	kvflag := &keyValueFlag{}
	assert.NoError(t, kvflag.Set("foo=bar"))
	assert.NoError(t, kvflag.Set("a=b=c"))
	assert.Error(t, kvflag.Set("meep"))

	assert.Equal(t, `"a=b=c", "foo=bar"`, kvflag.String())
}

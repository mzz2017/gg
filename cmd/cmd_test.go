package cmd

import (
	"reflect"
	"testing"

	"github.com/spf13/cobra"
)

func TestCommandArgumentsPassThroughUnchanged(t *testing.T) {
	var (
		gotArgs    []string
		gotVerbose bool
	)
	command := &cobra.Command{
		Use: "gg [flags] [command [argument ...]]",
		Run: func(command *cobra.Command, args []string) {
			gotArgs = args
			gotVerbose, _ = command.Flags().GetBool("verbose")
		},
	}
	command.Flags().BoolP("verbose", "v", false, "verbose")
	configureCommandPassthrough(command)
	command.SetArgs([]string{"--verbose", "curl", "--location", "-H", "accept: application/json"})

	if err := command.Execute(); err != nil {
		t.Fatal(err)
	}
	if !gotVerbose {
		t.Fatal("gg flag before command was not parsed")
	}
	wantArgs := []string{"curl", "--location", "-H", "accept: application/json"}
	if !reflect.DeepEqual(gotArgs, wantArgs) {
		t.Fatalf("command args = %#v, want %#v", gotArgs, wantArgs)
	}
}

package cmd

import (
	"bytes"
	"reflect"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

func TestLogUDPRedirectLimitation(t *testing.T) {
	tests := []struct {
		name       string
		noUDP      bool
		supportUDP bool
		wantLog    bool
	}{
		{name: "node disabled", supportUDP: false, wantLog: true},
		{name: "node supports UDP", supportUDP: true},
		{name: "globally disabled", noUDP: true, supportUDP: false},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var output bytes.Buffer
			log := NewLogger(1)
			log.SetOutput(&output)

			logUDPRedirectLimitation(log, test.noUDP, test.supportUDP)

			got := output.String()
			if test.wantLog != (got != "") {
				t.Fatalf("log output = %q, wantLog = %v", got, test.wantLog)
			}
			if test.wantLog && (!strings.Contains(got, "disabled or unsupported") || !strings.Contains(got, "DNS traffic may still be handled")) {
				t.Fatalf("log output = %q", got)
			}
		})
	}
}

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

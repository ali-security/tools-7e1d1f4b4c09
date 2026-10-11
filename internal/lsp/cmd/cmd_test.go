// Copyright 2019 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package cmd_test

import (
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"runtime"
	"strings"
	"testing"
	"time"

	"golang.org/x/tools/go/packages/packagestest"
	"golang.org/x/tools/internal/lsp/cmd"
	cmdtest "golang.org/x/tools/internal/lsp/cmd/test"
	"golang.org/x/tools/internal/lsp/tests"
	"golang.org/x/tools/internal/testenv"
	"golang.org/x/tools/internal/tool"
)

func TestMain(m *testing.M) {
	testenv.ExitIfSmallMachine()
	os.Exit(m.Run())
}

func TestCommandLine(t *testing.T) {
	packagestest.TestAll(t, testCommandLine)
}

func testCommandLine(t *testing.T, exporter packagestest.Exporter) {
	data := tests.Load(t, exporter, "../testdata")
	defer data.Exported.Cleanup()
	tests.Run(t, cmdtest.NewRunner(exporter, data, tests.Context(t), nil), data)
}

func TestDefinitionHelpExample(t *testing.T) {
	// TODO: https://golang.org/issue/32794.
	t.Skip()
	if runtime.GOOS == "android" {
		t.Skip("not all source files are available on android")
	}
	dir, err := os.Getwd()
	if err != nil {
		t.Errorf("could not get wd: %v", err)
		return
	}
	thisFile := filepath.Join(dir, "definition.go")
	baseArgs := []string{"query", "definition"}
	expect := regexp.MustCompile(`(?s)^[\w/\\:_-]+flag[/\\]flag.go:\d+:\d+-\d+: defined here as FlagSet struct {.*}$`)
	for _, query := range []string{
		fmt.Sprintf("%v:%v:%v", thisFile, cmd.ExampleLine, cmd.ExampleColumn),
		fmt.Sprintf("%v:#%v", thisFile, cmd.ExampleOffset)} {
		args := append(baseArgs, query)
		got := cmdtest.CaptureStdOut(t, func() {
			_ = tool.Run(tests.Context(t), cmd.New("gopls-test", "", nil, nil), args)
		})
		if !expect.MatchString(got) {
			t.Errorf("test with %v\nexpected:\n%s\ngot:\n%s", args, expect, got)
		}
	}
}

// TestServeListenImplicitAllInterfaces checks that -listen rejects an
// address with no host (e.g. ":12345"), which would implicitly bind all
// network interfaces.
func TestServeListenImplicitAllInterfaces(t *testing.T) {
	for _, args := range [][]string{
		{"serve", "-listen=:0"},
		// The serve flags are also accepted without the serve verb.
		{"-listen=:0"},
	} {
		errc := make(chan error, 1)
		go func(args []string) {
			errc <- tool.Run(tests.Context(t), cmd.New("gopls-test", "", nil, nil), args)
		}(args)
		select {
		case err := <-errc:
			if err == nil || !strings.Contains(err.Error(), "implicitly binds all network interfaces") {
				t.Errorf("gopls %v: got error %v, want rejection of the implicit all-interfaces address", args, err)
			}
		case <-time.After(10 * time.Second):
			t.Errorf("gopls %v: server was started listening on all network interfaces", args)
		}
	}
}

// TestServeNoPortFlag checks that the -port debugging flag, which listened
// on all network interfaces, is no longer provided.
func TestServeNoPortFlag(t *testing.T) {
	for _, typ := range []reflect.Type{
		reflect.TypeOf(cmd.Serve{}),
		reflect.TypeOf(cmd.Application{}),
	} {
		for i := 0; i < typ.NumField(); i++ {
			if f := typ.Field(i); f.Tag.Get("flag") == "port" {
				t.Errorf("%v.%s declares the -port flag", typ, f.Name)
			}
		}
	}
}

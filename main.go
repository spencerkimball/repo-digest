// Copyright 2016 The Cockroach Authors.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or
// implied. See the License for the specific language governing
// permissions and limitations under the License.
//
// Author: Spencer Kimball (spencer.kimball@gmail.com)

package main

import (
	"flag"
	"fmt"
	"os"
	"reflect"
	"strings"
	"time"

	"github.com/spencerkimball/repo-digest/cmd"
	"github.com/spf13/cobra"
)

// pflagValue wraps flag.Value and implements the extra methods of the
// pflag.Value interface.
type pflagValue struct {
	flag.Value
}

func (v pflagValue) Type() string {
	t := reflect.TypeOf(v.Value).Elem()
	return t.Kind().String()
}

func (v pflagValue) IsBoolFlag() bool {
	t := reflect.TypeOf(v.Value).Elem()
	return t.Kind() == reflect.Bool
}

func normalizeStdFlagName(s string) string {
	return strings.Replace(s, "_", "-", -1)
}

var digestCmd = &cobra.Command{
	Use:   "digest :owner/:repo --token=:access_token",
	Short: "generate daily digests of repository activity",
	Long: `
Generate an HTML digest of repository activity (default stylesheet
included). The digest includes two sections: a list of all newly-open
pull requests as well as a list of all recently-committed pull
requests.

Each pull request includes basic information, including title, author,
date, and metrics about which subdirectories of the repository are
most affected.

Pull requests are ordered by total modification size (additions +
deletions).
`,
	Example: `  digest cockroachdb/cockroach --token=f87456b1112dadb2d831a5792bf2ca9a6afca7bc`,
	RunE:    runDigest,
}

func runDigest(c *cobra.Command, args []string) error {
	if err := cmd.RunDigest(cmd.DigestCmd, args); err != nil {
		return err
	}
	return nil
}

func init() {
	// Map any flags registered in the standard "flag" package into the
	// top-level command.
	pf := digestCmd.PersistentFlags()
	flag.VisitAll(func(f *flag.Flag) {
		pf.Var(pflagValue{f.Value}, normalizeStdFlagName(f.Name), f.Usage)
	})
	now := time.Now().Local()
	now = now.Truncate(time.Hour * 24)
	defaultSince := now.Format(time.RFC3339)
	// Add persistent flags to the top-level command.
	digestCmd.PersistentFlags().StringVarP(&cmd.Repo, "repo", "r", "", cmd.RepoDesc)
	digestCmd.PersistentFlags().StringVarP(&cmd.FetchSince, "since", "s", defaultSince, cmd.FetchSinceDesc)
	digestCmd.PersistentFlags().StringVarP(&cmd.AccessToken, "token", "t", "", cmd.AccessTokenDesc)
}

// Run ...
func Run(args []string) error {
	digestCmd.SetArgs(args)
	return digestCmd.Execute()
}

func main() {
	if err := Run(os.Args[1:]); err != nil {
		fmt.Fprintf(os.Stderr, "failed running command %q: %v", os.Args[1:], err)
		os.Exit(1)
	}
}

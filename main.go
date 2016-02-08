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
	"errors"
	"flag"
	"fmt"
	"os"
	"reflect"
	"strings"
	"time"

	"github.com/cockroachdb/cockroach/util/log"
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

func mustParseTime3339(tStr string) time.Time {
	t, err := time.Parse(time.RFC3339, tStr)
	if err != nil {
		panic(fmt.Sprintf("couldn't parse time %q: %s", tStr, err))
	}
	return t
}

const accessTokenDesc = "GitHub access token for authorized rate limits"

const fetchSinceDesc = "Fetch all opened and closed pull requests since this date"

const repoDesc = "GitHub owner and repository, formatted as :owner/:repo"

const templateDesc = "Go HTML template filename (see templates/ for examples)"

const outDirDesc = "Output directory"

var digestCmd = &cobra.Command{
	Use:   "repo-digest",
	Short: "generate daily digests of repository activity",
	Long: `
Generate an HTML digest of repository activity (default stylesheet
included). The digest includes two sections: a list of all newly-open
pull requests as well as a list of all recently-committed pull
requests.

Fetches GitHub data for the specified repository and computes the digest
since the --since date. The digest contains two sections including:

Each pull request includes basic information, including title, author,
date, and metrics about which subdirectories of the repository are
most affected.

Pull requests are ordered by total modification size (additions +
deletions).

An access token can be specified via --token. By default, uses an empty
token, which is limited to only 50 GitHub requests per hour, rate limited
based on IP address.

To generate an access token with authorized rate limits (5000/hr), see:

https://help.github.com/articles/creating-an-access-token-for-command-line-use/

When creating a token, ensure that only the public_repo permission is enabled.
`,
	Example: `  repo-digest --repo=cockroachdb/cockroach --token=f87456b1112dadb2d831a5792bf2ca9a6afca7bc`,
	RunE:    runDigest,
}

// Context holds config information used to query GitHub.
type Context struct {
	Repo         string    // Repository (:owner/:repo)
	Token        string    // Access token
	FetchSince   time.Time // Fetch all opened and closed PRs since this time
	Template     string    // HTML template filename
	OutDir       string    // Output directory
	acceptHeader string    // Optional Accept: header value
}

var ctx = Context{}

func runDigest(c *cobra.Command, args []string) error {
	if len(ctx.Repo) == 0 {
		return errors.New("repository not specified; use --repo=:owner/:repo")
	}
	if len(ctx.Template) == 0 {
		return errors.New("template not specified; use --template=:html_template")
	}
	log.Infof("fetching GitHub data for repository %s", ctx.Repo)
	open, closed, err := Query(&ctx)
	if err != nil {
		log.Errorf("failed to query data: %s", err)
		return nil
	}
	log.Infof("creating digest for repository %s", ctx.Repo)
	if err := Digest(&ctx, open, closed); err != nil {
		log.Errorf("failed to create digest: %s", err)
		return nil
	}
	var latestTime time.Time
	for _, pr := range open {
		if t := mustParseTime3339(pr.CreatedAt); t.After(latestTime) {
			latestTime = t
		}
	}
	for _, pr := range closed {
		if t := mustParseTime3339(pr.ClosedAt); t.After(latestTime) {
			latestTime = t
		}
	}
	if len(open)+len(closed) == 0 {
		latestTime = time.Now()
	}
	fmt.Fprintf(os.Stdout, "nextsince: %s\n", latestTime.Format(time.RFC3339))
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
	var since string
	// Add persistent flags to the top-level command.
	digestCmd.PersistentFlags().StringVarP(&ctx.Repo, "repo", "r", ctx.Repo, repoDesc)
	digestCmd.PersistentFlags().StringVarP(&since, "since", "s", defaultSince, fetchSinceDesc)
	digestCmd.PersistentFlags().StringVarP(&ctx.Token, "token", "t", ctx.Token, accessTokenDesc)
	digestCmd.PersistentFlags().StringVarP(&ctx.Template, "template", "p", ctx.Template, templateDesc)
	digestCmd.PersistentFlags().StringVarP(&ctx.OutDir, "outdir", "o", ctx.OutDir, outDirDesc)

	var err error
	if ctx.FetchSince, err = time.Parse(time.RFC3339, since); err != nil {
		log.Errorf("failed to parse --since=%s: %s", since, err)
		os.Exit(1)
	}
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

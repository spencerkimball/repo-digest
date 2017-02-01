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
	"log"
	"os"
	"reflect"
	"strings"
	"time"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"github.com/spf13/cobra/doc"
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

const hostDesc = "GitHub API hostname, including scheme"

const accessTokenDesc = "GitHub access token for authorized rate limits"

const fetchBeforeDesc = "Fetch all opened and closed pull requests up until this date"

const fetchSinceDesc = "Fetch all opened and closed pull requests since this date"

const reposDesc = "GitHub repositories, formatted as comma-separated list :owner/:repo[,:owner/:repo,...]"

const templateDesc = "Go HTML template filename (see templates/ for examples)"

const outDirDesc = "Output directory"

const inlineStylesDesc = "Inline styles in generated html; good for standalone files"

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

For use against privately hosted github enterprise instances, the root API
address can be specified with the --host flag.  Note that while the core github
API root is specified via subdomain (https://api.github.com/), private enterprise
instances generally have an API root specified by URL path
(https://github.example.com/api/v3/, for example).
`,
	Example: `  repo-digest --repos=cockroachdb/cockroach --token=f87456b1112dadb2d831a5792bf2ca9a6afca7bc`,
	RunE:    runDigest,
}

// Config holds config information used to query GitHub.
type Config struct {
	Host         string    // Github API Hostname (https://api.github.com)
	Repos        []string  // Repositories (:owner/:repo)
	Token        string    // Access token
	Before       string    // RFC 3339 date
	Since        string    // RFC 3339 date
	Template     string    // HTML template filename
	OutDir       string    // Output directory
	InlineStyles bool      // Inline style into generated html
	Now          time.Time // Current time for this run of the repo-digest
	FetchSince   time.Time // Fetch all opened and closed PRs since this time
	acceptHeader string    // Optional Accept: header value
}

var cfg = Config{
	Template: "templates/default",
}

func initConfig() error {
	if len(cfg.Repos) == 0 {
		return errors.Errorf("repositories not specified; use --repos=:owner/:repo[,:owner/:repo,...]")
	}
	if len(cfg.Template) == 0 {
		return errors.Errorf("template not specified; use --template=:html_template")
	}

	// Parse dates and recast as local timezone.
	var err error
	if cfg.Now, err = time.Parse(time.RFC3339, cfg.Before); err != nil {
		return errors.Errorf("failed to parse --before=%s: %s", cfg.Before, err)
	}
	cfg.Now = cfg.Now.Local()
	if cfg.FetchSince, err = time.Parse(time.RFC3339, cfg.Since); err != nil {
		return errors.Errorf("failed to parse --since=%s: %s", cfg.Since, err)
	}
	cfg.FetchSince = cfg.FetchSince.Local()

	return nil
}

func runDigest(c *cobra.Command, args []string) error {
	if err := initConfig(); err != nil {
		return err
	}

	log.Printf("fetching GitHub data for repositories %s\n", cfg.Repos)
	open, closed, err := Query(&cfg)
	if err != nil {
		return errors.Errorf("failed to query data: %s", err)
	}
	log.Printf("creating digest for repositories %s\n", cfg.Repos)
	if err := Digest(&cfg, open, closed); err != nil {
		return errors.Errorf("failed to create digest: %s", err)
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
	latestTime = latestTime.Local()
	fmt.Fprintf(os.Stdout, "since: %s\n", cfg.FetchSince.Format(time.RFC3339))
	fmt.Fprintf(os.Stdout, "prettysince: %s\n", cfg.FetchSince.Format(time.UnixDate))
	fmt.Fprintf(os.Stdout, "nextsince: %s\n", latestTime.Format(time.RFC3339))
	return nil
}

var countMonthlyCmd = &cobra.Command{
	Use:   "count-monthly",
	Short: "count monthly PRs",
	Long: `
Output monthly counts of pull requests
`,
	Example: `  repo-digest count-monthly`,
	RunE:    runCountMonthly,
}

func runCountMonthly(c *cobra.Command, args []string) error {
	if err := initConfig(); err != nil {
		return err
	}

	log.Printf("counting monthly pull requests for repositories %s", cfg.Repos)

	counts, err := CountMonthly(&cfg)
	if err != nil {
		return err
	}

	var idx int
	for t := cfg.Now; !t.Before(cfg.FetchSince); {
		fmt.Printf("%s, %d\n", t, counts[idx])
		t = t.AddDate(0, -1, 0)
		idx++
	}

	return nil
}

var genDocCmd = &cobra.Command{
	Use:   "gendoc",
	Short: "generate markdown documentation",
	Long: `
Generate markdown documentation
`,
	Example: `  repo-digest gendoc`,
	RunE:    runGenDoc,
}

func runGenDoc(c *cobra.Command, args []string) error {
	return doc.GenMarkdown(digestCmd, os.Stdout)
}

func init() {
	digestCmd.AddCommand(
		countMonthlyCmd,
		genDocCmd,
	)
	// Map any flags registered in the standard "flag" package into the
	// top-level command.
	pf := digestCmd.PersistentFlags()
	flag.VisitAll(func(f *flag.Flag) {
		pf.Var(pflagValue{f.Value}, normalizeStdFlagName(f.Name), f.Usage)
	})
	defaultBefore := time.Now().Local()
	defaultBeforeStr := defaultBefore.Format(time.RFC3339)
	defaultSince := defaultBefore.Add(-24 * time.Hour)
	defaultSinceStr := defaultSince.Format(time.RFC3339)
	// Add persistent flags to the top-level command.
	digestCmd.PersistentFlags().StringVar(&cfg.Host, "host", "https://api.github.com/", hostDesc)
	digestCmd.PersistentFlags().StringSliceVarP(&cfg.Repos, "repos", "r", cfg.Repos, reposDesc)
	digestCmd.PersistentFlags().StringVarP(&cfg.Before, "before", "b", defaultBeforeStr, fetchBeforeDesc)
	digestCmd.PersistentFlags().StringVarP(&cfg.Since, "since", "s", defaultSinceStr, fetchSinceDesc)
	digestCmd.PersistentFlags().StringVarP(&cfg.Token, "token", "t", cfg.Token, accessTokenDesc)
	digestCmd.PersistentFlags().StringVarP(&cfg.Template, "template", "p", cfg.Template, templateDesc)
	digestCmd.PersistentFlags().StringVarP(&cfg.OutDir, "outdir", "o", cfg.OutDir, outDirDesc)
	digestCmd.PersistentFlags().BoolVar(&cfg.InlineStyles, "inline-styles", true, inlineStylesDesc)
}

// Run ...
func Run(args []string) error {
	digestCmd.SetArgs(args)
	return digestCmd.Execute()
}

func main() {
	if err := Run(os.Args[1:]); err != nil {
		fmt.Fprintf(os.Stderr, "failed running command %q: %v\n", os.Args[1:], err)
		os.Exit(1)
	}
}

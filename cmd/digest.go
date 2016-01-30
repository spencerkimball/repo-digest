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

package cmd

import (
	"errors"
	"fmt"
	"time"

	"github.com/cockroachdb/cockroach/util/log"
	"github.com/spencerkimball/repo-digest/digest"
	"github.com/spencerkimball/repo-digest/fetch"
	"github.com/spf13/cobra"
)

// DigestCmd digests previously fetched GitHub data.
var DigestCmd = &cobra.Command{
	Use:   "digest --repo=:owner/:repo --token=:access_token",
	Short: "create a digest of repository activity for current day",
	Long: `

Fetches GitHub data for the specified repository and computes the digest for
the previous 24 hours. The digest contains two sections including:

    - Newly opened pull requests
    - Committed pull requests
`,
	Example: `  repo-digest digest --repo=cockroachdb/cockroach --token=f87456b1112dadb2d831a5792bf2ca9a6afca7bc`,
	RunE:    RunDigest,
}

// RunDigest fetches saved GitHub data for the specified repo and
// computes the digest.
func RunDigest(cmd *cobra.Command, args []string) error {
	if len(Repo) == 0 {
		return errors.New("repository not specified; use --repo=:owner/:repo")
	}
	token, err := getAccessToken()
	if err != nil {
		return err
	}
	log.Infof("fetching GitHub data for repository %s", Repo)
	fetchSince, err := time.Parse(time.RFC3339, FetchSince)
	if err != nil {
		return err
	}
	fetchCtx := &fetch.Context{
		Repo:       Repo,
		Token:      token,
		FetchSince: fetchSince,
	}
	open, closed, err := fetch.Query(fetchCtx)
	if err != nil {
		log.Errorf("failed to query data: %s", err)
		return nil
	}
	log.Infof("creating digest for repository %s", Repo)
	if err := digest.Digest(fetchCtx, open, closed); err != nil {
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
	log.Infof("next digest should specify --since=%s", latestTime.Format(time.RFC3339))
	return nil
}

func mustParseTime3339(tStr string) time.Time {
	t, err := time.Parse(time.RFC3339, tStr)
	if err != nil {
		panic(fmt.Sprintf("couldn't parse time %q: %s", tStr, err))
	}
	return t
}

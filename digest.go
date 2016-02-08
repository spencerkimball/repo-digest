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
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"sort"
	"text/template"
	"time"

	"github.com/shurcooL/github_flavored_markdown"
	"github.com/vanng822/go-premailer/premailer"
)

type PullRequests []*PullRequest

func (slice PullRequests) Len() int {
	return len(slice)
}

func (slice PullRequests) Less(i, j int) bool {
	return slice[i].TotalChanges() > slice[j].TotalChanges()
}

func (slice PullRequests) Swap(i, j int) {
	slice[i], slice[j] = slice[j], slice[i]
}

func markDowner(args ...interface{}) string {
	return string(github_flavored_markdown.Markdown([]byte(fmt.Sprintf("%s", args...))))
}

// Digest computes the digest from provided slices of open and
// closed pull requests.
func Digest(c *Context, open, closed []*PullRequest) error {
	sortedOpen := PullRequests(open)
	sortedClosed := PullRequests(closed)
	sort.Sort(sortedOpen)
	sort.Sort(sortedClosed)

	// Open file for digest HTML.
	now := time.Now()
	content := struct {
		Repo   string
		Open   []*PullRequest
		Closed []*PullRequest
	}{
		Repo:   c.Repo,
		Open:   sortedOpen,
		Closed: sortedClosed,
	}
	htmlTemplate, err := ioutil.ReadFile(c.Template)
	if err != nil {
		return fmt.Errorf("failed to read template file %q: %s", c.Template, err)
	}
	tmpl := template.Must(template.New("digest").Funcs(template.FuncMap{"markDown": markDowner}).Parse(string(htmlTemplate)))

	buf := new(bytes.Buffer)
	if err := tmpl.Execute(buf, content); err != nil {
		return err
	}

	contents := buf.String()

	if c.InlineStyles {
		options := premailer.NewOptions()
		options.CssToAttributes = true
		prem := premailer.NewPremailerFromString(buf.String(), options)
		contents, err = prem.Transform()
		if err != nil {
			return err
		}
	}

	f, err := createFile(c.OutDir, fmt.Sprintf("digest-%s.html", now.Format("01-02-2006")))
	if err != nil {
		return err
	}
	defer f.Close()

	_, err = f.WriteString(contents)
	if err != nil {
		return err
	}

	fmt.Fprintf(os.Stdout, "digest: %s\n", f.Name())
	return nil
}

func createFile(dir, baseName string) (*os.File, error) {
	filename := filepath.Join(dir, baseName)
	f, err := os.Create(filename)
	if err != nil {
		return nil, err
	}
	return f, nil
}

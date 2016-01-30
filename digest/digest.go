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

package digest

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"text/template"
	"time"

	"github.com/cockroachdb/cockroach/util/log"
	"github.com/shurcooL/github_flavored_markdown"
	"github.com/spencerkimball/repo-digest/fetch"
)

type PullRequests []*fetch.PullRequest

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

const htmlTemplate = `
<!DOCTYPE html>
<html>
  <head>
    <meta charset="UTF-8">
    <link href=".../github-flavored-markdown.css" media="all" rel="stylesheet" type="text/css" />
    <link href="//cdnjs.cloudflare.com/ajax/libs/octicons/2.1.2/octicons.css" media="all" rel="stylesheet" type="text/css" />
    <title>Daily Digest of {{.Repo}}</title>
  </head>
  <body>
    <div class="open-requests">
      <div class="section-title">Opened Pull Requests</div>
  		{{range .Open}}
      <div class="{{ .Class }}">
        <div class="title"><img src="{{ .User.AvatarURL }}" class="avatar"/><a href="{{ .HtmlURL }}">{{ .Title }}</a></div>
        <div class="stats">Opened by {{ .User.Login }} at {{ .CreatedAtStr }} with {{ .Additions }} additions, {{ .Deletions }} deletions, {{ .Comments }} comments</div>
        <div class="subdirectories">{{ range $index, $el := .Subdirectories}}{{if $index}}, {{end}}{{$el.Name}}: {{$el.TotalChanges}}{{end}}</div>
    	  <article class="markdown-body entry-content" style="padding: 30px;">{{ .Body | markDown }}</article>
      </div>{{else}}
      <div><strong>No new pull requests were opened</strong></div>{{end}}
    </div>

    <div class="closed-requests">
      <div class="section-title">Closed Pull Requests</div>
  		{{range .Closed}}
      <div class="{{ .Class }}">
        <div class="title"><img src="{{ .User.AvatarURL }}" class="avatar"/><a href="{{ .HtmlURL }}">{{ .Title }}</a></div>
        <div class="stats">Closed by {{ .User.Login }} at {{ .CreatedAtStr }} with {{ .Additions }} additions, {{ .Deletions }} deletions, {{ .Comments }} comments</div>
        <div class="subdirectories">{{ range $index, $el := .Subdirectories}}{{if $index}}, {{end}}{{$el.Name}}: {{$el.TotalChanges}}{{end}}</div>
    	  <article class="markdown-body entry-content" style="padding: 30px;">{{ .Body | markDown }}</article>
      </div>{{else}}
      <div><strong>No pull requests were closed</strong></div>{{end}}
    </div>
  </body>
</html>
`

// Digest computes the digest from provided slices of open and
// closed pull requests.
func Digest(c *fetch.Context, open, closed []*fetch.PullRequest) error {
	sortedOpen := PullRequests(open)
	sortedClosed := PullRequests(closed)
	sort.Sort(sortedOpen)
	sort.Sort(sortedClosed)

	// Open file for digest HTML.
	now := time.Now()
	f, err := createFile(fmt.Sprintf("digest-%s.html", now.Format("01-02-2006")))
	if err != nil {
		return fmt.Errorf("failed to create file: %s", err)
	}
	defer f.Close()

	content := struct {
		Repo   string
		Open   []*fetch.PullRequest
		Closed []*fetch.PullRequest
	}{
		Repo:   c.Repo,
		Open:   sortedOpen,
		Closed: sortedClosed,
	}
	tmpl := template.Must(template.New("digest").Funcs(template.FuncMap{"markDown": markDowner}).Parse(htmlTemplate))
	if err != nil {
		return err
	}
	if err = tmpl.Execute(f, content); err != nil {
		return err
	}
	log.Infof("wrote HTML digest to %s", f.Name())
	return nil
}

func createFile(baseName string) (*os.File, error) {
	filename := filepath.Join("./", baseName)
	f, err := os.Create(filename)
	if err != nil {
		return nil, err
	}
	return f, nil
}

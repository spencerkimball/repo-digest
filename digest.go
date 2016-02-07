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
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"text/template"
	"time"

	"github.com/cockroachdb/cockroach/util/log"
	"github.com/shurcooL/github_flavored_markdown"
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

const htmlTemplate = `
<!DOCTYPE html>
<html>
  <head>
    <meta charset="UTF-8">
    <link href=".../github-flavored-markdown.css" media="all" rel="stylesheet" type="text/css" />
    <link href="//cdnjs.cloudflare.com/ajax/libs/octicons/2.1.2/octicons.css" media="all" rel="stylesheet" type="text/css" />
    <style type="text/css">
body {
    padding: 25px;
    width: 800px;
}

a, u {
    text-decoration: none;
    color: #409414;
}

table {
    width: 100%;
    border-spacing: 0px;
}

td {
    padding: 40px;
}

.logo {
    line-height: 60px;
}

.section-title {
    font-family: Arial-BoldMT;
    font-size: 22px;
    color: #142649;
    line-height: 48px;
}

.open-request {
    border: 1px solid #47A417;
}

.closed-request {
    border: 1px solid #CDCDCD;
}

.header {
    background: #F7F7F7;
}

.title {
    font-family: Arial-BoldMT;
    font-size: 18px;
    color: #409414;
    line-height: 32px;
}

.avatar {
    height: 50px;
}

.stats {
    font-family: ArialMT;
    font-size: 12px;
    color: #142649;
    line-height: 24px;
}

.rank-stats {
    font-family: ArialMT;
    font-size: 12px;
    color: #142649;
    line-height: 18px;
}

.body {
    font-family: ArialMT;
    font-size: 12px;
    color: #142649;
    line-height: 16px;
}

.rank {
    font-size: 14px;
    color: #409414;
}

.line-count {
    font-size: 12px;
    color: #409414;
}

.importance {
    font-family: Arial-BoldMT;
    font-size: 8px;
    color: #C9C9C9;
    line-height: 16px;
}

.subdirectory {
    font-family: Arial-BoldMT;
    font-size: 12px;
    line-height: 16px;
}

.spacing {
    line-height: 24px;
}
    </style>
    <title>Daily Digest of {{.Repo}}</title>
  </head>
  <body>
    <div class="logo">
      <img src="http://www.cockroachlabs.com/wp-content/themes/cockroach-labs/images/CL_Logo_Horizontal.png" height="25px" valign="top"/>
    </div>
    <div class="section-title">Opened Pull Requests</div>
		{{range .Open}}
    <table class="open-request">
      <tr class="header">
        <td class="title">
          <a href="{{ .HtmlURL }}">{{ .Title }}</a>
          <div class="stats">Opened by {{ .User.Login }} at {{ .CreatedAtStr }} with {{ .Additions }} additions, {{ .Deletions }} deletions, {{ .Comments }} comments</div>
          <div class="rank-stats"><span class="rank">{{ .Class }}</span>&nbsp;<span class="importance">IMPORTANCE</span>&nbsp;&nbsp;&nbsp;&nbsp;
            {{ range $index, $el := .Subdirectories}}
              <span class="subdirectory">{{$el.Name}}</span>: <span class="line-count">{{$el.TotalChanges}}</span>{{if $index}},&nbsp;&nbsp;{{end}}
            {{end}}
          </div>
        </td>
        <td class="title"><img src="{{ .User.AvatarURL }}" class="avatar"/></td>
      </tr>
      <tr class="body">
        <td>
      	  <article class="markdown-body entry-content">{{ .Body | markDown }}</article>
        </td>
      </tr>
    </table>
    <div class="spacer">&nbsp</div>
    {{else}}
    <div class="title">No new pull requests were opened</div>
    {{end}}

    <div class="section-title">Closed Pull Requests</div>
		{{range .Closed}}
    <table class="closed-request">
      <tr class="header">
        <td class="title">
          <a href="{{ .HtmlURL }}">{{ .Title }}</a>
          <div class="stats">Closed by {{ .User.Login }} at {{ .ClosedAtStr }} with {{ .Additions }} additions, {{ .Deletions }} deletions, {{ .Comments }} comments</div>
          <div class="rank-stats"><span class="rank">{{ .Class }}</span>&nbsp;<span class="importance">IMPORTANCE</span>&nbsp;&nbsp;&nbsp;&nbsp;
            {{ range $index, $el := .Subdirectories}}
              <span class="subdirectory">{{$el.Name}}</span>: <span class="line-count">{{$el.TotalChanges}}</span>{{if $index}},&nbsp;&nbsp;{{end}}
            {{end}}
          </div>
        </td>
        <td class="title"><img src="{{ .User.AvatarURL }}" class="avatar"/></td>
      </tr>
      <tr class="body">
        <td>
      	  <article class="markdown-body entry-content">{{ .Body | markDown }}</article>
        </td>
      </tr>
    </table>
    <div class="spacer">&nbsp</div>
    {{else}}
    <div class="title">No pull requests were closed</div>
    {{end}}
  </body>
</html>
`

// Digest computes the digest from provided slices of open and
// closed pull requests.
func Digest(c *Context, open, closed []*PullRequest) error {
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
		Open   []*PullRequest
		Closed []*PullRequest
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

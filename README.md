# repo-digest

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

```
Usage:
  repo-digest [flags]

Examples:
  repo-digest --repo=cockroachdb/cockroach --token=f87456b1112dadb2d831a5792bf2ca9a6afca7bc

Flags:
      --alsologtostderr    log to standard error as well as files
      --color              colorize standard error output according to severity (default "auto")
      --log-backtrace-at   when logging hits line file:N, emit a stack trace (default :0)
      --log-dir            if non-empty, write log files in this directory
      --logtostderr        log to standard error instead of files (default true)
  -r, --repo string        GitHub owner and repository, formatted as :owner/:repo
  -s, --since string       Fetch all opened and closed pull requests since this date (default "2016-02-07T19:00:00-05:00")
  -p, --template string    Go HTML template filename (see templates/ for examples)
  -t, --token string       GitHub access token for authorized rate limits
      --verbosity          log level for V logs
      --vmodule            comma-separated list of pattern=N settings for file-filtered logging
```

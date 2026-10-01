# Issue tracker: GitHub

Issues and specs live in GitHub Issues for
`natasha-audrey/lastfm-collage-generator`. Use the `gh` CLI
from this repository; it infers the repository from the remote.

## Conventions

- Create: `gh issue create --title "..." --body-file <path>`.
- Read: `gh issue view <number> --comments`.
- List: `gh issue list --state open --json number,title,body,labels,comments`.
  Apply label and state filters as needed.
- Comment: `gh issue comment <number> --body-file <path>`.
- Apply labels: `gh issue edit <number> --add-label "..."`.
- Remove labels: `gh issue edit <number> --remove-label "..."`.
- Close: `gh issue close <number> --comment "..."`.

For multiline bodies, write the exact Markdown to a temporary file
and pass it with `--body-file`.

## Pull requests as a triage surface

PRs as a request surface: no.

GitHub issues and PRs share a number space. When a reference is
ambiguous, try `gh pr view <number>` and fall back to
`gh issue view <number>`.

## Skill instructions

When a skill says "publish to the issue tracker", create a GitHub issue.
When it says "fetch the relevant ticket", run
`gh issue view <number> --comments`.

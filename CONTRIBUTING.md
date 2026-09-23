# Contributing

Thanks for taking the time to look at this repository! Contributions to this
repository are welcome. Consider submitting a PR or [issue](https://github.com/natasha-audrey/lastfm-collage-generator/issues/new)

## Style

* Refer to [the go code review comments](https://go.dev/wiki/CodeReviewComments)
and [effective go](https://go.dev/doc/effective_go).
* As an emphasis, return errors for recoverable errors instead of calling panic.
* Test fns should be structured to be named `TestFnName_TestCase` e.g.
`TestGetTopAlbums_ErrorsOnEmptyAPIKey`.

## Setup

Refer to [`.envrc.example`](./.envrc.example), you'll want to make a copy as `.envrc`.
Utilize [direnv](https://direnv.net/) to keep the `.envrc` sourced
(or run `source .envrc` any time you're running the code).
See [the readme](./README.md#generating-lastfm-api-keys) for info on generating
API keys for last.fm.

## Commits and PRs

Use [Conventional Commits](https://www.conventionalcommits.org/en/v1.0.0/) for
commit messages and PR titles. We use squash commits for PRs. This repo uses
[Release Please](https://github.com/googleapis/release-please) to automate releases.

## Validation

Run `go test ./...` and format with `go fmt` before making changes. Refer to
[.github/workflows/ci.yml](./.github/workflows/ci.yml) to see everything that runs
on PRs.

You'll also want to do manual validation by running the CLI. Ensure you have the
`.envrc` file generated and either use `go build` and `./lastfm-collage-generator`
or `go run .`  with the corresponding flags.

## Documentation

Add concise [godoc](https://go.dev/doc/comment) comments where useful, and update the README when CLI behavior changes.

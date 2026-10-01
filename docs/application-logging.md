# Application logging design

Design for [issue #96](https://github.com/natasha-audrey/lastfm-collage-generator/issues/96), confirmed during the design interview on October 1, 2026.

## Output and configuration

Use Go's `log/slog` consistently across CLI generation, the local server, and artwork processing. Write readable structured text to stderr, leaving command output usable.

Provide a persistent `--log-level` flag for CLI generation and `serve`, accepting `debug`, `info`, `warn`, and `error`, with `info` as the default. Debug verbosity uses this flag rather than a separate debug switch. Invalid values fail immediately with a clear stderr message. No JSON configuration file or JSON output is needed.

The selected threshold applies to all application logs. Explicitly choosing `warn` or `error` suppresses lower-level summaries and lifecycle messages, including the startup URL.

## Attempts and summaries

Emit one completion summary per CLI generation attempt or server generation request, including requests rejected before generation begins. Attach an application-generated `attempt_id` to each attempt's summary, artwork warnings, and debug messages. Keep IDs in logs; exposing them through HTTP responses is outside this change.

Summaries include an explicit outcome and duration, with HTTP status where applicable and useful available context such as username, listening period, and grid size. Failed attempts carry sanitized diagnostic details in their summary rather than duplicate error records.

Measure duration until the command or request finishes. A timeout is summarized immediately, even if its generation worker is still stopping; subsequent worker cleanup appears at debug level with the same attempt ID. Client cancellation has its own outcome and must not be misreported as HTTP success when no response status was written.

Use these levels:

- Info: success, validation rejection, unknown user, no albums, busy response, and client cancellation.
- Warn: individual artwork fallbacks encountered during the attempt.
- Error: timeout, upstream failure, or internal failure.

## Artwork warnings

Emit an individual warning when an attempt encounters a missing artwork URL, download failure, or decode failure. Include album, artist, reason, and attempt ID, with sanitized diagnostic details where available. Ten such problems produce ten warnings. A successfully produced collage still has a successful summary despite these warnings.

Do not inspect or redesign the cache to rediscover historical artwork failures. Existing cached fallback tiles cannot be distinguished from normal cached artwork. No aggregate fallback count is required by this design.

## Debug progress and server lifecycle

Debug logging covers stage transitions and durations for Last.fm fetching, album parsing, artwork preparation, composition, and PNG writing, plus per-album cache hits and downloads. Include sanitized failure causes, but do not dump request or response bodies.

Retain server startup and shutdown messages, and log unexpected server failures. Startup reports the actual bound address, including the assigned port when port zero is requested. These lifecycle events are separate from per-attempt summaries.

## Credential handling

Credentials are forbidden at every log level. Usernames, artist and album names, and local paths are allowed. Sanitize credential-bearing URLs and error messages before logging them, including Last.fm API-key query parameters and credentials in configured upstream or artwork URLs. Do not dump credential configuration.

The existing CLI can print URL-bearing transport errors directly. Implementation must cover that stderr path as well as structured log attributes. Preserve useful internal failure causes while keeping HTTP error responses appropriately generic.

## Verification and documentation

Document the conventions and flag in the README. Verify representative CLI and server successes and failures, rejected generation requests, cancellation and timeout behavior, artwork warnings, attempt correlation, level filtering, valid structured attributes, and credential exclusion. Confirm that port-zero startup reports the bound address and that logs use stderr. Run `go test ./...` before committing or pushing.

This change preserves the private local-server and shared-cache scope of ADR 0001. It does not require a separate ADR: the logging choices are readily reversible and do not introduce architectural lock-in.

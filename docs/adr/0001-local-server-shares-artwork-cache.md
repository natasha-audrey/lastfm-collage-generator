# Local server shares the CLI artwork cache

The initial server is a private, local tool with one active generation at a time. It deliberately reuses the CLI's `./generated` artwork cache without cross-process locking or atomic writes: manual deletion of damaged cache files is acceptable for this use. Cancellation or overlapping CLI/server runs can leave damaged files; revisit cache isolation and coordination before supporting multiple users or hosting the service.

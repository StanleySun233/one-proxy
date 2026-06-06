# Product Requirements: CLI Connection Monitor

## Requirement

`onep monitor <program>` must collect connection information for the launched program and its child processes. The command should not expose implementation mode flags such as `--native`; monitor is the product-level connection audit command.

## Scope

- Record connection events to `time-app.log` in the current working directory.
- Print recorded events to the console when they are written.
- On Windows, Linux, and macOS, collect process, PID, protocol, local endpoint, remote endpoint, and domain fields when available.
- Fail clearly when the platform implementation is not available.

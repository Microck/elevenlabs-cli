# Verification and coverage

This file records the current implementation as of 2026-10-29. It distinguishes
successful browser captures, fixture tests and live CLI execution.

## Working command families

| Commands | Evidence |
| --- | --- |
| auth login/status/refresh | Live Galleton import, renewal, account check and explicit refresh passed with an exported browser session |
| auth logout, doctor | Original direct auth worked live; real-daemon fixture tests cover managed session deletion and account checks |
| profile list/use/remove, config path/list/get/set | Isolated command tests cover saved selection, explicit override, removal and malformed config |
| me show/subscription, model list | Live API success |
| voice list/search/collections/show/settings/defaults | Live API success, including settings and history-era capture follow-up |
| history list/search | Live API success; page envelope preserved |
| history show/download | Live download passed and decoded audio matched the generated sample; browser capture and local HTTP tests cover detail, binary stdout and failures |
| speech generate | Live released CLI produced a valid 0.679-second MP3 from a three-character sample |
| dialogue generate | Successful Windows browser capture and exact payload fixture tests; direct live CLI unverified |
| sound generate/download | Successful Windows browser capture; fixture tests cover payload and Explore sharing guard; direct live CLI unverified |
| skills list/get, auth export-script, completion | Offline command tests |

## Authentication

The Galleton adapter reproduces the captured Firebase form exchange, including
the app referrer and the exported browser user agent. Tests run the real v0.1.1
daemon and SDK against a local provider that rotates refresh credentials. They
verify renewal before requests, encrypted vault storage, recovery after engine
restart and API failure, origin restrictions, failed-login preservation, session
replacement/deletion, and token-free local profiles. Audio tests obtain real
SDK bearer headers and then stream from a local HTTP server.

The lifecycle test starts the managed host from a source build, shares leases,
refuses shutdown with an active client, drains on stop, and restarts from its
saved state. No real account or operator engine is used in these tests. Live
Galleton login, renewal and explicit refresh also passed separately with the
owner-supplied browser export and released v0.1.0 Linux arm64 binary.

## Windows capture

The supplied archive contains 234 sanitized request records and four playable
samples: single-voice speech, dialogue, sound effects and a Studio export. All
manifest SHA-256 hashes were verified on import. The retained capture lives in
`artifacts/windows-capture`, outside the source package.

Successful speech bodies use `/v1/text-to-dialogue/stream`, model
`eleven_v4_exp`, text/voice inputs and settings. The retained successful bodies
contain no CAPTCHA field. Header names were observed, but their necessity was
not independently tested. This is browser evidence, not proof of direct replay.

## Service restriction

The research account/connection returned HTTP 401 with
`detail.status=detected_unusual_activity` after a minimal direct CLI generation.
The CLI returned exit 4 and removed its newly created output file. Earlier
browser attempts on the original machine, MacBook and local Camoufox displayed
the same restriction. Using Camoufox's normal user agent did not remove it.
The user's existing Windows Helium session generated successfully. A later
CLI check using that account's exported Firebase session also generated speech
successfully from this research machine. Its history download decoded to exactly
the same audio. No retry or account upgrade was needed.

This does not establish which combination of account, network and browser
caused the restriction. No paid upgrade or automatic generation retry was made.

## Remaining coverage

The Windows capture verified Studio isolation/voice-changing operations and an
early export, but project-creation multipart values were not retained. Later
storage downloads were blocked by the browser. Those workflows are not exposed
as complete CLI commands.

Music generation appeared in history, but its terminal SSE messages were not
retained and downloading was paywalled. Standalone uploads could not be attached
by the capture tool. Dubbing generation was not submitted because no non-cloning
mode was established. Dedicated cursor traversal remains unverified.

A full implementation still needs verified upload, asynchronous completion,
project lifecycle, music and dubbing contracts. The API can change independently
of this project. No claims of full feature parity are made.

## Source-project checks

`make check` validates Go code, unit/integration tests, formatting, and generated
command documentation. `npm run check` validates the small Node installer/launcher.
The packed source installation is exercised in an isolated prefix with disposable
config; it never uses a real browser session. Release publication and package
registry registration are not part of these checks.

## Source installation checks

The source project passed `make check` (vet, all tests, generated command docs
and formatting) in a disposable Raft workspace. `make install` with a temporary
GOBIN and a fresh npm installation from the packed source both succeeded.
Before the Galleton change, the npm-installed command passed session export/import, profile selection,
status and config reads, shell completion, embedded guide output, invalid-command
exit 2, profile removal and missing-session exit 3, using disposable fixture
credentials without network requests. Node script syntax checks passed.

## Galleton replacement checks

The complete `make check` passed in a disposable Raft workspace after the
Galleton change, including the real-engine auth and lifecycle tests described
above. Node syntax checks and installation from the packed npm source passed.
The installed package exposes the new managed-login help. The Windows amd64
cross-build passed; native Windows Galleton lifecycle remains untested.

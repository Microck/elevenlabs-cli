# Verification and coverage

This file records the current implementation as of 2026-10-29. It distinguishes
successful browser captures, fixture tests and live CLI execution.

## Working command families

| Commands | Evidence |
| --- | --- |
| auth import/status/refresh/logout, doctor | Live account import and renewal worked; isolated tests cover persistence, errors and local removal |
| profile list/use/remove, config path/list/get/set | Isolated command tests cover saved selection, explicit override, removal and malformed config |
| me show/subscription, model list | Live API success |
| voice list/search/collections/show/settings/defaults | Live API success, including settings and history-era capture follow-up |
| history list/search | Live API success; page envelope preserved |
| history show/download | Successful Windows browser capture; local HTTP tests cover download request, binary stdout and failures |
| speech/dialogue generate | Successful Windows browser capture; exact payload fixture tests; live research CLI receives service restriction |
| sound generate/download | Successful Windows browser capture; fixture tests cover payload and Explore sharing guard; direct live CLI unverified |
| skills list/get, auth export-script, completion | Offline command tests |

## Authentication

The Firebase refresh endpoint requires the ElevenLabs web referrer. Tests check
that header and preservation of the imported browser user agent. The session
schema requires `user_agent`; exports made before that field was added must be
recreated. No historical-state conversion or guessed browser identity is used.

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
The user's existing Windows Helium session generated successfully.

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
The npm-installed command passed session export/import, profile selection,
status and config reads, shell completion, embedded guide output, invalid-command
exit 2, profile removal and missing-session exit 3, using disposable fixture
credentials without network requests. Node script syntax checks passed.

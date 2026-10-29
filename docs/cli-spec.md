# CLI contract

`elevenlabs` operates the ElevenLabs web API using an imported browser session.
It follows wallapop-cli and vinted-cli: Go, Cobra, JSON by default, noun/verb
commands, named profiles, and diagnostics on stderr.

## Output and execution

- `--format json|jsonl|pretty` selects JSON, one list item per line, or indented JSON.
- `--error-format text|json` controls errors on stderr.
- `--profile NAME` overrides the saved profile selection, initially `default`.
- `--no-input` guarantees no prompts. Initial commands never prompt.
- `--debug` logs method, path, status and duration, never headers or payloads.
- Help and version work offline without credentials.
- Exit codes: 0 success, 1 local/network failure, 2 usage, 3 authentication,
  4 permission/plan restriction, 5 rate limit, 6 remote API error, 130 interrupted.
- No automatic retries of generation or mutation requests.
- Audio downloads require `--output FILE` or `--output -`, refuse existing
  files, and keep binary stdout free of JSON.

## Account and discovery

- `auth login --session FILE` imports the canonical session JSON from the browser
  export script. `--session -` reads stdin. Never accept passwords or tokens as flags.
- `auth status` shows local presence and expiry without revealing secrets.
- `auth refresh` asks Galleton to renew the web session in its encrypted vault.
- `auth logout --yes` forgets the managed session and removes the chosen local profile.
- `profile list` lists local profile names, never token values.
- `doctor` checks session renewal and account access.
- `me show`, `me subscription`, `model list`, `voice list` use observed web routes.

Profile references live under the OS user config directory in `elevenlabs-web-cli`.
`ELEVENLABS_CONFIG_DIR` overrides that directory for disposable tests. Directories
use mode 0700 and files use 0600 on Unix; Windows uses directory ACLs.
Credential rotation must survive command failure.

## Feature admission

Generation, history, voice management, transcription, isolation, dubbing, studio,
music and agent commands are added only after observing their request contract.
A visible feature is not proof of free-account entitlement. Record blocked and
untested features in the evidence report. Never publish voice clones, make phone
calls, add billing methods, or upgrade a plan during verification.

## Browsing

- `voice search TEXT --page-size N --page N --language CODE --sort SORT` searches
  the shared voice library. Return the complete page envelope so
  has_more and total_count are preserved. Page size must be 1-100; page >= 0.
- `voice collections` lists the account's voice collections.
- `history list --page-size N --source SOURCE` returns the history
  page envelope. Page size must be 1-100. Omitted source returns all sources;
  the command requests the first page. No implicit pagination or retries.

HTTP 401 with detail.status=detected_unusual_activity is an account/connection
restriction (exit 4), not an expired session. Report it without recommending a
fresh login or automatic retry.

## Build and use

Requires Go 1.25 or newer. Run `make build`; the executable is `bin/elevenlabs`.
Run `make check` for vet, tests and formatting checks.

Sign into elevenlabs.io in a browser, then run `scripts/export-session.js` in
that tab's developer console. Treat the downloaded JSON as a password.

```sh
bin/elevenlabs auth login --session /private/path/elevenlabs-session.json
bin/elevenlabs doctor
bin/elevenlabs me subscription --format pretty
bin/elevenlabs model list
bin/elevenlabs voice list --format jsonl
bin/elevenlabs voice search narrator --page-size 10
bin/elevenlabs history list
```

Use `--profile NAME` on import and subsequent commands for multiple accounts.
`auth logout --yes` deletes only that local profile; it does not revoke the
browser session. Unknown commands, including commands inside a group, return
exit 2. A group without a subcommand shows help.

## Verification, 2026-10-29

The isolated Go checks pass: `go vet`, all tests, and `gofmt` validation.
The built Linux executable passed live checks for doctor, account, subscription,
models (11), voices (21), collections, shared-voice search and history.
The unknown-subcommand workflow returned exit 2 and a JSON error as specified.
Detailed command results are in `artifacts/live-cli-checks.json`.

Browser request logs are sanitized and omit authorization headers. These logs
are observations, not a guarantee that every recorded route is supported by
the CLI. Speech generation was denied on both the original browser and MacBook
with HTTP 401 `detected_unusual_activity`. The later Windows capture below verifies successful browser generation. Direct
CLI generation must still be tested with an account and connection that permit it.

Local Camoufox authentication also succeeded. Clicking Generate in that browser
produced the same free-tier restriction in the UI, with no generated audio.
The Camoufox in-page recorder did not capture the generation request, so no
HTTP status is asserted for that run. Screenshot: artifacts/camoufox-editor.png.

## Commands from the Windows capture

The verified Windows archive is preserved under `artifacts/windows-capture`.
Its manifest hashes match. Browser success is distinct from direct CLI success.

- `speech generate --text-file FILE --voice ID --output FILE` accepts UTF-8 text;
  `--text-file -` reads stdin. Uses the captured dialogue endpoint with one input.
- `dialogue generate --input FILE --output FILE` accepts a JSON array of
  `{text, voice_id}` entries, or stdin with `--input -`.
- Both default to `--model eleven_v4_exp`, `--stability 0.5`, `--similarity 0.75`.
  `--text-focus` is false by default. These commands consume account credits.
- `history show ID` returns item details. `history list --search TEXT` searches.
- `history download ID --output FILE` downloads one item as MP3.
- `voice show ID`, `voice settings ID`, `voice defaults` return captured metadata.
- `sound generate --text-file FILE --duration SECONDS` returns the generation
  JSON envelope. Defaults: four variants, model eleven_text_to_sound_v3,
  prompt influence 0.3, loop false. It consumes credits. The command checks
  that Explore sharing is disabled and otherwise stops with instructions.
- `sound download ID --output FILE` downloads an Opus generation.

Audio commands require explicit output. `--output -` writes only audio bytes to
stdout; file output prints JSON with path and byte count. Files are opened with
exclusive creation before requesting audio, preventing overwrite and avoiding
credit consumption when the destination already exists. Failed downloads remove
only their own incomplete file. A partial stdout stream cannot be recalled.
No generation request is retried. Empty or non-audio responses are errors.
Input text must be nonempty UTF-8; dialogue JSON must match the stated array.

### Generation examples

```sh
printf 'Hi.' | bin/elevenlabs speech generate --text-file - --voice Xb7hH8MSUJpSbSDYk0k2 --output speech.mp3
bin/elevenlabs dialogue generate --input dialogue.json --output dialogue.mp3
bin/elevenlabs history list --source TTS --search 'Hi.'
bin/elevenlabs history show HISTORY_ID
bin/elevenlabs history download HISTORY_ID --output history.mp3
bin/elevenlabs voice settings onwK4e9ZLuTAKqWW03F9
bin/elevenlabs sound generate --text-file prompt.txt --duration 1 --variants 1
bin/elevenlabs sound download SOUND_ID --output sound.opus
```

`dialogue.json` contains:

```json
[
  {"text":"Hi.","voice_id":"Xb7hH8MSUJpSbSDYk0k2"},
  {"text":"Hello.","voice_id":"onwK4e9ZLuTAKqWW03F9"}
]
```

On Windows, use `elevenlabs.exe` with the same arguments and text files.
These stock voice IDs and request defaults come from the supplied capture.
No claim is made that additional models or parameter combinations were exercised.
Music, Studio creation/export, transcription submission, standalone transforms
and dubbing generation are not implemented yet. The capture's missing multipart
values, asynchronous contracts and plan restrictions remain explicit gaps.

### Direct CLI verification after implementation

All Go tests, vet and formatting checks pass. Live voice details, settings,
defaults and history search succeeded from the research account. A single
`Hi.` speech-generation request using the captured model returned the existing
free-tier restriction (exit 4); the reserved output file was removed.
Generation/download handlers are tested with local HTTP fixtures, but successful
direct CLI generation and history/sound download against the user's working
Windows session remain unverified. The Windows amd64 executable starts and
prints its version on the user's Windows 10 computer.

## Project installation and local configuration

This is a Go/Cobra source project, installed as `elevenlabs`. `make install`
uses Go's standard installation directory; `go install ./cmd/elevenlabs` works
from the checkout. The npm source package also installs `elevenlabs`, compiling
its bundled Go source during installation. It requires Go 1.25+ and Node 22+;
there is no published npm release or remote installer URL yet.

`profile use NAME` selects an already imported profile. Explicit `--profile`
overrides the saved selection. With no config, the selection is `default`.
`profile remove NAME --yes` deletes only that profile's session. Removing or
logging out the selected profile leaves the selection intact, so the next
account command reports that authentication is required; no account is silently
selected instead.

`config path`, `config list`, `config get default_profile`, and
`config set default_profile NAME` manage `config.json` in ELEVENLABS_CONFIG_DIR
or the OS user config directory. This file contains no tokens. A missing config
means default settings. Malformed configuration is an error, never silently
replaced. Config path, help, version and completion work without authentication.

`auth export-script` prints the browser export script, so installation does not
require keeping a checkout. Session JSON now includes `user_agent` from the
signed-in browser. Authenticated requests and token refresh preserve that value.
Exports missing it must be recreated; no fabricated browser header is substituted.

`skills list` returns available agent guides as JSON; `skills get elevenlabs`
prints a Markdown guide. `completion bash|zsh|fish|powershell` prints completion
scripts. `make docs` generates docs/commands.md from the Cobra tree;
`make docs-check` verifies that it matches without rewriting files.

## Galleton authentication contract

Galleton v0.1.1 owns Firebase token refresh and encrypted credential storage.
Login imports the browser JSON into a new engine session, renews it, verifies
account access, and only then saves a profile reference. A failed replacement
keeps the previous profile. Local profiles contain the session ID, browser user
agent and Firebase application key, never access or refresh tokens.

The CLI obtains short-lived bearer headers through the official Go SDK, as
wallapop-cli does. Audio keeps streaming directly from ElevenLabs. The engine
serializes refresh across processes. Authentication uses the existing browser
export because ElevenLabs stores Firebase credentials in localStorage rather
than marketplace cookies. No password or manual daemon setup is needed.

`auth status` reads local metadata only; `auth status --check` checks Galleton
and account access. `auth service status` inspects the engine without starting
it; `auth service stop` stops it when idle. Commands lease the engine and it
stops ten seconds after the last client exits. No OS startup service is installed. Profile mutation locks serialize login/logout;
retired session IDs remain recorded until Galleton confirms deletion.
Release archives, `make install` and local npm installs embed the pinned engine.
Plain `go install` builds cache the engine on first authenticated use, verifying
its module checksums. Go is required for that first preparation.
Old plaintext profile files require a fresh login; no legacy auth path remains.

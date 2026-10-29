# elevenlabs-web-cli

`elevenlabs` is a terminal client for the ElevenLabs web app. It imports your
browser session, renews it when needed, and exposes account, voice, speech,
dialogue, sound-effect and history commands. Output is JSON by default for
scripts and agents; `--format pretty` is for reading in a terminal.

The project follows the Go/Cobra, noun/verb, profile and session-import
conventions used by wallapop-cli and vinted-cli. It uses the web API rather than
an ElevenLabs developer API key. It is an unofficial client.

## Install from this checkout

Requires Go 1.25 or newer.

```sh
make install
elevenlabs --version
elevenlabs --help
```

Go installs the command in `GOBIN`, or `$(go env GOPATH)/bin` when GOBIN is unset.
Add that directory to PATH. `go install ./cmd/elevenlabs` is equivalent; `make
build` keeps a development build in `bin/elevenlabs`.

For npm installation, also install Node 22 or newer:

```sh
npm install -g .
elevenlabs --version
```

The npm package compiles the included Go source during installation. It has no
runtime npm dependencies. This checkout is not published to npm, Homebrew,
Scoop or GitHub Releases; these commands do not depend on a remote release.

## Sign in

Sign into ElevenLabs in your normal browser. Print the export script:

```sh
elevenlabs auth export-script
```

Run that script in the developer console of your signed-in ElevenLabs tab. It
downloads a session JSON file containing your tokens and the browser's actual
user agent. Import it without putting secrets on the command line:

```sh
elevenlabs auth login --session /path/to/elevenlabs-session.json
elevenlabs auth status
elevenlabs doctor
```

`--session -` reads the JSON from stdin. Keep the export private and delete your
copy when it is no longer needed. Never commit it. Old test exports without
`user_agent` must be recreated with the current script.

The CLI stores sessions under your OS user config directory in
`elevenlabs-web-cli/profiles`. Files use mode 0600 on systems that support Unix
permissions. Windows uses the containing directory's ACL. Set
`ELEVENLABS_CONFIG_DIR` to isolate another installation or test environment.

Tokens renew before authenticated requests when expiry approaches. The CLI
saves the renewed credentials before sending the API request. It does not
install a daemon, background service or scheduled task.

## Use the CLI

```sh
elevenlabs me subscription --format pretty
elevenlabs model list
elevenlabs voice list --format jsonl
elevenlabs voice search narrator --language en --page-size 10
elevenlabs voice show Xb7hH8MSUJpSbSDYk0k2
elevenlabs voice settings onwK4e9ZLuTAKqWW03F9

elevenlabs speech generate --text-file narration.txt \
  --voice Xb7hH8MSUJpSbSDYk0k2 --output narration.mp3

elevenlabs dialogue generate --input dialogue.json --output dialogue.mp3
elevenlabs history list --source TTS --search 'Hello'
elevenlabs history show HISTORY_ID
elevenlabs history download HISTORY_ID --output saved.mp3

elevenlabs sound generate --text-file prompt.txt --duration 1 --variants 1
elevenlabs sound download SOUND_ID --output effect.opus
```

Dialogue input is a JSON array:

```json
[
  {"text": "Hello.", "voice_id": "Xb7hH8MSUJpSbSDYk0k2"},
  {"text": "Hi there.", "voice_id": "onwK4e9ZLuTAKqWW03F9"}
]
```

Generation consumes credits. Sound generation requires Explore sharing to
already be disabled in your account. Audio commands require `--output FILE` or
`--output -` for binary stdout. Existing files are never overwritten, and failed
downloads remove their incomplete output. Generation is never retried.

## Profiles and configuration

```sh
elevenlabs auth login --profile work --session work-session.json
elevenlabs profile list
elevenlabs profile use work
elevenlabs me show
elevenlabs me show --profile default

elevenlabs config path
elevenlabs config list
elevenlabs config get default_profile
elevenlabs config set default_profile work

elevenlabs auth logout --yes
elevenlabs profile remove work --yes
```

An explicit `--profile` overrides the saved selection. Removing a profile only
removes its local credentials; it does not revoke browser sessions or switch
to another account. Config contains only settings, not credentials.

## Agents and shell completion

```sh
elevenlabs skills list
elevenlabs skills get elevenlabs
elevenlabs completion bash
elevenlabs completion zsh
elevenlabs completion fish
elevenlabs completion powershell
```

Global flags include `--profile`, `--format json|jsonl|pretty`,
`--error-format text|json`, `--no-input` and `--debug`. Commands never prompt.
Errors and request diagnostics go to stderr; debug logs omit headers and bodies.
Paginated JSON retains its page envelope. Lists currently fetch one page.

Exit codes are 0 for success, 1 for local/network errors, 2 for usage, 3 for
invalid authentication, 4 for permission/plan/connection restrictions, 5 for
rate limits, 6 for other API errors, and 130 for interruption.

## Development

```sh
make build
make docs
make check
npm run check
npm pack --ignore-scripts
```

`make check` runs vet, tests, generated-reference verification and formatting
checks. Tests use disposable files and local HTTP fixture servers. CI also
installs the packed npm source into an isolated prefix and exercises its command.
`.goreleaser.yaml` defines cross-platform snapshot builds; publishing is disabled.

- [Generated command reference](docs/commands.md)
- [CLI contract](docs/cli-spec.md)
- [Verification and coverage](docs/verification.md)

## Current limits

Successful Windows browser captures establish the generation and download
request shapes. Live CLI account and browsing commands work. Direct CLI
speech generation from the research connection receives ElevenLabs'
`detected_unusual_activity` restriction; successful direct generation on the
working Windows session is still unverified. Login success does not prove
credit-generating operations are allowed.

Music, Studio, transcription uploads, standalone voice transformations, dubbing,
voice cloning and agent execution are not implemented. Some appeared in the
browser capture, but their complete CLI workflows remain unverified. This is
not full ElevenLabs feature parity.

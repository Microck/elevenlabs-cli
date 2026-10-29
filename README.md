<p align="center">
  <img src=".github/assets/elevenlabs-cli-logo.svg" width="720" alt="elevenlabs cli">
</p>

<p align="center">
  <a href="https://github.com/Microck/elevenlabs-web-cli/releases">releases</a> ·
  <a href="https://elevenlabs.micr.dev/docs/">documentation</a> ·
  <a href="docs-site/content/docs/examples.mdx">examples</a> ·
  <a href="docs/commands.md">command reference</a>
</p>

---

`elevenlabs` is a terminal cli for the ElevenLabs web app. browse voices, generate speech and dialogue, create sound effects, and download audio from your own account. output is json by default so it works with `jq`, scripts and agents. `--format pretty` is for reading in a terminal.

## why

if you already use ElevenLabs and want the same account in your terminal, this cli gives you commands for the audio workflows without a developer api key. Galleton manages the imported browser session, and audio streams straight to a file or stdout.

## quickstart

### release downloads

download the archive for your system from [releases](https://github.com/Microck/elevenlabs-web-cli/releases).

```sh
# example: linux arm64
gh release download --repo Microck/elevenlabs-web-cli \
  --pattern '*_linux_arm64.tar.gz' --pattern checksums.txt
sha256sum --ignore-missing --check checksums.txt
tar -xzf elevenlabs_0.1.0_linux_arm64.tar.gz
./elevenlabs --version
```

archives cover linux, macos and windows on amd64 and arm64. windows downloads are zip files. extract the executable into a directory on PATH. each release includes the pinned Galleton engine, so no Go installation or separate daemon setup is needed. macos builds are not notarized; windows builds are not code-signed.

### building from source

requires Go 1.25 or newer:

```sh
gh repo clone Microck/elevenlabs-web-cli
cd elevenlabs-web-cli
make install
elevenlabs --version
```

Go installs into `GOBIN`, or `$(go env GOPATH)/bin` by default. `make install` bundles Galleton. plain `go install ./cmd/elevenlabs` prepares the same pinned engine on first authenticated use.

with Node 22+ and Go installed, `npm install -g .` installs from this checkout. there is no public npm, Homebrew, Scoop or AUR package yet.

### auth

ElevenLabs keeps its Firebase login in browser local storage. a cookie-only export is not enough.

1. sign into ElevenLabs in your regular browser.
2. run `elevenlabs auth export-script` and paste the script into that tab's developer console.
3. import the downloaded session file:

```sh
elevenlabs auth login --session ~/Downloads/elevenlabs-session.json
elevenlabs auth status --check
elevenlabs me subscription --format pretty
```

`--session -` reads from stdin. keep the export private and delete it when it is no longer needed. Galleton stores the tokens in an encrypted vault; local profiles store only a session reference and non-secret settings. the imported browser user agent is preserved.

```sh
elevenlabs auth status
elevenlabs auth refresh
elevenlabs auth logout --yes
```

the engine starts when needed and stops ten seconds after its last client exits. no OS startup service is installed. `auth service status` inspects it; `auth service stop` stops an idle engine. see [session lifecycle and recovery](docs-site/content/docs/galleton.mdx).

## command surface

| command | what it does |
| --- | --- |
| `auth login / status / refresh / logout` | import, inspect, renew or forget a managed session |
| `auth export-script` | print the browser session export script |
| `auth service status / stop` | inspect or stop the local engine |
| `profile list / use / remove` | keep multiple accounts separate |
| `me show / subscription` | account details, plan and credit allowance |
| `model list` | available models and capabilities |
| `voice list / search / collections / show / settings / defaults` | browse voices and settings |
| `speech generate` | generate speech from a text file |
| `dialogue generate` | generate dialogue from text and voice pairs |
| `sound generate / download` | create and download sound effects |
| `history list / show / download` | find, inspect and download generated audio |
| `config path / list / get / set` | local settings and default profile |
| `doctor` | check session renewal and account access |
| `skills list / get` | embedded instructions for agents |
| `completion bash\|zsh\|fish\|powershell` | shell completion |

global flags: `--profile`, `--format json|jsonl|pretty`, `--error-format text|json`, `--no-input`, `--debug`.

## examples

```sh
elevenlabs voice search narrator --language en --page-size 10
elevenlabs voice list --format jsonl
elevenlabs voice settings onwK4e9ZLuTAKqWW03F9

elevenlabs speech generate --text-file narration.txt \
  --voice Xb7hH8MSUJpSbSDYk0k2 --output narration.mp3

elevenlabs dialogue generate --input dialogue.json --output dialogue.mp3
elevenlabs history list --source TTS --search Hello
elevenlabs history download HISTORY_ID --output saved.mp3

elevenlabs sound generate --text-file prompt.txt --duration 1 --variants 1
elevenlabs sound download SOUND_ID --output effect.opus
```

`dialogue.json` contains an array of text and voice pairs:

```json
[
  {"text":"Hello.","voice_id":"Xb7hH8MSUJpSbSDYk0k2"},
  {"text":"Hi there.","voice_id":"onwK4e9ZLuTAKqWW03F9"}
]
```

generation consumes account credits and is never retried automatically. sound generation requires Explore sharing to already be disabled in the website. audio commands require `--output FILE` or `--output -` for binary stdout. existing files are never overwritten.

## profiles

```sh
elevenlabs auth login --profile work --session work-session.json
elevenlabs profile use work
elevenlabs me show
elevenlabs me show --profile default
elevenlabs config list
elevenlabs profile remove work --yes
```

`--profile` overrides the saved default. removing a profile forgets its managed session without signing out your browser or silently switching accounts. `ELEVENLABS_CONFIG_DIR` overrides the OS user config directory.

## shell completion

```sh
# bash: load for this shell
source <(elevenlabs completion bash)

# fish
mkdir -p ~/.config/fish/completions
elevenlabs completion fish > ~/.config/fish/completions/elevenlabs.fish

# zsh: save into a writable directory in fpath
elevenlabs completion zsh > "${fpath[1]}/_elevenlabs"
```

```powershell
elevenlabs completion powershell >> $PROFILE
```

## exit codes

`0` success, `1` local/network failure, `2` usage, `3` authentication, `4` permission or connection restriction, `5` rate limit, `6` remote api error, `130` interrupted. errors go to stderr; `--error-format json` makes them machine-readable.

## current limits

live checks with an exported browser session passed Galleton login, token renewal, subscription and voice reads, speech generation, and history download. the downloaded audio matched the generated sample. fixture tests also cover credential rotation and engine recovery. direct live dialogue and sound generation remain unverified.

successful browser captures establish the speech, dialogue and download request shapes. the research connection receives `detected_unusual_activity` during generation. the cli reports that restriction without retrying; a valid login does not guarantee generation access.

music, Studio, transcription uploads, standalone voice transformations, dubbing, voice cloning and agent execution are not implemented. see [verification and coverage](docs/verification.md).

## development and releases

```sh
make build       # bundled development build in bin/
make check       # vet, real-engine tests, docs and formatting
make docs        # regenerate both command references
```

the Fumadocs site is in `docs-site`. it includes installation, auth, examples, profiles, output, troubleshooting and the generated command reference. see [docs-site/README.md](docs-site/README.md) to build it.

version tags trigger the [release workflow](.github/workflows/release.yml). it checks the code, builds six platform archives with the pinned engine, and publishes a stable GitHub release with checksums. see [release instructions](docs-site/content/docs/releases.mdx).

## disclaimer

this is an unofficial client and is not affiliated with ElevenLabs. web endpoints can change independently of the cli. protect your exported session and only generate content you are authorized to create.

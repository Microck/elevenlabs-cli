## ElevenLabs web CLI v0.1.1

Install the CLI through npm, Homebrew, Scoop or AUR:

```sh
npm install -g elevenlabs-web-cli
brew install --cask Microck/tap/elevenlabs
scoop install microck/elevenlabs
paru -S elevenlabs-web-cli-bin
```

Add the Scoop bucket first with `scoop bucket add microck https://github.com/Microck/scoop-bucket`.

The npm installer downloads the matching release archive and checks its SHA-256. Galleton is bundled, so Go is not required. npm needs Node 22+ and enabled install scripts.

Live verification passed Galleton login and token renewal, speech generation, and history download with matching decoded audio. Direct live dialogue and sound-effects generation remain unverified. This release does not change CLI commands or authentication behavior.

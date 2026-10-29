## ElevenLabs web CLI v0.1.0

Use your ElevenLabs web account from the terminal with JSON output and named profiles.

- Browse account credits, models, voices and generation history.
- Generate speech, dialogue and sound effects; download audio without overwriting existing files.
- Import a browser session into Galleton's encrypted vault, with automatic token renewal and a shared local engine.
- Read embedded agent help, generate shell completions and use the generated command reference.

Downloads support Linux, macOS and Windows on amd64 and arm64. Every archive bundles the pinned Galleton engine. Verify downloads against `checksums.txt`.

### Known limits

Generation can be refused by ElevenLabs' free-tier connection checks. Live checks passed Galleton login and renewal, speech generation, and history download with matching decoded audio. Direct live dialogue and sound generation remain unverified. Cookie-only exports cannot authenticate; use `elevenlabs auth export-script` to export the Firebase session.

Music, Studio, transcription uploads, standalone voice transformations, dubbing, voice cloning and agent execution are not implemented. macOS binaries are not notarized and Windows binaries are not code-signed.

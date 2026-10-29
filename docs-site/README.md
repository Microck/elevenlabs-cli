# Documentation site

Fumadocs and Next.js static documentation, following the wallapop-cli and
vinted-cli structure, with system fonts and neutral light/dark colors matched
to https://elevenlabs.io/docs/overview. The site has sidebar navigation, search and `llms.txt`.

```sh
npm ci
npm run lint
npm run typecheck
npm run build
npm run verify
npm run dev
```

The development server binds to `0.0.0.0`; open `/`.
Static output is in `out/`. The documentation workflow checks each pull request
and deploys `main` to https://elevenlabs.micr.dev/ using GitHub Pages.

`content/docs/reference.mdx` comes from the Cobra tree. Run `make docs` from the
repository root to update it and `docs/commands.md` together. Do not edit it by
hand. Other MDX files are authored guides. Never copy captures or session exports
into this site.

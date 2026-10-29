import Link from 'next/link';

export default function Home() {
  return <>
    <header className="flex h-[50px] items-center justify-between border-b px-6 lg:px-10">
      <Link href="/" className="font-semibold"><span aria-hidden="true" className="pr-2 font-bold">Ⅱ</span>ElevenLabs CLI</Link>
      <nav className="flex gap-6 text-sm" aria-label="Main navigation">
        <Link href="/docs">Documentation</Link>
        <a href="https://github.com/Microck/elevenlabs-web-cli">GitHub</a>
      </nav>
    </header>
    <main className="mx-auto flex w-full max-w-[848px] flex-1 flex-col gap-6 px-6 py-16 lg:py-24">
      <h1 className="text-4xl leading-tight">ElevenLabs CLI documentation</h1>
      <p className="max-w-2xl text-lg text-fd-muted-foreground">Use your ElevenLabs account from the terminal to browse voices, generate speech and dialogue, and download audio.</p>
      <nav className="flex flex-wrap gap-3 text-sm font-medium" aria-label="Get started">
        <Link className="rounded-lg bg-fd-primary px-4 py-2.5 text-fd-primary-foreground" href="/docs/install">Get started</Link>
        <a className="rounded-lg border px-4 py-2.5" href="https://github.com/Microck/elevenlabs-web-cli/releases">Download the CLI</a>
      </nav>
      <section className="mt-6" aria-labelledby="session-heading">
        <h2 id="session-heading" className="mb-4 text-xl font-semibold">Connect your browser session</h2>
        <pre className="overflow-auto rounded-lg border bg-fd-muted p-5 text-sm leading-7"><code>{'elevenlabs auth login --session elevenlabs-session.json\nelevenlabs voice list --format pretty'}</code></pre>
        <p className="mt-4 text-sm text-fd-muted-foreground">Follow the <Link className="underline underline-offset-4" href="/docs/auth">authentication guide</Link> to export your session. JSON output works with scripts and agents.</p>
      </section>
      <p className="mt-8 border-t pt-6 text-sm text-fd-muted-foreground">This is an unofficial client and is not affiliated with ElevenLabs.</p>
    </main>
  </>;
}

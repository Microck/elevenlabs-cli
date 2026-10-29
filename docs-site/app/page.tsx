import Link from 'next/link';

export default function Home() {
  return <main className="mx-auto flex w-full max-w-3xl flex-1 flex-col justify-center gap-6 px-6 py-20">
    <h1 className="text-4xl font-semibold tracking-tight">elevenlabs-web-cli</h1>
    <p className="text-lg">Use your ElevenLabs account from the terminal to browse voices, generate speech and dialogue, and download audio. JSON output works with scripts and agents.</p>
    <pre className="overflow-auto border-y py-5"><code>{'elevenlabs auth login --session elevenlabs-session.json\nelevenlabs voice list --format pretty'}</code></pre>
    <nav className="flex gap-6" aria-label="Get started"><Link className="underline" href="/docs">Read the docs</Link><a className="underline" href="https://github.com/Microck/elevenlabs-web-cli/releases">Releases</a></nav>
    <p>This unofficial client uses your existing ElevenLabs browser session.</p>
  </main>;
}

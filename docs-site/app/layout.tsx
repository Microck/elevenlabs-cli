import './global.css';
import { RootProvider } from 'fumadocs-ui/provider/next';
import type { ReactNode } from 'react';
import type { Metadata } from 'next';

export const metadata: Metadata = {
  metadataBase: new URL('https://elevenlabs.micr.dev/'),
  title: { default: 'elevenlabs-web-cli', template: '%s | elevenlabs-web-cli' },
  description: 'Use ElevenLabs from the terminal: voices, speech, dialogue and history, with JSON output.',
};

export default function Layout({ children }: { children: ReactNode }) {
  return (
    <html lang="en" className="light" suppressHydrationWarning>
      <body className="flex min-h-screen flex-col">
        <RootProvider theme={{ defaultTheme: 'light', enableSystem: false }} search={{ options: { type: 'static' } }}>
          {children}
        </RootProvider>
      </body>
    </html>
  );
}

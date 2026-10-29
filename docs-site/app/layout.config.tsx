import type { BaseLayoutProps } from 'fumadocs-ui/layouts/shared';

export const baseOptions: BaseLayoutProps = {
  nav: {
    title: (
      <span className="inline-flex items-center gap-2 font-semibold">
        <span aria-hidden="true" className="font-bold tracking-[-0.2em] pr-2">Ⅱ</span>
        <span>ElevenLabs CLI</span>
      </span>
    ),
  },
  githubUrl: 'https://github.com/Microck/elevenlabs-web-cli',
};

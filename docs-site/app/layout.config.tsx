import type { BaseLayoutProps } from 'fumadocs-ui/layouts/shared';

export const baseOptions: BaseLayoutProps = {
  nav: {
    title: (
      <img src="/elevenlabs-cli-logo.svg" alt="ElevenLabs CLI" width={190} height={21} className="h-auto w-[190px] dark:invert" />
    ),
  },
  githubUrl: 'https://github.com/Microck/elevenlabs-cli',
};

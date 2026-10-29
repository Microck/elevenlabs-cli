import { source } from '@/lib/source';
import { llms } from 'fumadocs-core/source';

export const revalidate = false;
export const dynamic = 'force-static';

export async function GET(): Promise<Response> {
  return new Response((await llms(source).index()).replaceAll('](/docs', '](https://microck.github.io/elevenlabs-web-cli/docs'), {
    headers: { 'content-type': 'text/plain; charset=utf-8' },
  });
}

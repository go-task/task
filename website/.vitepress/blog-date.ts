import { existsSync } from 'node:fs';
import { resolve } from 'node:path';

export function createBlogDateResolver(
  channel: string,
  latestBlogDir: string,
  now = new Date()
) {
  const buildDate = now.toISOString().slice(0, 10);
  return (file: string, storedDate: string): Date =>
    new Date(
      channel === 'next' &&
        file !== 'index.md' &&
        !existsSync(resolve(latestBlogDir, file))
        ? buildDate
        : storedDate
    );
}

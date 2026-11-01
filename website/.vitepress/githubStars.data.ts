import { defineLoader } from 'vitepress';

const REPO = 'go-task/task';

// Deploy previews build from shared, unauthenticated Netlify IPs, so a 403 from
// GitHub's 60 requests/hour limit is routine. Falling back keeps the build green
// with a number that is only ever too low.
const FALLBACK_STARS = 16000;

export interface GithubStars {
  count: number;
  label: string;
}

declare const data: GithubStars;
export { data };

export default defineLoader({
  async load(): Promise<GithubStars> {
    const count = await fetchStarCount();
    return { count, label: formatCount(count) };
  }
});

async function fetchStarCount(): Promise<number> {
  const headers: Record<string, string> = {
    Accept: 'application/vnd.github+json'
  };
  if (process.env.GITHUB_TOKEN) {
    headers.Authorization = `Bearer ${process.env.GITHUB_TOKEN}`;
  }

  try {
    const response = await fetch(`https://api.github.com/repos/${REPO}`, {
      headers,
      signal: AbortSignal.timeout(5000)
    });
    if (!response.ok) {
      throw new Error(`GitHub answered ${response.status}`);
    }

    const { stargazers_count } = await response.json();
    if (typeof stargazers_count !== 'number') {
      throw new Error('no stargazers_count in the response');
    }
    return stargazers_count;
  } catch (error) {
    console.warn(
      `Could not read the star count for ${REPO} (${error}), falling back to ${FALLBACK_STARS}.`
    );
    return FALLBACK_STARS;
  }
}

// Rounding is deliberate: a count that never shows its last digits cannot look
// stale between two deployments.
function formatCount(count: number): string {
  return new Intl.NumberFormat('en', {
    notation: 'compact',
    maximumFractionDigits: 1
  })
    .format(count)
    .replace('K', 'k');
}

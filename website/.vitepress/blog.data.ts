import { createContentLoader } from 'vitepress';

export interface Post {
  title: string;
  url: string;
  date: {
    time: number;
    string: string;
  };
  author: string;
  tags: string[];
  excerpt?: string;
}

declare const data: Post[];
export { data };

// extractExcerpt renders the whole document first so that links which have
// `[ref]: url` style definitions outside of the excerpt still resolve. The
// rendered HTML is then cut at the `<!-- more -->` marker leaving just the
// excerpt text.
function extractExcerpt(html: string): string | undefined {
  const markerIndex = html.indexOf('<!-- more -->');
  if (markerIndex === -1) return undefined;
  return html
    .slice(0, markerIndex)
    .replace(/<h1[^>]*>[\s\S]*?<\/h1>/, '')
    .replace(/<AuthorCard\b[^>]*\/?>/gi, '')
    .trim();
}

export default createContentLoader('blog/*.md', {
  render: true,
  transform(raw) {
    return raw
      .filter(({ url }) => url !== '/blog/')
      .map(({ frontmatter, html, url }) => {
        const date = new Date(frontmatter.date);
        return {
          title: frontmatter.title,
          url,
          date: {
            time: date.getTime(),
            string: date.toISOString().slice(0, 10)
          },
          author: frontmatter.author,
          tags: frontmatter.tags ?? [],
          excerpt: html ? extractExcerpt(html) : undefined
        };
      })
      .sort((a, b) => b.date.time - a.date.time);
  }
});

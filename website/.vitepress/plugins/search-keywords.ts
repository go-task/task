import type MarkdownIt from 'markdown-it';

// Keep alternate search terms attached to the heading they describe. HTML
// attributes don't change the visible heading, its permalink or the outline.
export function searchKeywordsPlugin(md: MarkdownIt): void {
  md.core.ruler.push('search-keywords', (state) => {
    const keywords = state.env.frontmatter?.searchKeywords;
    if (keywords === undefined) return;
    const fail = (message: string): never => {
      throw new Error(
        `searchKeywords (${state.env.relativePath ?? 'page'}): ${message}`
      );
    };
    if (!keywords || typeof keywords !== 'object' || Array.isArray(keywords)) {
      fail('expected a mapping of heading IDs to lists of search terms');
    }

    const headings = new Map(
      state.tokens
        .filter((token) => token.type === 'heading_open')
        .map((token) => [token.attrGet('id'), token])
    );
    for (const [anchor, terms] of Object.entries(keywords)) {
      const heading = headings.get(anchor);
      if (!heading) return fail(`heading #${anchor} does not exist`);
      if (
        !Array.isArray(terms) ||
        !terms.length ||
        terms.some((term) => typeof term !== 'string' || !term.trim())
      ) {
        return fail(`#${anchor} needs a non-empty list of strings`);
      }
      heading.attrSet('data-search-keywords', terms.join(' '));
    }
  });
}

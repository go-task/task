import type { MarkdownEnv, MarkdownRenderer } from 'vitepress';

export function renderSearchContent(
  src: string,
  env: MarkdownEnv,
  md: MarkdownRenderer
): string {
  let html = md.render(src, env);
  if (env.frontmatter?.search === false) return '';

  // Only this index renderer adds the alternate terms to section content.
  // The actual page keeps them in an attribute, invisible to readers.
  html = html.replace(
    /(<h[1-6]\b[^>]*\bdata-search-keywords="([^"]*)"[^>]*>[\s\S]*?<\/h[1-6]>)/g,
    '$1<p>$2</p>'
  );

  // Otherwise stripping the tags joins the language label to the first
  // command ("shellbrew"), preventing searches for "brew install".
  html = html.replace(/<span class="lang">[^<]*<\/span>/g, '');

  // Local search renders Markdown without mounting Vue. Include the platform
  // label from each card's props after its heading, so searches such as Ubuntu
  // resolve to apt rather than losing the distribution names with the markup.
  html = html.replace(
    /(<InstallationMethod\b[^>]*\bplatform-label="([^"]*)"[^>]*>[\s\S]*?<\/h3>)/g,
    '$1<p>$2</p>'
  );

  // VitePress 1.x mistakes a linked heading's first <a> for its permalink,
  // leaving an empty title and dropping the section from the local index.
  // Unwrap heading links only in the search renderer; keep the permalink.
  return html.replace(/<h([1-6])\b[^>]*>[\s\S]*?<\/h\1>/g, (heading) =>
    heading.replace(/<a\b([^>]*)>([\s\S]*?)<\/a>/g, (link, attrs, text) =>
      /\bclass="[^"]*\bheader-anchor\b/.test(attrs) ? link : text
    )
  );
}

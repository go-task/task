import type MarkdownIt from 'markdown-it';

interface PluginOptions {
  repo: string;
}

function githubLinksPlugin(
  md: MarkdownIt,
  options: PluginOptions = {} as PluginOptions
): void {
  const baseUrl = 'https://github.com';
  const { repo } = options;

  md.core.ruler.after('inline', 'github-links', (state): void => {
    const tokens = state.tokens;

    for (let i = 0; i < tokens.length; i++) {
      if (tokens[i].type === 'inline' && tokens[i].children) {
        const inlineTokens = tokens[i].children!;

        for (let j = 0; j < inlineTokens.length; j++) {
          if (inlineTokens[j].type === 'text') {
            let text: string = inlineTokens[j].content!;

            const protectedRefs: string[] = [];
            let protectIndex: number = 0;

            text = text.replace(
              /[\w.-]+\/[\w.-]+#(\d+)/g,
              (match: string): string => {
                const placeholder: string = `__PROTECTED_${protectIndex}__`;
                protectedRefs[protectIndex] = match;
                protectIndex++;
                return placeholder;
              }
            );

            text = text.replace(
              /#(\d+)/g,
              `<a href="${baseUrl}/${repo}/issues/$1" target="_blank" class="github-pr-link">#$1</a>`
            );

            text = text.replace(
              /@([a-zA-Z0-9_-]+)(?![\w@.])/g,
              `<a href="${baseUrl}/$1" target="_blank" class="github-user-mention">@$1</a>`
            );

            protectedRefs.forEach((ref: string, index: number): void => {
              text = text.replace(`__PROTECTED_${index}__`, ref);
            });

            if (text !== inlineTokens[j].content) {
              inlineTokens[j].content = text;
              inlineTokens[j].type = 'html_inline';
            }
          }
        }
      }
    }
  });
}

export default githubLinksPlugin;

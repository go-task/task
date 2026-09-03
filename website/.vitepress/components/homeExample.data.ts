import fs from 'node:fs';
import { fileURLToPath } from 'node:url';
import { defineLoader } from 'vitepress';
import { codeToHtml } from 'shiki';

// Same themes VitePress uses for fenced code blocks, so the homepage sample
// picks up the site's own light/dark code colours through `.vp-code`.
const THEMES = { light: 'github-light', dark: 'github-dark' } as const;

const taskfilePath = fileURLToPath(
  new URL('../snippets/homepage-taskfile.yml', import.meta.url)
);
const terminalPath = fileURLToPath(
  new URL('../snippets/homepage-terminal.txt', import.meta.url)
);

export interface HomeExample {
  taskfile: string;
  terminal: string;
}

declare const data: HomeExample;
export { data };

export default defineLoader({
  watch: [
    '../snippets/homepage-taskfile.yml',
    '../snippets/homepage-terminal.txt'
  ],
  async load(): Promise<HomeExample> {
    const taskfile = fs.readFileSync(taskfilePath, 'utf-8').trimEnd();
    return {
      taskfile: await codeToHtml(taskfile, {
        lang: 'yaml',
        themes: THEMES,
        defaultColor: false,
        transformers: [
          {
            pre(node) {
              this.addClassToHast(node, 'vp-code');
            }
          }
        ]
      }),
      terminal: renderTerminal(fs.readFileSync(terminalPath, 'utf-8'))
    };
  }
});

function renderTerminal(source: string): string {
  return source
    .trimEnd()
    .split('\n')
    .map((line) =>
      line.startsWith('$ ')
        ? `<span class="terminal-prompt">$</span>${escapeHtml(line.slice(1))}`
        : escapeHtml(line)
    )
    .join('\n');
}

function escapeHtml(value: string): string {
  return value
    .replace(/&/g, '&amp;')
    .replace(/</g, '&lt;')
    .replace(/>/g, '&gt;');
}

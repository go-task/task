import { existsSync, readFileSync, readdirSync } from 'node:fs';
import { resolve } from 'node:path';
import matter from 'gray-matter';
import { createMarkdownRenderer } from 'vitepress';

// Check the published files: VitePress validates HTML routes, but the LLM
// plugin uses different output paths for directory indexes.
const outDir = resolve(process.argv[2] ?? '.vitepress/dist');
const files = readdirSync(outDir, { recursive: true }).filter(
	(file) => file.endsWith('.md') || /^llms(?:-full)?\.txt$/.test(file)
);
if (!files.length) throw new Error(`No Markdown exports found in ${outDir}`);

const md = await createMarkdownRenderer(process.cwd());
const missing = new Set();
let checked = 0;

function checkTokens(tokens, file) {
	for (const token of tokens) {
		if (token.type === 'link_open') {
			const href = token.attrGet('href');
			const url = new URL(href, `https://docs.invalid/${file}`);
			if (
				url.origin === 'https://docs.invalid' &&
				url.pathname.endsWith('.md')
			) {
				checked++;
				if (
					!existsSync(resolve(outDir, `.${decodeURIComponent(url.pathname)}`))
				) {
					missing.add(`${file}: ${href}`);
				}
			}
		}
		if (token.children) checkTokens(token.children, file);
	}
}

for (const file of files) {
	const { content } = matter(readFileSync(resolve(outDir, file), 'utf8'));
	checkTokens(md.parse(content, {}), file);
}

if (missing.size) {
	throw new Error(`Missing Markdown targets:\n${[...missing].join('\n')}`);
}
console.log(`Checked ${checked} Markdown links in ${files.length} exports.`);

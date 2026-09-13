import assert from 'node:assert/strict';
import { execFile } from 'node:child_process';
import { test } from 'node:test';
import { promisify } from 'node:util';

const run = promisify(execFile);
const inspect = `
  import { resolveConfig } from 'vitepress';
  import { mkdtemp, readFile, rm } from 'node:fs/promises';
  import { tmpdir } from 'node:os';
  import { join } from 'node:path';
  const config = await resolveConfig(process.cwd(), 'build');
  const outDir = await mkdtemp(join(tmpdir(), 'task-search-test-'));
  try {
    await config.buildEnd({ ...config, outDir });
    const heads = [];
    for (const frontmatter of [{}, { search: false }, { noindex: true }]) {
      heads.push(await config.transformHead({pageData: {
        relativePath: 'docs/guide/includes.md', title: 'Including Taskfiles', frontmatter
      }}));
    }
    const search = config.site.themeConfig.search;
    console.log(JSON.stringify({
      provider: search.provider, indexName: search.options.indexName,
      keyOverride: search.options.apiKey === 'test-search-only-key',
      sitemap: config.sitemap.hostname,
      robots: await readFile(join(outDir, 'robots.txt'), 'utf8'), heads
    }));
  } finally { await rm(outDir, { recursive: true }); }
`;

for (const [channel, site, local, key] of [
	['latest', 'production', '', 'test-search-only-key'],
	['next', 'production', '', 'test-search-only-key'],
	['next', 'preview', '', 'test-search-only-key'],
	['latest', 'preview', '', 'test-search-only-key'],
	['next', 'production', '1', 'test-search-only-key'],
	['next', 'production', '', ''],
	['latest', 'production', '', '']
]) {
	test(`${channel}/${site}/local=${local || '0'}/key=${Boolean(key)} selects the intended search and crawl policy`, async () => {
		const { stdout } = await run(
			process.execPath,
			['--input-type=module', '-e', inspect],
			{
				env: {
					...process.env,
					DOCS_CHANNEL: channel,
					DOCS_SITE: site,
					DOCS_LOCAL: local,
					DOCS_ALGOLIA_SEARCH_API_KEY: key
				}
			}
		);
		const result = JSON.parse(stdout.trim());
		const isPublic = site === 'production' && local !== '1';
		const usesAlgolia = isPublic && (channel === 'latest' || Boolean(key));
		assert.equal(result.provider, usesAlgolia ? 'algolia' : 'local');
		assert.equal(
			result.indexName,
			usesAlgolia
				? channel === 'latest'
					? 'taskfile'
					: 'taskfile-next'
				: undefined
		);
		if (usesAlgolia) assert.equal(result.keyOverride, Boolean(key));
		assert.equal(
			result.sitemap,
			channel === 'latest'
				? 'https://taskfile.dev'
				: 'https://next.taskfile.dev'
		);
		const robotsMeta = result.heads[0].find(
			([tag, attrs]) => tag === 'meta' && attrs.name === 'robots'
		);
		if (isPublic && channel === 'latest') {
			assert.equal(robotsMeta, undefined);
			assert.match(result.robots, /User-agent: \*\nAllow: \//);
		} else {
			assert.equal(robotsMeta[1].content, 'noindex, nofollow');
			assert.match(result.robots, /User-agent: \*\nDisallow: \//);
			assert.equal(
				result.robots.includes('User-agent: Algolia Crawler'),
				isPublic
			);
		}
		// Explicit opt-outs are separate from next's global noindex directive.
		for (const head of result.heads.slice(1)) {
			assert.ok(
				head.some(
					([tag, attrs]) =>
						tag === 'meta' &&
						attrs.name === 'docsearch:exclude' &&
						attrs.content === 'true'
				)
			);
		}
		assert.ok(
			!result.heads[0].some(([, attrs]) => attrs.name === 'docsearch:exclude')
		);
	});
}

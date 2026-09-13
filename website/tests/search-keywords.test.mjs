import assert from 'node:assert/strict';
import { readFile } from 'node:fs/promises';
import { test } from 'node:test';
import { runInNewContext } from 'node:vm';
import { load } from 'cheerio';
import { createMarkdownRenderer } from 'vitepress';
import { searchKeywordsPlugin } from '../.vitepress/plugins/search-keywords.ts';
import { renderSearchContent } from '../.vitepress/plugins/local-search.ts';

const md = await createMarkdownRenderer(process.cwd(), {
	config: (md) => md.use(searchKeywordsPlugin)
});
let crawler;
runInNewContext(await readFile('docsearch.config.js', 'utf8'), {
	Crawler: function (config) {
		crawler = config;
	}
});
const source = `---
searchKeywords:
  namespace-aliases: [aliases, namespace aliases]
---
# Including Taskfiles

Reuse tasks.

## Shorten namespace names {#namespace-aliases}

Choose a shorter name.

## Allow a missing file {#optional-includes}

Include a file when present.
`;

test('alternate terms change neither visible headings nor permalinks', () => {
	const $ = load(md.render(source, {}));
	const heading = $('#namespace-aliases');
	assert.equal(
		heading.attr('data-search-keywords'),
		'aliases namespace aliases'
	);
	assert.equal(heading.find('a').attr('href'), '#namespace-aliases');
	assert.equal(
		heading.clone().find('a').remove().end().text().trim(),
		'Shorten namespace names'
	);
	assert.ok(!$('body').text().includes('aliases'));
	assert.equal($('#optional-includes').attr('data-search-keywords'), undefined);
});

test('local search adds alternate terms only to the matching section', () => {
	const $ = load(renderSearchContent(source, {}, md));
	assert.equal(
		$('#namespace-aliases').next('p').text(),
		'aliases namespace aliases'
	);
	assert.equal(
		$('#optional-includes').next('p').text(),
		'Include a file when present.'
	);
});

test('search terms are escaped as data, including HTML and quotes', () => {
	const src = source.replace(
		'[aliases, namespace aliases]',
		JSON.stringify(['<img src=x>', 'a & b', 'a "quote"'])
	);
	const $ = load(renderSearchContent(src, {}, md));
	assert.equal($('img').length, 0);
	assert.equal(
		$('#namespace-aliases').next('p').text(),
		'<img src=x> a & b a "quote"'
	);
});

for (const value of [
	'false',
	'[]',
	'{missing: [aliases]}',
	'{namespace-aliases: aliases}',
	'{namespace-aliases: []}',
	'{namespace-aliases: [123]}',
	'{namespace-aliases: [" "]}'
]) {
	test(`invalid metadata fails the build: ${value}`, () => {
		const src = source.replace(
			'searchKeywords:\n  namespace-aliases: [aliases, namespace aliases]',
			`searchKeywords: ${value}`
		);
		assert.throws(
			() => md.render(src, { relativePath: 'guide/includes.md' }),
			/searchKeywords \(guide\/includes.md\):/
		);
	});
}

test('crawler associates terms with section records, without changing their display', () => {
	const $ = load(
		`<html><body><div class="vp-doc">${md.render(source, {})}</div></body></html>`
	);
	const records = [
		{
			anchor: 'including-taskfiles',
			type: 'lvl1',
			hierarchy: { lvl1: 'Including Taskfiles' }
		},
		{
			anchor: 'namespace-aliases',
			type: 'lvl2',
			hierarchy: { lvl2: 'Shorten namespace names' }
		},
		{
			anchor: 'namespace-aliases',
			type: 'content',
			content: 'Choose a shorter name.'
		},
		{
			anchor: 'optional-includes',
			type: 'lvl2',
			hierarchy: { lvl2: 'Allow a missing file' }
		}
	];
	// The hosted DocSearch helper is the service boundary. Exercise the real
	// extractor and DOM here with representative heading and content records.
	const extracted = crawler.actions[0].recordExtractor({
		$,
		url: new URL('https://taskfile.dev/docs/guide/includes'),
		helpers: { docsearch: () => records }
	});
	assert.deepEqual(
		Array.from(extracted, (record) => record.keywords),
		['', 'aliases namespace aliases', 'aliases namespace aliases', '']
	);
	for (let i = 0; i < records.length; i++) {
		const { keywords, ...record } = extracted[i];
		assert.deepEqual(record, records[i]);
	}
});

test('pages without search metadata remain compatible with the crawler', () => {
	const record = { anchor: 'old-guide', content: 'Existing content.' };
	const extracted = crawler.actions[0].recordExtractor({
		$: load(
			'<div class="vp-doc"><h1 id="old-guide">Guide</h1><p>Existing content.</p></div>'
		),
		url: new URL('https://taskfile.dev/docs/guide'),
		helpers: { docsearch: () => [record] }
	});
	assert.equal(extracted[0].keywords, '');
	assert.equal(extracted[0].content, record.content);
});

test('keywords rank below headings and above prose, and stay out of snippets', () => {
	const settings = crawler.initialIndexSettings.taskfile;
	const attrs = settings.searchableAttributes;
	assert.ok(
		attrs.indexOf('unordered(keywords)') >
			attrs.indexOf('unordered(hierarchy.lvl6)')
	);
	assert.ok(attrs.indexOf('unordered(keywords)') < attrs.indexOf('content'));
	for (const field of [
		'attributesToRetrieve',
		'attributesToHighlight',
		'attributesToSnippet'
	]) {
		assert.ok(!settings[field].some((attr) => attr.startsWith('keywords')));
	}
});

for (const [page, anchor] of [
	['includes', 'namespace-aliases'],
	['defining-tasks', 'task-aliases']
]) {
	test(`the actual ${page} guide exposes aliases on #${anchor}`, async () => {
		const src = await readFile(`src/next/docs/guide/${page}.md`, 'utf8');
		const $ = load(md.render(src, {}));
		assert.match($(`#${anchor}`).attr('data-search-keywords'), /\baliases\b/);
		assert.equal($('[data-search-keywords]').length, 1);
	});
}

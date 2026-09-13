import assert from 'node:assert/strict';
import { readFile } from 'node:fs/promises';
import { test } from 'node:test';
import { runInNewContext } from 'node:vm';
import { load } from 'cheerio';

let crawler;
runInNewContext(await readFile('docsearch.config.js', 'utf8'), {
	Crawler: function (config) {
		crawler = config;
	}
});

// Run the extractor independently, as the hosted crawler may serialize it.
// This catches accidental dependencies on configuration-scope variables.
const extract = runInNewContext(
	`(${crawler.actions[0].recordExtractor.toString()})`
);

test('released and next URLs write to separate indices with shared extraction', () => {
	assert.equal(crawler.actions.length, 2);
	for (const [i, origin, indexName] of [
		[0, 'https://taskfile.dev', 'taskfile'],
		[1, 'https://next.taskfile.dev', 'taskfile-next']
	]) {
		const action = crawler.actions[i];
		assert.equal(action.indexName, indexName);
		assert.deepEqual(Array.from(action.pathsToMatch), [`${origin}/docs/**`]);
		assert.ok(crawler.startUrls.includes(`${origin}/`));
		assert.ok(crawler.sitemaps.includes(`${origin}/sitemap.xml`));
		assert.ok(crawler.discoveryPatterns.includes(`${origin}/docs/**`));
		for (const path of [
			'/docs/changelog**',
			'/docs/**/template',
			'/docs/**/*.md'
		]) {
			assert.ok(crawler.exclusionPatterns.includes(origin + path));
		}
		assert.equal(action.recordExtractor, crawler.actions[0].recordExtractor);
		assert.deepEqual(
			crawler.initialIndexSettings[indexName],
			crawler.initialIndexSettings.taskfile
		);
	}
	assert.equal(crawler.ignoreCanonicalTo, true);
	assert.equal(crawler.ignoreNoIndex, true);
	assert.notEqual(crawler.ignoreRobotsTxtRules, true);
});

test('nested lists, paragraphs, table cells and code contribute each text once', () => {
	const $ = load(`<aside><p>Sidebar noise</p></aside><div class="vp-doc">
    <div data-nosnippet><p>LLM banner</p></div>
    <h1 id="guide">Guide</h1>
    <h2 id="steps">Steps</h2>
    <ul><li>Before <strong>the list</strong>
      <p>First paragraph.</p>
      <ul><li>Nested item.<p>Nested paragraph.</p></li></ul>
      <pre><code>echo unique-command</code></pre>
      After the list.
    </li></ul>
    <h2 id="options">Options</h2>
    <table><tr><th>Column label</th><td><p>Cell paragraph</p><ul><li>Cell item</li></ul></td></tr></table>
    <pre><code>echo standalone-command</code></pre>
    <p>Last paragraph.</p>
  </div>`);
	const headings = $('.vp-doc h1, .vp-doc h2')
		.toArray()
		.map((e) => $(e).attr('id'));
	extract({
		$,
		url: new URL('https://taskfile.dev/docs/guide'),
		helpers: {
			docsearch: ({ recordProps }) => {
				const selected = $(recordProps.content);
				const elements = new Set(selected.toArray());
				for (const element of selected) {
					assert.ok(
						!$(element)
							.parents()
							.toArray()
							.some((parent) => elements.has(parent))
					);
				}
				const text = selected
					.toArray()
					.map((e) => $(e).text())
					.join('\n');
				for (const phrase of [
					'Before the list',
					'First paragraph.',
					'Nested item.',
					'Nested paragraph.',
					'echo unique-command',
					'After the list.',
					'Column label',
					'Cell paragraph',
					'Cell item',
					'echo standalone-command',
					'Last paragraph.'
				])
					assert.equal(text.split(phrase).length - 1, 1, phrase);
				assert.ok(!text.includes('Sidebar noise'));
				assert.ok(!text.includes('LLM banner'));
				assert.deepEqual(
					$('.vp-doc h1, .vp-doc h2')
						.toArray()
						.map((e) => $(e).attr('id')),
					headings
				);
				return [];
			}
		}
	});
});

test('noindex is overridden for next but respected for released pages', () => {
	for (const [origin, expectedCalls] of [
		['https://taskfile.dev', 0],
		['https://next.taskfile.dev', 1]
	]) {
		let calls = 0;
		extract({
			$: load(
				'<meta name="robots" content="noindex, nofollow"><div class="vp-doc"><h1>Guide</h1><p>Text</p></div>'
			),
			url: new URL(`${origin}/docs/guide`),
			helpers: {
				docsearch: () => {
					calls++;
					return [];
				}
			}
		});
		assert.equal(calls, expectedCalls);
	}
});

test('explicit page opt-outs are respected in both channels', () => {
	for (const origin of ['https://taskfile.dev', 'https://next.taskfile.dev']) {
		extract({
			$: load(
				'<meta name="docsearch:exclude" content="true"><div class="vp-doc"><p>Excluded</p></div>'
			),
			url: new URL(`${origin}/docs/guide`),
			helpers: { docsearch: () => assert.fail('excluded page was indexed') }
		});
	}
});

test('the serialized extractor preserves result URLs and attaches section keywords', () => {
	for (const origin of ['https://taskfile.dev', 'https://next.taskfile.dev']) {
		const url = `${origin}/docs/guide/includes#namespace-aliases`;
		const records = extract({
			$: load(
				'<div class="vp-doc"><h2 id="namespace-aliases" data-search-keywords="namespace aliases">Shorten names</h2><p>Text</p></div>'
			),
			url: new URL(url),
			helpers: { docsearch: () => [{ anchor: 'namespace-aliases', url }] }
		});
		assert.equal(records[0].url, url);
		assert.equal(records[0].keywords, 'namespace aliases');
	}
});

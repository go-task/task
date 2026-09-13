import assert from 'node:assert/strict';
import { execFile } from 'node:child_process';
import { mkdtemp, readFile, readdir, rm } from 'node:fs/promises';
import { tmpdir } from 'node:os';
import { join } from 'node:path';
import { fileURLToPath } from 'node:url';
import { test } from 'node:test';
import { promisify } from 'node:util';
import { runInNewContext } from 'node:vm';
import { load } from 'cheerio';

const run = promisify(execFile);
const source = await readFile(
	new URL('../docsearch.config.js', import.meta.url),
	'utf8'
);
let original;
runInNewContext(source, {
	Crawler: function (config) {
		original = config;
	}
});

test('CLI exports deterministic, standalone configuration and matching index settings', async () => {
	const outDir = await mkdtemp(join(tmpdir(), 'task-docsearch-export-'));
	try {
		const script = fileURLToPath(
			new URL('../scripts/export-docsearch.mjs', import.meta.url)
		);
		// It must find its source even when called from a different directory.
		await run(process.execPath, [script, outDir], { cwd: tmpdir() });
		assert.deepEqual((await readdir(outDir)).sort(), [
			'crawler.config.js',
			'taskfile-next.settings.json',
			'taskfile.settings.json'
		]);
		const exported = await readFile(join(outDir, 'crawler.config.js'), 'utf8');
		let generated;
		runInNewContext(exported, {
			Crawler: function (config) {
				generated = config;
			}
		});
		assert.equal(JSON.stringify(generated), JSON.stringify(original));
		assert.equal(generated.apiKey, '<ALGOLIA_CRAWLER_WRITE_KEY>');

		for (const action of generated.actions) {
			const settings = JSON.parse(
				await readFile(
					join(outDir, `${action.indexName}.settings.json`),
					'utf8'
				)
			);
			assert.deepEqual(
				settings,
				JSON.parse(
					JSON.stringify(original.initialIndexSettings[action.indexName])
				)
			);
			// JSON.stringify drops functions. Exercise each exported extractor too,
			// in a fresh context without access to the source's shared variables.
			const extract = runInNewContext(`(${action.recordExtractor.toString()})`);
			const record = {
				anchor: 'aliases',
				url: action.pathsToMatch[0].replace('**', 'includes#aliases')
			};
			const records = extract({
				$: load(
					'<div class="vp-doc"><h2 id="aliases" data-search-keywords="namespace aliases">Shorten names</h2><p>Text</p></div>'
				),
				url: new URL(record.url),
				helpers: { docsearch: () => [record] }
			});
			assert.equal(records[0].keywords, 'namespace aliases');
			assert.equal(records[0].url, record.url);
		}
		const before = await Promise.all(
			(await readdir(outDir))
				.sort()
				.map((file) => readFile(join(outDir, file), 'utf8'))
		);
		await run(process.execPath, [script, outDir], { cwd: tmpdir() });
		const after = await Promise.all(
			(await readdir(outDir))
				.sort()
				.map((file) => readFile(join(outDir, file), 'utf8'))
		);
		assert.deepEqual(after, before);
	} finally {
		await rm(outDir, { recursive: true });
	}
});

import assert from 'node:assert/strict';
import { mkdtempSync, mkdirSync, rmSync, writeFileSync } from 'node:fs';
import { tmpdir } from 'node:os';
import { join } from 'node:path';
import { test } from 'node:test';
import { createBlogDateResolver } from '../.vitepress/blog-date.ts';

test('preview dates use one UTC day for new posts and preserve published dates', (t) => {
	const latest = mkdtempSync(join(tmpdir(), 'task-blog-'));
	t.after(() => rmSync(latest, { recursive: true, force: true }));
	mkdirSync(join(latest, 'nested'));
	writeFileSync(join(latest, 'published.md'), 'Published content');
	writeFileSync(join(latest, 'nested/published.md'), 'Published content');
	const date = createBlogDateResolver(
		'next',
		latest,
		new Date('2026-09-27T23:30:00-02:00')
	);

	for (const stored of [undefined, '2000-01-01']) {
		assert.equal(
			date('new.md', stored).toISOString(),
			'2026-09-28T00:00:00.000Z'
		);
	}
	for (const file of ['published.md', 'nested/published.md', 'index.md']) {
		assert.equal(
			date(file, '2024-05-09').toISOString(),
			'2024-05-09T00:00:00.000Z'
		);
	}
	const posts = ['published.md', 'new.md'].map((file) => ({
		file,
		time: date(file, '2024-05-09').getTime()
	}));
	posts.sort((a, b) => b.time - a.time);
	assert.equal(posts[0].file, 'new.md');
});

test('latest always uses the stored date', () => {
	const date = createBlogDateResolver(
		'latest',
		'/unused',
		new Date('2026-09-27')
	);
	assert.equal(
		date('post.md', '2024-05-09').toISOString(),
		'2024-05-09T00:00:00.000Z'
	);
});

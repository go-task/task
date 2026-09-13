import assert from 'node:assert/strict';
import { test } from 'node:test';
import { createMarkdownRenderer } from 'vitepress';
import { renderSearchContent } from '../.vitepress/plugins/local-search.ts';

const md = await createMarkdownRenderer(process.cwd());

test('linked headings retain their text and permalink in the search index', () => {
	const html = renderSearchContent(
		'### [Homebrew](https://brew.sh) {#homebrew}\n\nInstall Task.\n',
		{},
		md
	);

	// VitePress splits sections at the first link leading to a hash. A link
	// around Homebrew used to make the extracted title empty.
	const heading = html.match(/<h3[^>]*>(.*?)<\/h3>/s)[1];
	assert.match(heading, /^Homebrew <a class="header-anchor" href="#homebrew"/);
	assert.ok(!heading.includes('href="https://brew.sh"'));
	assert.match(html, /<p>Install Task\.<\/p>/);
});

test('partial heading links preserve formatting without changing body links', () => {
	const src = [
		'### Arch ([**pacman**](https://example.com/pacman)) {#arch}',
		'',
		'[View package](https://example.com/package)',
		'',
		'```html',
		'<h3><a href="https://example.com">Example</a></h3>',
		'```'
	].join('\n');
	const original = md.render(src, {});
	const html = renderSearchContent(src, {}, md);

	assert.match(
		html,
		/Arch \(<strong>pacman<\/strong>\) <a class="header-anchor"/
	);
	assert.equal(
		html.slice(html.indexOf('</h3>')),
		original
			.slice(original.indexOf('</h3>'))
			.replace('<span class="lang">html</span>', '')
	);
});

test('pages opting out of search remain excluded', () => {
	assert.equal(
		renderSearchContent(
			'---\nsearch: false\n---\n\n# Private\n\nText.',
			{},
			md
		),
		''
	);
});

test('installation platform labels are indexed with their own heading', () => {
	const html = renderSearchContent(
		[
			'<InstallationMethod platforms="linux" platform-label="Ubuntu · Debian · Linux Mint">',
			'',
			'<template v-slot:heading>',
			'',
			'### [apt](https://example.com/apt) {#apt}',
			'',
			'</template>',
			'',
			'```shell',
			'apt install task',
			'```',
			'',
			'</InstallationMethod>'
		].join('\n'),
		{},
		md
	);

	assert.match(html, /<\/h3><p>Ubuntu · Debian · Linux Mint<\/p>/);
	assert.match(html, /<h3[^>]*>apt <a class="header-anchor" href="#apt"/);
});

test('code language labels do not become part of command names', () => {
	const html = renderSearchContent(
		'### Homebrew\n\n```shell\nbrew install go-task\n```',
		{},
		md
	);
	const text = html.replace(/<[^>]*>/g, '');
	assert.match(text, /\bbrew install go-task\b/);
	assert.ok(!text.includes('shellbrew'));
});

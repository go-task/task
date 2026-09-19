// Run `pnpm docsearch:export` from website/ to generate the dashboard configuration
// and per-index settings in docsearch-export/. Preserve the crawler's write key
// when pasting the generated configuration. Apply changed settings to existing
// indices separately; initialIndexSettings only initializes new indices.
//
// Both actions share an extractor, but write to separate indices. Keep the
// extractor self-contained: the hosted crawler can serialize it independently.

const channels = [
  { origin: 'https://taskfile.dev', indexName: 'taskfile' },
  { origin: 'https://next.taskfile.dev', indexName: 'taskfile-next' }
];

const recordExtractor = ({ $, helpers, url }) => {
  // ignoreNoIndex is needed for the next channel. Still honor page-level
  // opt-outs and the released site's noindex directives.
  if ($('meta[name="docsearch:exclude"]').attr('content') === 'true') return [];
  if (
    url.hostname === 'taskfile.dev' &&
    /\b(noindex|none)\b/i.test($('meta[name="robots"]').attr('content') || '')
  )
    return [];
  // The banner the llms plugin injects sits inside .vp-doc, ahead of the
  // h1. It is display:none for readers and must not become content.
  $('[data-nosnippet]').remove();

  // DocSearch expects content selectors to target paragraphs or list
  // items. Copy code blocks into crawler-only paragraphs so experienced
  // users can search for exact Taskfile keys and command syntax without
  // changing the page rendered to readers.
  $('.vp-doc pre code').each((_, element) => {
    const code = $(element).text().trim();
    if (!code) return;
    const paragraph = $('<p></p>').addClass('docsearch-code').text(code);
    $(element).closest('pre').replaceWith(paragraph);
  });

  // Select each text subtree once. A list item may contain paragraphs and
  // nested lists; selecting both parent and child repeats their content.
  $('.vp-doc p, .vp-doc li, .vp-doc td, .vp-doc th').each((_, element) => {
    if (!$(element).parents('.docsearch-content').length) {
      $(element).addClass('docsearch-content');
    }
  });

  // Frontmatter metadata is available after the refactor. Infer the same
  // values from the URL while the old monolithic guide is still live, so
  // this configuration can be installed before the website PR merges.
  // Remove this URL inference once the refactored documentation is live
  // and every indexed page exposes the DocSearch metadata.
  const pathname = url.pathname.replace(/\/+$/, '') || '/';
  const inferredSection = (() => {
    if (pathname === '/docs') return 'Overview';
    if (
      /^\/docs\/(installation|getting-started|integrations)$/.test(pathname)
    ) {
      return 'Getting Started';
    }
    if (/^\/docs\/reference\//.test(pathname)) return 'Reference';
    if (/^\/docs\/(contributing|releasing|styleguide)$/.test(pathname)) {
      return 'Contributing';
    }
    if (
      /^\/docs\/(experiments|deprecations|security)(\/|$)/.test(pathname) ||
      /^\/docs\/(changelog|faq|taskfile-versions|community)$/.test(pathname)
    ) {
      return 'Project';
    }
    return 'Guide';
  })();
  const section =
    $('meta[name="docsearch:section"]').attr('content') || inferredSection;
  const docType =
    $('meta[name="docsearch:doc_type"]').attr('content') ||
    ({
      Overview: 'overview',
      Reference: 'reference',
      Contributing: 'contributing',
      Project: 'project',
      Guide: 'guide',
      'Getting Started': 'guide'
    }[section] ??
      'guide');

  const keywordsByAnchor = new Map();
  $('.vp-doc [data-search-keywords]').each((_, element) => {
    const heading = $(element);
    keywordsByAnchor.set(
      heading.attr('id'),
      heading.attr('data-search-keywords')
    );
  });

  const records = helpers.docsearch({
    recordProps: {
      // Not a heading on the page: the section the page belongs to,
      // stated in its own frontmatter and emitted by transformHead. The
      // usual DocSearch recipe reads the active sidebar link out of the
      // DOM instead, which ties the index to the theme's markup and
      // breaks silently when that markup changes.
      lvl0: {
        // Algolia documents an empty selector as the way to provide a
        // raw, dynamically computed lvl0 through defaultValue.
        selectors: '',
        defaultValue: section
      },
      // Everything below is scoped to .vp-doc. VitePress renders the
      // sidebar's section labels as <h2 class="text"> inside
      // <aside class="VPSidebar">, five per page; an unscoped h2
      // selector would index those on all 46 pages.
      lvl1: '.vp-doc h1',
      lvl2: '.vp-doc h2',
      lvl3: '.vp-doc h3',
      lvl4: '.vp-doc h4',
      lvl5: '.vp-doc h5',
      lvl6: '.vp-doc h6',
      content: '.vp-doc .docsearch-content',
      section: { defaultValue: section },
      doc_type: { defaultValue: docType },
      lang: {
        defaultValue: $('html').attr('lang') || 'en-US'
      },
      // Deprecation notices remain findable, but don't outrank the page
      // that documents the supported replacement.
      pageRank: pathname.startsWith('/docs/deprecations/') ? '-10' : '0'
    },
    indexHeadings: true,
    aggregateContent: true,
    recordVersion: 'v3'
  });

  // Matching an alternate term should open its exact section. Adding
  // page-wide keywords would make every section on the page compete.
  return records.map((record) => ({
    ...record,
    keywords: keywordsByAnchor.get(record.anchor) || ''
  }));
};

const indexSettings = {
  hitsPerPage: 20,
  maxValuesPerFacet: 100,
  attributesForFaceting: ['type', 'lang', 'section', 'doc_type'],
  attributesToRetrieve: [
    'hierarchy',
    'content',
    'anchor',
    'url',
    'url_without_anchor',
    'type',
    'lang',
    'section',
    'doc_type'
  ],
  attributesToHighlight: ['hierarchy', 'content'],
  attributesToSnippet: ['content:10'],
  camelCaseAttributes: ['hierarchy', 'content'],
  searchableAttributes: [
    'unordered(hierarchy.lvl0)',
    'unordered(hierarchy.lvl1)',
    'unordered(hierarchy.lvl2)',
    'unordered(hierarchy.lvl3)',
    'unordered(hierarchy.lvl4)',
    'unordered(hierarchy.lvl5)',
    'unordered(hierarchy.lvl6)',
    'unordered(keywords)',
    'content'
  ],
  distinct: true,
  attributeForDistinct: 'url',
  customRanking: [
    'desc(weight.pageRank)',
    'desc(weight.level)',
    'asc(weight.position)'
  ],
  ranking: [
    'words',
    'filters',
    'typo',
    'attribute',
    'proximity',
    'exact',
    'custom'
  ],
  highlightPreTag: '<span class="algolia-docsearch-suggestion--highlight">',
  highlightPostTag: '</span>',
  minWordSizefor1Typo: 3,
  minWordSizefor2Typos: 7,
  allowTyposOnNumericTokens: false,
  minProximity: 1,
  ignorePlurals: true,
  advancedSyntax: true,
  attributeCriteriaComputedByMinProximity: true,
  removeWordsIfNoResults: 'allOptional',
  separatorsToIndex: '_',
  paginationLimitedTo: 1000,
  exactOnSingleWordQuery: 'attribute',
  queryType: 'prefixLast',
  snippetEllipsisText: '',
  alternativesAsExact: ['ignorePlurals', 'singleWordSynonym']
};

new Crawler({
  appId: '7IZIJ13AI7',
  apiKey: '<ALGOLIA_CRAWLER_WRITE_KEY>',
  indexPrefix: '',
  rateLimit: 8,
  maxDepth: 10,
  schedule: 'at 9:50 AM on Thursday',
  // next retains latest canonicals and noindex for public search engines.
  ignoreCanonicalTo: true,
  ignoreNoIndex: true,
  saveBackup: true,
  safetyChecks: {
    beforeIndexPublishing: { maxLostRecordsPercentage: 10 },
    maxFailedUrls: 5
  },
  startUrls: channels.map(({ origin }) => `${origin}/`),
  sitemaps: channels.map(({ origin }) => `${origin}/sitemap.xml`),
  discoveryPatterns: channels.map(({ origin }) => `${origin}/docs/**`),
  exclusionPatterns: channels.flatMap(({ origin }) => [
    `${origin}/docs/changelog**`,
    `${origin}/docs/**/template`,
    `${origin}/docs/**/*.md`
  ]),
  actions: channels.map(({ origin, indexName }) => ({
    indexName,
    pathsToMatch: [`${origin}/docs/**`],
    recordExtractor
  })),
  initialIndexSettings: Object.fromEntries(
    channels.map(({ indexName }) => [indexName, indexSettings])
  )
});

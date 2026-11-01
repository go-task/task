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
  // The crawler editor types $ as accepting strings only. Use eq() to
  // access selected nodes without passing a Cheerio Element back to $.
  const codeBlocks = $('.vp-doc pre code');
  codeBlocks.each((index) => {
    const block = codeBlocks.eq(index);
    const code = block.text().trim();
    if (!code) return;
    const paragraph = $('<p></p>').addClass('docsearch-code').text(code);
    block.closest('pre').replaceWith(paragraph);
  });

  // Select each text subtree once. A list item may contain paragraphs and
  // nested lists; selecting both parent and child repeats their content.
  const contentElements = $('.vp-doc p, .vp-doc li, .vp-doc td, .vp-doc th');
  contentElements.each((index) => {
    const content = contentElements.eq(index);
    if (!content.parents('.docsearch-content').length) {
      content.addClass('docsearch-content');
    }
  });

  // Frontmatter metadata is available after the refactor. Infer the same
  // values from the URL while the old monolithic guide is still live, so
  // this configuration can be installed before the website PR merges.
  // Remove this URL inference once the refactored documentation is live
  // and every indexed page exposes the DocSearch metadata.
  const pathname = url.pathname.replace(/\/+$/, '') || '/';
  const isNavigationPage = pathname === '/docs' || pathname === '/docs/guide';
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
    {
      Overview: 'overview',
      Reference: 'reference',
      Contributing: 'contributing',
      Project: 'project',
      Guide: 'guide',
      'Getting Started': 'guide'
    }[section] ||
    'guide';

  const keywordsByAnchor = new Map();
  const keywordHeadings = $('.vp-doc [data-search-keywords]');
  keywordHeadings.each((index) => {
    const heading = keywordHeadings.eq(index);
    keywordsByAnchor.set(
      heading.attr('id'),
      heading.attr('data-search-keywords')
    );
  });

  // Common task-oriented phrases should lead to their practical guide.
  // These aliases also cover deployed pages that don't yet emit them in HTML.
  const guideKeywords = {
    '/docs/guide/includes': {
      'including-other-taskfiles':
        'include another taskfile including taskfiles'
    },
    '/docs/guide/running-tasks': {
      'dry-run-mode': 'dry run preview commands'
    }
  };
  const pageKeywords = guideKeywords[pathname] || {};

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
      pageRank: pathname.startsWith('/docs/deprecations/')
        ? '-10'
        : /^\/docs\/experiments(\/|$)/.test(pathname)
          ? '-5'
          : '0'
    },
    indexHeadings: true,
    aggregateContent: true,
    recordVersion: 'v3'
  });

  // Matching an alternate term should open its exact section. Adding
  // page-wide keywords would make every section on the page compete.
  return records
    .filter((record) => {
      // Navigation hubs contain summaries of other pages. Keep their page
      // title findable without indexing each card or summary as a result.
      if (isNavigationPage) return record.type === 'lvl1';
      // These headings organize template variables; their child entries
      // (ALIAS, CHECKSUM, EXIT_CODE, etc.) remain indexed with breadcrumbs.
      return (
        pathname !== '/docs/reference/templating' ||
        !['task', 'status', 'loop', 'defer'].includes(record.anchor)
      );
    })
    .map((record) => {
      // Rank a section by its own title, independently of its heading depth.
      // VitePress adds a zero-width space for the heading permalink.
      const title =
        [
          record.hierarchy.lvl6,
          record.hierarchy.lvl5,
          record.hierarchy.lvl4,
          record.hierarchy.lvl3,
          record.hierarchy.lvl2,
          record.hierarchy.lvl1
        ].find(Boolean) || '';
      return {
        ...record,
        title: title.replace(/\u200b/g, '').trim(),
        keywords: [
          keywordsByAnchor.get(record.anchor) || '',
          pageKeywords[record.anchor] || ''
        ]
          .filter(Boolean)
          .join(' ')
      };
    });
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
  camelCaseAttributes: ['title', 'hierarchy', 'content'],
  searchableAttributes: [
    // Ordered titles favor a query at the beginning of the section title.
    'title',
    'unordered(keywords)',
    // Breadcrumb headings provide context at the same priority, without
    // favoring a page title over a more specific section heading.
    'hierarchy.lvl1,hierarchy.lvl2,hierarchy.lvl3,hierarchy.lvl4,hierarchy.lvl5,hierarchy.lvl6',
    'unordered(hierarchy.lvl0)',
    'content'
  ],
  distinct: true,
  attributeForDistinct: 'url',
  customRanking: ['desc(weight.level)', 'asc(weight.position)'],
  ranking: [
    'words',
    'filters',
    'typo',
    // Exact words beat prefixes such as "for" matching "force". Apply
    // deprecation penalties before considering the matched attribute.
    'exact',
    'desc(weight.pageRank)',
    'attribute',
    'proximity',
    'custom'
  ],
  highlightPreTag: '<span class="algolia-docsearch-suggestion--highlight">',
  highlightPostTag: '</span>',
  minWordSizefor1Typo: 3,
  minWordSizefor2Typos: 7,
  allowTyposOnNumericTokens: false,
  minProximity: 1,
  // The documentation is English. Applying every language's plural rules
  // can turn an unfinished query such as "completi" into another word.
  queryLanguages: ['en'],
  ignorePlurals: ['en'],
  // Exact words in a long paragraph or an ancestor's title shouldn't beat
  // an exact match in the destination section's own title or keywords.
  disableExactOnAttributes: [
    'content',
    'hierarchy.lvl0',
    'hierarchy.lvl1',
    'hierarchy.lvl2',
    'hierarchy.lvl3',
    'hierarchy.lvl4',
    'hierarchy.lvl5',
    'hierarchy.lvl6'
  ],
  advancedSyntax: true,
  attributeCriteriaComputedByMinProximity: true,
  removeWordsIfNoResults: 'allOptional',
  separatorsToIndex: '_',
  paginationLimitedTo: 1000,
  exactOnSingleWordQuery: 'word',
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

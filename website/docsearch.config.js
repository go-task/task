// Algolia Crawler configuration for taskfile.dev.
//
// This file is the source of truth. The crawler itself runs on Algolia's side,
// configured through the dashboard at https://crawler.algolia.com, and until now
// nothing described it here: `git log --all -- '*algolia*' '*docsearch*'`
// returned nothing, so the only copy lived in a web form nobody could review.
// When you change the configuration in the dashboard, change it here too.
//
// The API key below is the crawler's *write* key and is deliberately not in
// this repository. Set it in the dashboard, or in the crawler's environment.
//
// Every selector here was checked against the generated HTML, not assumed.

new Crawler({
  appId: '7IZIJ13AI7',
  apiKey: process.env.ALGOLIA_CRAWLER_WRITE_KEY,
  indexPrefix: '',
  rateLimit: 8,
  maxDepth: 10,
  schedule: 'every 1 day',

  // Only the released site. next.taskfile.dev serves the same URLs from the
  // upcoming release, and both sites share the single `taskfile` index, so
  // crawling it as well would give every page a duplicate record.
  startUrls: ['https://taskfile.dev/docs/'],
  sitemaps: ['https://taskfile.dev/sitemap.xml'],
  // Only /docs is indexed, so there is no reason to fetch the blog, the
  // homepage or /adopters on every crawl.
  discoveryPatterns: ['https://taskfile.dev/docs/**'],

  exclusionPatterns: [
    // Long, low-value for search, and it would outrank real pages on any
    // version number or feature name it mentions.
    'https://taskfile.dev/docs/changelog**',
    // Scaffolding for writing new pages; already out of the sitemap.
    'https://taskfile.dev/docs/*/template'
  ],

  actions: [
    {
      indexName: 'taskfile',
      pathsToMatch: ['https://taskfile.dev/docs/**'],
      recordExtractor: ({ $, helpers }) => {
        // The banner the llms plugin injects sits inside .vp-doc, ahead of the
        // h1. It is display:none for readers and must not become content.
        $('[data-nosnippet]').remove();

        return helpers.docsearch({
          recordProps: {
            // Not a heading on the page: the section the page belongs to,
            // stated in its own frontmatter and emitted by transformHead. The
            // usual DocSearch recipe reads the active sidebar link out of the
            // DOM instead, which ties the index to the theme's markup and
            // breaks silently when that markup changes.
            lvl0: {
              selectors: 'meta[name="docsearch:section"]',
              defaultValue: 'Documentation'
            },
            // Everything below is scoped to .vp-doc. VitePress renders the
            // sidebar's section labels as <h2 class="text"> inside
            // <aside class="VPSidebar">, five per page; an unscoped h2
            // selector would index those on all 46 pages.
            lvl1: '.vp-doc h1',
            lvl2: '.vp-doc h2',
            lvl3: '.vp-doc h3',
            lvl4: '.vp-doc h4',
            content: '.vp-doc p, .vp-doc li, .vp-doc td',
            // Faceted so a search can be narrowed to reference pages, or
            // weighted differently later.
            docType: {
              selectors: 'meta[name="docsearch:type"]',
              defaultValue: 'guide'
            }
          },
          aggregateContent: true,
          recordVersion: 'v3'
        });
      }
    },
    {
      // Deprecation notices describe what to stop doing. They should be
      // findable by name, but never ahead of the page documenting the
      // replacement.
      indexName: 'taskfile',
      pathsToMatch: ['https://taskfile.dev/docs/deprecations/**'],
      pageRank: -10,
      recordExtractor: ({ $, helpers }) => {
        $('[data-nosnippet]').remove();
        return helpers.docsearch({
          recordProps: {
            lvl0: {
              selectors: 'meta[name="docsearch:section"]',
              defaultValue: 'Deprecations'
            },
            lvl1: '.vp-doc h1',
            lvl2: '.vp-doc h2',
            lvl3: '.vp-doc h3',
            content: '.vp-doc p, .vp-doc li, .vp-doc td'
          },
          aggregateContent: true,
          recordVersion: 'v3'
        });
      }
    }
  ],

  initialIndexSettings: {
    taskfile: {
      attributesForFaceting: ['type', 'lang', 'docType'],
      attributesToRetrieve: [
        'hierarchy',
        'content',
        'anchor',
        'url',
        'docType'
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
      removeWordsIfNoResults: 'allOptional'
    }
  }
});

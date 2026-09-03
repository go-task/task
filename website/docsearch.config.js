// Algolia Crawler configuration for taskfile.dev.
//
// This file is the source of truth. The crawler itself runs on Algolia's side,
// configured through the dashboard at https://crawler.algolia.com, and until now
// nothing described it here: `git log --all -- '*algolia*' '*docsearch*'`
// returned nothing, so the only copy lived in a web form nobody could review.
// When you change the configuration in the dashboard, change it here too.
//
// The API key below is the crawler's *write* key and is deliberately not in
// this repository. Keep the existing key when pasting this file into the
// dashboard; never replace this placeholder in Git.
//
// Every selector here was checked against the generated HTML, not assumed.

new Crawler({
  appId: '7IZIJ13AI7',
  apiKey: '<ALGOLIA_CRAWLER_WRITE_KEY>',
  indexPrefix: '',
  rateLimit: 8,
  maxDepth: 10,
  schedule: 'at 9:50 AM on Thursday',
  ignoreCanonicalTo: true,
  saveBackup: true,

  safetyChecks: {
    beforeIndexPublishing: {
      maxLostRecordsPercentage: 10
    },
    // The dashboard's current config.d.ts exposes this at the safetyChecks
    // level, rather than inside beforeIndexPublishing.
    maxFailedUrls: 5
  },

  // Only the released site. next.taskfile.dev serves the same URLs from the
  // upcoming release, and both sites share the single `taskfile` index, so
  // crawling it as well would give every page a duplicate record.
  // The root exists both before and after the documentation refactor. The old
  // production site has no /docs/ landing page yet.
  startUrls: ['https://taskfile.dev/'],
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
      recordExtractor: ({ $, helpers, url }) => {
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
          $(element).closest('pre').after(paragraph);
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
            /^\/docs\/(installation|getting-started|integrations)$/.test(
              pathname
            )
          ) {
            return 'Getting Started';
          }
          if (/^\/docs\/reference\//.test(pathname)) return 'Reference';
          if (/^\/docs\/(contributing|releasing|styleguide)$/.test(pathname)) {
            return 'Contributing';
          }
          if (
            /^\/docs\/(experiments|deprecations|security)(\/|$)/.test(
              pathname
            ) ||
            /^\/docs\/(changelog|faq|taskfile-versions|community)$/.test(
              pathname
            )
          ) {
            return 'Project';
          }
          return 'Guide';
        })();
        const section =
          $('meta[name="docsearch:section"]').attr('content') ||
          inferredSection;
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

        return helpers.docsearch({
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
            content: '.vp-doc p, .vp-doc li, .vp-doc td, .vp-doc th',
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
      }
    }
  ],

  // These settings initialize a new index. Algolia doesn't apply them to an
  // existing index, so import the same values in the taskfile index settings
  // when this configuration changes them.
  initialIndexSettings: {
    taskfile: {
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
    }
  }
});

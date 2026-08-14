import { defineConfig, HeadConfig } from 'vitepress';
import githubLinksPlugin from './plugins/github-links';
import { readFileSync } from 'fs';
import { resolve } from 'path';
import { tabsMarkdownPlugin } from 'vitepress-plugin-tabs';
import {
  groupIconMdPlugin,
  groupIconVitePlugin,
  localIconLoader
} from 'vitepress-plugin-group-icons';
import { team } from './team.ts';
import { adopters } from './adopters.ts';
import { taskDescription, taskName, ogUrl, ogImage } from './meta.ts';
import { fileURLToPath, URL } from 'node:url';
import llmstxt from 'vitepress-plugin-llms';
import { sidebar as nextSidebar } from './sidebar/next.ts';
import { sidebar as latestSidebar } from './sidebar/latest.ts';

const version = readFileSync(
  resolve(__dirname, '../../internal/version/version.txt'),
  'utf8'
).trim();

// Which set of docs to build. `next` (the default) serves `src/docs`, the docs
// being written for the upcoming release. `latest` serves `src/latest/docs`
// under the very same `/docs/` URLs, so that taskfile.dev never documents
// features that are not in the released binary.
const isLatest = process.env.DOCS_CHANNEL === 'latest';

// Where the docs of the active channel live, relative to `srcDir`. Plugins that
// match on source paths need this; `rewrites` only affects the output URLs.
const docsRoot = isLatest ? 'latest/docs' : 'docs';

const urlVersion =
  process.env.NODE_ENV === 'development'
    ? {
        current: 'https://taskfile.dev/',
        next: 'http://localhost:3002/'
      }
    : {
        current: 'https://taskfile.dev/',
        next: 'https://next.taskfile.dev/'
      };

// https://vitepress.dev/reference/site-config
export default defineConfig({
  title: taskName,
  description: taskDescription,
  lang: 'en-US',
  head: [
    // Favicon ICO for legacy browsers (auto-discovery)
    ['link', { rel: 'icon', href: '/favicon.ico', sizes: '48x48' }],
    // Favicon SVG for modern browsers (scalable)
    ['link', { rel: 'icon', href: '/img/logo.svg', type: 'image/svg+xml' }],
    // Apple Touch Icon for iOS devices
    ['link', { rel: 'apple-touch-icon', href: '/img/logo.png' }],
    [
      'meta',
      { name: 'author', content: `${team.map((c) => c.name).join(', ')}` }
    ],
    // Open Graph
    ['meta', { property: 'og:type', content: 'website' }],
    ['meta', { property: 'og:site_name', content: 'Task' }],
    ['meta', { property: 'og:image', content: ogImage }],
    // Twitter Card
    ['meta', { name: 'twitter:card', content: 'summary_large_image' }],
    ['meta', { name: 'twitter:site', content: '@taskfiledev' }],
    ['meta', { name: 'twitter:image', content: ogImage }],
    [
      'meta',
      {
        name: 'keywords',
        content:
          'task runner, build tool, taskfile, yaml build tool, go task runner, make alternative, cross-platform build tool, makefile alternative, automation tool, ci cd pipeline, developer productivity, build automation, command line tool, go binary, yaml configuration'
      }
    ],
    [
      "script",
      {
        defer: "",
        src: "https://u.taskfile.dev/script.js",
        "data-website-id": "084030b0-0e3f-4891-8d2a-0c12c40f5933"
      }
    ],
    [
      "script",
      { type: "application/ld+json" },
      JSON.stringify({
        "@context": "https://schema.org",
        "@type": "WebSite",
        "name": "Task",
        "url": "https://taskfile.dev/"
      })
    ]
  ],
  transformHead({ pageData }) {
    const head: HeadConfig[] = []

    // Canonical URL dynamique
    const canonicalUrl = `https://taskfile.dev/${pageData.relativePath
      .replace(/\.md$/, '')
      .replace(/index$/, '')}`
    head.push(['link', { rel: 'canonical', href: canonicalUrl }])

    // Dynamic Open Graph and Twitter meta tags
    const isHome = pageData.relativePath === 'index.md';
    var pageTitle = pageData.frontmatter.title || pageData.title || taskName;
    if (!isHome) {
      pageTitle = `${pageTitle} | ${taskName}`;
    }
    const pageDescription = pageData.frontmatter.description || pageData.description || taskDescription
    head.push(['meta', { property: 'og:title', content: pageTitle }])
    head.push(['meta', { property: 'og:description', content: pageDescription }])
    head.push(['meta', { property: 'og:url', content: canonicalUrl }])
    head.push(['meta', { name: 'twitter:title', content: pageTitle }])
    head.push(['meta', { name: 'twitter:description', content: pageDescription }])

    // Noindex pour 404
    if (pageData.relativePath === '404.md') {
      head.push(['meta', { name: 'robots', content: 'noindex, nofollow' }])
    }

    // Structured data for the adopters carousel on the homepage: an ItemList
    // of Organization entities so search engines can surface Task's adopters
    // directly in rich results.
    if (isHome) {
      head.push([
        'script',
        { type: 'application/ld+json' },
        JSON.stringify({
          '@context': 'https://schema.org',
          '@type': 'ItemList',
          name: 'Organizations and projects using Task',
          itemListOrder: 'https://schema.org/ItemListUnordered',
          numberOfItems: adopters.length,
          itemListElement: adopters.map((a, i) => ({
            '@type': 'ListItem',
            position: i + 1,
            item: {
              '@type': 'Organization',
              name: a.name,
              url: a.url,
              logo: a.img,
              sameAs: [a.url]
            }
          }))
        })
      ])
    }

    // On the /adopters page, emit CollectionPage + ItemList (richer than the
    // homepage snippet because it targets this specific URL) and FAQPage for
    // the question block at the bottom of the page. Kept in sync by hand with
    // components/Adopters.vue.
    if (pageData.relativePath === 'adopters.md') {
      head.push([
        'script',
        { type: 'application/ld+json' },
        JSON.stringify({
          '@context': 'https://schema.org',
          '@type': 'CollectionPage',
          name: 'Who uses Task',
          url: 'https://taskfile.dev/adopters',
          description:
            'Organizations and open source projects that use Task as their build and release runner.',
          mainEntity: {
            '@type': 'ItemList',
            numberOfItems: adopters.length,
            itemListElement: adopters.map((a, i) => ({
              '@type': 'ListItem',
              position: i + 1,
              item: {
                '@type': 'Organization',
                name: a.name,
                url: a.url,
                logo: a.img,
                description: a.description,
                sameAs: [a.url]
              }
            }))
          }
        })
      ])

      head.push([
        'script',
        { type: 'application/ld+json' },
        JSON.stringify({
          '@context': 'https://schema.org',
          '@type': 'FAQPage',
          mainEntity: [
            {
              '@type': 'Question',
              name: 'Is Task production-ready?',
              acceptedAnswer: {
                '@type': 'Answer',
                text: 'Yes. Task ships as a single static binary, has been in wide production use since 2018, and powers the release workflows of projects with millions of downloads including Arduino CLI, GoReleaser, FerretDB, and Gogs.'
              }
            },
            {
              '@type': 'Question',
              name: 'Who uses Task in enterprise?',
              acceptedAnswer: {
                '@type': 'Answer',
                text: 'Docker, Vercel, HashiCorp, Microsoft (Azure Sentinel), Google Cloud, AWS, and Anthropic are among the organizations that ship code with a Taskfile.yml. Task is also embedded end-to-end in Arduino’s developer tooling stack across more than 70 repositories.'
              }
            },
            {
              '@type': 'Question',
              name: 'How is Task different from Make?',
              acceptedAnswer: {
                '@type': 'Answer',
                text: 'Task uses plain YAML instead of Make’s tab-sensitive syntax, runs identically on Linux, macOS, and Windows, and provides built-in caching based on file fingerprints. It also comes with an ecosystem of editor and CI integrations that Make lacks by default.'
              }
            },
            {
              '@type': 'Question',
              name: 'Where can I find real-world Taskfile examples?',
              acceptedAnswer: {
                '@type': 'Answer',
                text: 'Every adopter listed above links directly to a public repository containing a production Taskfile.yml. Browsing those is the fastest way to see Task used in real codebases at different scales.'
              }
            }
          ]
        })
      ])
    }

    return head
  },
  srcDir: 'src',
  cleanUrls: true,
  srcExclude: isLatest ? ['docs/**'] : ['latest/**'],
  rewrites: isLatest ? { 'latest/docs/:path*': 'docs/:path*' } : {},
  markdown: {
    config: (md) => {
      md.use(githubLinksPlugin, {
        baseUrl: 'https://github.com',
        repo: 'go-task/task'
      });
      md.use(tabsMarkdownPlugin);
      md.use(groupIconMdPlugin);
    }
  },
  vite: {
    plugins: [
      llmstxt({
        ignoreFiles: [
          'index.md',
          'team.md',
          'donate.md',
          `${docsRoot}/styleguide.md`,
          `${docsRoot}/contributing.md`,
          `${docsRoot}/releasing.md`,
          `${docsRoot}/changelog.md`,
          'blog/*'
        ]
      }),
      groupIconVitePlugin({
        customIcon: {
          '.taskrc.yml': localIconLoader(
            import.meta.url,
            './theme/icons/task.svg'
          ),
          'Taskfile.yml': localIconLoader(
            import.meta.url,
            './theme/icons/task.svg'
          )
        }
      })
    ],
    resolve: {
      alias: [
        {
          find: /^.*\/VPTeamMembersItem\.vue$/,
          replacement: fileURLToPath(
            new URL('./components/VPTeamMembersItem.vue', import.meta.url)
          )
        }
      ]
    }
  },

  themeConfig: {
    logo: '/img/logo.svg',
    carbonAds: {
      code: 'CESI65QJ',
      placement: 'taskfiledev'
    },
    search: {
      provider: 'algolia',
      options: {
        appId: '7IZIJ13AI7',
        apiKey: '34b64ae4fc8d9da43d9a13d9710aaddc',
        indexName: 'taskfile'
      }
    },
    nav: [
      { text: 'Home', link: '/' },
      {
        text: 'Docs',
        link: '/docs/guide',
        activeMatch: '^/docs'
      },
      { text: 'Blog', link: '/blog', activeMatch: '^/blog' },
      { text: 'Donate', link: '/donate' },
      { text: 'Team', link: '/team' },
      {
        text: process.env.NODE_ENV === 'development' ? 'Next' : `v${version}`,
        items: [
          {
            items: [
              {
                text: `v${version}`,
                link: urlVersion.current
              },
              {
                text: 'Next',
                link: urlVersion.next
              }
            ]
          }
        ]
      }
    ],

    sidebar: {
      '/blog/': [
        {
          text: '2026',
          collapsed: false,
          items: [
            {
              text: 'GitHub SOSF',
              link: '/blog/github-secure-open-source-program'
            },
            {
              text: 'Using `go tool task`',
              link: '/blog/go-tool-task'
            },
            {
              text: 'Conditionals Statements',
              link: '/blog/if-and-variable-prompt'
            }
          ]
        },
        {
          text: '2025',
          collapsed: false,
          items: [
            {
              text: 'Built-in Core Utilities',
              link: '/blog/windows-core-utils'
            }
          ]
        },
        {
          text: '2024',
          collapsed: false,
          items: [
            {
              text: 'Any Variables',
              link: '/blog/any-variables'
            }
          ]
        },
        {
          text: '2023',
          collapsed: false,
          items: [
            {
              text: 'Introducing Experiments',
              link: '/blog/task-in-2023'
            }
          ]
        }
      ],
      '/': isLatest ? latestSidebar : nextSidebar,
      // Hacky to disable sidebar for these pages
      '/donate': [],
      '/team': [],
      '/adopters': []
    },

    socialLinks: [
      { icon: 'github', link: 'https://github.com/go-task/task' },
      { icon: 'discord', link: 'https://discord.gg/6TY36E39UK' },
      { icon: 'x', link: 'https://twitter.com/taskfiledev' },
      { icon: 'bluesky', link: 'https://bsky.app/profile/taskfile.dev' },
      { icon: 'mastodon', link: 'https://fosstodon.org/@task' }
    ],

    editLink: {
      text: 'Edit this page on GitHub',
      // Docs are always edited in `src/docs`, even when the latest channel
      // serves them from `src/latest/docs`.
      pattern: ({ filePath }: { filePath: string }) =>
        `https://github.com/go-task/task/edit/main/website/src/${filePath.replace(/^latest\//, '')}`
    },

    footer: {
      message:
        'Built with <a target="_blank" href="https://www.netlify.com">Netlify</a>'
    }
  },
  sitemap: {
    hostname: 'https://taskfile.dev',
    transformItems: (items) => {
      return items.map((item) => ({
        ...item,
        lastmod: new Date().toISOString()
      }));
    }
  }
});

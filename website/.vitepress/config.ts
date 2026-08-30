import { defineConfig, HeadConfig } from 'vitepress';
import githubLinksPlugin from './plugins/github-links';
import { readdirSync, readFileSync, writeFileSync } from 'fs';
import { resolve } from 'path';
import matter from 'gray-matter';
import { tabsMarkdownPlugin } from 'vitepress-plugin-tabs';
import {
  groupIconMdPlugin,
  groupIconVitePlugin,
  localIconLoader
} from 'vitepress-plugin-group-icons';
import { team } from './team.ts';
import { adopters } from './adopters.ts';
import { taskDescription, taskName, ogImage } from './meta.ts';
import { fileURLToPath, URL } from 'node:url';
import llmstxt from 'vitepress-plugin-llms';
import { sidebar as nextSidebar } from './sidebar/next.ts';
import { sidebar as latestSidebar } from './sidebar/latest.ts';

const version = readFileSync(
  resolve(__dirname, '../../internal/version/version.txt'),
  'utf8'
).trim();

// Which channel to build. `src/next` is written for the upcoming release and
// serves next.taskfile.dev; `src/latest` is its copy at the released version
// and serves taskfile.dev. Both mount at the same URLs, so taskfile.dev never
// documents or announces a feature that is not in the released binary.
// cmd/release owns the other half of this: it promotes one over the other.
const isLatest = process.env.DOCS_CHANNEL === 'latest';
const channel = isLatest ? 'latest' : 'next';
const other = isLatest ? 'next' : 'latest';
const isPublicDeploy =
  process.env.DOCS_SITE === 'production' && process.env.DOCS_LOCAL !== '1';
const isProduction = isLatest && isPublicDeploy;

const docsSidebar = isLatest ? latestSidebar : nextSidebar;

// Builds the "/blog/" sidebar from each blog post's frontmatter.
function buildBlogSidebar() {
  const blogDir = resolve(__dirname, `../src/${channel}/blog`);
  const posts = readdirSync(blogDir)
    .filter((file) => file.endsWith('.md') && file !== 'index.md')
    .map((file) => {
      const { data: frontmatter } = matter(
        readFileSync(resolve(blogDir, file), 'utf8')
      );
      return {
        slug: file.replace(/\.md$/, ''),
        title: frontmatter.sidebarTitle ?? frontmatter.title,
        date: new Date(frontmatter.date)
      };
    })
    .sort((a, b) => b.date.getTime() - a.date.getTime());

  const byYear = new Map<number, { text: string; link: string }[]>();
  for (const post of posts) {
    const year = post.date.getFullYear();
    if (!byYear.has(year)) byYear.set(year, []);
    byYear.get(year)!.push({ text: post.title, link: `/blog/${post.slug}` });
  }

  return [...byYear.entries()]
    .sort((a, b) => b[0] - a[0])
    .map(([year, items]) => ({
      text: String(year),
      collapsed: false,
      items
    }));
}

// Ports are the ones the dev tasks bind to; keep them in sync with
// website/Taskfile.yml. DOCS_LOCAL is set by those tasks alone, so a build can
// never end up shipping localhost URLs.
const localPorts = { latest: 3002, next: 3001 };
const urlVersion =
  process.env.DOCS_LOCAL === '1'
    ? {
        current: `http://localhost:${localPorts.latest}/`,
        next: `http://localhost:${localPorts.next}/`
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
    ['meta', { property: 'og:site_name', content: 'Task' }],
    ['meta', { property: 'og:image', content: ogImage }],
    // Twitter Card
    ['meta', { name: 'twitter:card', content: 'summary_large_image' }],
    ['meta', { name: 'twitter:site', content: '@taskfiledev' }],
    ['meta', { name: 'twitter:image', content: ogImage }],
    ...(isPublicDeploy
      ? ([
          [
            'script',
            {
              defer: '',
              src: 'https://u.taskfile.dev/script.js',
              'data-website-id': '084030b0-0e3f-4891-8d2a-0c12c40f5933'
            }
          ]
        ] satisfies HeadConfig[])
      : []),
    [
      'script',
      { type: 'application/ld+json' },
      JSON.stringify({
        '@context': 'https://schema.org',
        '@type': 'WebSite',
        name: 'Task',
        url: 'https://taskfile.dev/'
      })
    ]
  ],
  transformHead({ pageData }) {
    const head: HeadConfig[] = [];

    const canonicalPath = pageData.relativePath
      .replace(/\.md$/, '')
      .replace(/index$/, '');
    const canonicalUrl = new URL(
      typeof pageData.frontmatter.canonical === 'string'
        ? pageData.frontmatter.canonical
        : canonicalPath,
      'https://taskfile.dev/'
    ).href;
    head.push(['link', { rel: 'canonical', href: canonicalUrl }]);

    // Dynamic Open Graph and Twitter meta tags
    const isHome = new URL(canonicalUrl).pathname === '/';
    let pageTitle = pageData.frontmatter.title || pageData.title || taskName;
    if (!isHome) {
      pageTitle = `${pageTitle} | ${taskName}`;
    }
    const pageDescription =
      pageData.frontmatter.description ||
      pageData.description ||
      taskDescription;
    head.push([
      'meta',
      {
        property: 'og:type',
        content:
          canonicalUrl.includes('/blog/') && !canonicalUrl.endsWith('/blog/')
            ? 'article'
            : 'website'
      }
    ]);
    head.push(['meta', { property: 'og:title', content: pageTitle }]);
    head.push([
      'meta',
      { property: 'og:description', content: pageDescription }
    ]);
    head.push(['meta', { property: 'og:url', content: canonicalUrl }]);
    head.push(['meta', { name: 'twitter:title', content: pageTitle }]);
    head.push([
      'meta',
      { name: 'twitter:description', content: pageDescription }
    ]);

    // Only the released public site is indexable. The public next site and
    // previews keep production canonicals but must never be indexed.
    if (
      !isProduction ||
      pageData.relativePath === '404.md' ||
      pageData.frontmatter.noindex === true
    ) {
      head.push(['meta', { name: 'robots', content: 'noindex, nofollow' }]);
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
      ]);
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
      ]);

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
      ]);
    }

    return head;
  },
  srcDir: 'src',
  cleanUrls: true,
  srcExclude: [`${other}/**`, `${channel}/docs/**/template.md`],
  rewrites: { [`${channel}/:path*`]: ':path*' },
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
          // Matched against source paths, which `rewrites` does not touch.
          `${channel}/docs/styleguide.md`,
          `${channel}/docs/contributing.md`,
          `${channel}/docs/releasing.md`,
          `${channel}/docs/changelog.md`,
          `${channel}/blog/*`
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
        },
        {
          find: /^.*\/VPSponsorsGrid\.vue$/,
          replacement: fileURLToPath(
            new URL('./components/VPSponsorsGrid.vue', import.meta.url)
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
    search: isProduction
      ? {
          provider: 'algolia',
          options: {
            appId: '7IZIJ13AI7',
            apiKey: '34b64ae4fc8d9da43d9a13d9710aaddc',
            indexName: 'taskfile'
          }
        }
      : {
          provider: 'local',
          options: {
            detailedView: true,
            miniSearch: {
              searchOptions: {
                fuzzy: 0.2,
                prefix: true,
                boost: { title: 4, titles: 2, text: 1 }
              }
            }
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
        text: isLatest ? `v${version}` : 'Next',
        items: [
          {
            items: [
              // Absolute links, so VitePress would treat them as external and
              // open them in a new tab. Switching channels is navigation, not a
              // detour off the site.
              {
                text: `v${version}`,
                link: urlVersion.current,
                target: '_self',
                noIcon: true
              },
              {
                text: 'Next',
                link: urlVersion.next,
                target: '_self',
                noIcon: true
              }
            ]
          }
        ]
      }
    ],

    sidebar: {
      '/blog/': buildBlogSidebar(),
      '/': docsSidebar,
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
      // Docs are always edited in `src/next`, even when the latest channel
      // serves them from `src/latest/docs`.
      // Serialized with toString() and evaluated in the browser, so it must not
      // reference anything from this module. Both channels are edited in
      // src/next, so strip whichever prefix the page was built from.
      pattern: ({ filePath }) =>
        `https://github.com/go-task/task/edit/main/website/src/next/${filePath.replace(/^(next|latest)\//, '')}`
    },

    footer: {
      message:
        'Built with <a target="_blank" href="https://www.netlify.com">Netlify</a>'
    }
  },
  sitemap: {
    hostname: 'https://taskfile.dev'
  },
  buildEnd({ outDir }) {
    const robots = isProduction
      ? [
          'User-agent: *',
          'Allow: /',
          '',
          'Sitemap: https://taskfile.dev/sitemap.xml',
          ''
        ]
      : ['User-agent: *', 'Disallow: /', ''];
    writeFileSync(resolve(outDir, 'robots.txt'), robots.join('\n'));
  }
});

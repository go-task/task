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
import { taskDescription, taskName, ogUrl, ogImage } from './meta.ts';
import { fileURLToPath, URL } from 'node:url';
import llmstxt, { copyOrDownloadAsMarkdownButtons } from 'vitepress-plugin-llms';

const version = readFileSync(
  resolve(__dirname, '../../internal/version/version.txt'),
  'utf8'
).trim();

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
    [
      'link',
      {
        rel: 'icon',
        type: 'image/x-icon',
        href: '/img/favicon.ico',
        sizes: '48x48'
      }
    ],
    [
      'link',
      {
        rel: 'icon',
        sizes: 'any',
        type: 'image/svg+xml',
        href: '/img/logo.svg'
      }
    ],
    [
      'meta',
      { name: 'author', content: `${team.map((c) => c.name).join(', ')}` }
    ],
    // Open Graph
    ['meta', { property: 'og:type', content: 'website' }],
    ['meta', { property: 'og:site_name', content: taskName }],
    ['meta', { property: 'og:title', content: taskName }],
    ['meta', { property: 'og:description', content: taskDescription }],
    ['meta', { property: 'og:image', content: ogImage }],
    ['meta', { property: 'og:url', content: ogUrl }],
    // Twitter Card
    ['meta', { name: 'twitter:card', content: 'summary_large_image' }],
    ['meta', { name: 'twitter:site', content: '@taskfiledev' }],
    ['meta', { name: 'twitter:title', content: taskName }],
    ['meta', { name: 'twitter:description', content: taskDescription }],
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
    ]
  ],
  transformHead({ pageData }) {
    const head: HeadConfig[] = []

    // Canonical URL dynamique
    const canonicalUrl = `https://taskfile.dev/${pageData.relativePath
      .replace(/\.md$/, '')
      .replace(/index$/, '')}`
    head.push(['link', { rel: 'canonical', href: canonicalUrl }])

    // Noindex pour 404
    if (pageData.relativePath === '404.md') {
      head.push(['meta', { name: 'robots', content: 'noindex, nofollow' }])
    }

    return head
  },
  srcDir: 'src',
  cleanUrls: true,
  markdown: {
    config: (md) => {
      md.use(githubLinksPlugin, {
        baseUrl: 'https://github.com',
        repo: 'go-task/task'
      });
      md.use(tabsMarkdownPlugin);
      md.use(groupIconMdPlugin);
      md.use(copyOrDownloadAsMarkdownButtons);
    }
  },
  vite: {
    plugins: [
      llmstxt({
        ignoreFiles: [
          'index.md',
          'team.md',
          'donate.md',
          'docs/styleguide.md',
          'docs/contributing.md',
          'docs/releasing.md',
          'docs/changelog.md',
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
      '/': [
        {
          text: 'Installation',
          link: '/docs/installation'
        },
        {
          text: 'Getting Started',
          link: '/docs/getting-started'
        },
        {
          text: 'Guide',
          link: '/docs/guide'
        },
        {
          text: 'Reference',
          collapsed: true,
          items: [
            {
              text: 'Taskfile Schema',
              link: '/docs/reference/schema'
            },
            {
              text: 'Environment',
              link: '/docs/reference/environment'
            },
            {
              text: 'Configuration',
              link: '/docs/reference/config'
            },
            {
              text: 'CLI',
              link: '/docs/reference/cli'
            },
            {
              text: 'Templating',
              link: '/docs/reference/templating'
            },
            {
              text: 'Package API',
              link: '/docs/reference/package'
            }
          ]
        },
        {
          text: 'Experiments',
          collapsed: true,
          link: '/docs/experiments/',
          items: [
            {
              text: 'Env Precedence (#1038)',
              link: '/docs/experiments/env-precedence'
            },
            {
              text: 'Gentle Force (#1200)',
              link: '/docs/experiments/gentle-force'
            },
            {
              text: 'Remote Taskfiles (#1317)',
              link: '/docs/experiments/remote-taskfiles'
            }
          ]
        },
        {
          text: 'Deprecations',
          collapsed: true,
          link: '/docs/deprecations/',
          items: [
            {
              text: 'Completion Scripts',
              link: '/docs/deprecations/completion-scripts'
            },
            {
              text: 'Template Functions',
              link: '/docs/deprecations/template-functions'
            },
            {
              text: 'Version 2 Schema (#1197)',
              link: '/docs/deprecations/version-2-schema'
            }
          ]
        },
        {
          text: 'Taskfile Versions',
          link: '/docs/taskfile-versions'
        },
        {
          text: 'Integrations',
          link: '/docs/integrations'
        },
        {
          text: 'Community',
          link: '/docs/community'
        },
        {
          text: 'Style Guide',
          link: '/docs/styleguide'
        },
        {
          text: 'Contributing',
          link: '/docs/contributing'
        },
        {
          text: 'Releasing',
          link: '/docs/releasing'
        },
        {
          text: 'Changelog',
          link: '/docs/changelog'
        },
        {
          text: 'FAQ',
          link: '/docs/faq'
        }
      ],
      // Hacky to disable sidebar for these pages
      '/donate': [],
      '/team': []
    },

    socialLinks: [
      { icon: 'github', link: 'https://github.com/go-task/task' },
      { icon: 'discord', link: 'https://discord.gg/6TY36E39UK' },
      { icon: 'x', link: 'https://twitter.com/taskfiledev' },
      { icon: 'bluesky', link: 'https://bsky.app/profile/taskfile.dev' },
      { icon: 'mastodon', link: 'https://fosstodon.org/@task' }
    ],

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

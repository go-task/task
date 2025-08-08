import { defineConfig } from 'vitepress';
import githubLinksPlugin from './plugins/github-links';
import { readFileSync } from 'fs';
import { resolve } from 'path';
import { tabsMarkdownPlugin } from 'vitepress-plugin-tabs';
import {
  groupIconMdPlugin,
  groupIconVitePlugin
} from 'vitepress-plugin-group-icons';
import { team } from './team.ts';
import { ogUrl, taskDescription, taskName } from './meta.ts';

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
        href: '/img/favicon.icon',
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
      'link',
      {
        rel: 'canonical',
        href: 'https://taskfile.dev/'
      }
    ],
    [
      'meta',
      { name: 'author', content: `${team.map((c) => c.name).join(', ')}` }
    ],
    [
      'meta',
      {
        name: 'keywords',
        content:
          'task runner, build tool, taskfile, yaml build tool, go task runner, make alternative, cross-platform build tool, makefile alternative, automation tool, ci cd pipeline, developer productivity, build automation, command line tool, go binary, yaml configuration'
      }
    ],
    ['meta', { property: 'og:title', content: taskName }],
    ['meta', { property: 'og:description', content: taskDescription }],
    ['meta', { property: 'og:type', content: 'website' }],
    ['meta', { property: 'og:site_name', content: taskName }],
    ['meta', { property: 'og:url', content: ogUrl }],
    ['meta', { property: 'twitter:card', content: 'summary_large_image' }],
    ['meta', { property: 'twitter:title', content: taskName }],
    ['meta', { property: 'twitter:description', content: taskDescription }]
  ],
  srcDir: 'src',
  cleanUrls: true,
  rewrites(id) {
    return id.replace(/^docs\//, '');
  },
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
    plugins: [groupIconVitePlugin()]
  },

  themeConfig: {
    logo: '/logo.svg',
    carbonAds: {
      code: 'CESI65QJ',
      placement: 'taskfiledev'
    },
    search: {
      provider: 'local'
      // options: {
      // 	appId: '...',
      // 	apiKey: '...',
      // 	indexName: '...'
      // }
    },
    nav: [
      { text: 'Home', link: '/' },
      {
        text: 'Docs',
        link: '/getting-started',
        activeMatch: '^/(?!donate|team|blog).'
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
          link: '/installation'
        },
        {
          text: 'Getting Started',
          link: '/getting-started'
        },
        {
          text: 'Usage',
          link: '/usage'
        },
        {
          text: 'Reference',
          collapsed: true,
          items: [
            {
              text: 'CLI',
              link: '/reference/cli'
            },
            {
              text: 'Schema',
              link: '/reference/schema'
            },
            {
              text: 'Templating',
              link: '/reference/templating'
            },
            {
              text: 'Package API',
              link: '/reference/package'
            }
          ]
        },
        {
          text: 'Experiments',
          collapsed: true,
          link: '/experiments/',
          items: [
            {
              text: 'Env Precedence (#1038)',
              link: '/experiments/env-precedence'
            },
            {
              text: 'Gentle Force (#1200)',
              link: '/experiments/gentle-force'
            },
            {
              text: 'Remote Taskfiles (#1317)',
              link: '/experiments/remote-taskfiles'
            }
          ]
        },
        {
          text: 'Deprecations',
          collapsed: true,
          link: '/deprecations/',
          items: [
            {
              text: 'Completion Scripts',
              link: '/deprecations/completion-scripts'
            },
            {
              text: 'Template Functions',
              link: '/deprecations/template-functions'
            },
            {
              text: 'Version 2 Schema (#1197)',
              link: '/deprecations/version-2-schema'
            }
          ]
        },
        {
          text: 'Taskfile Versions',
          link: '/taskfile-versions'
        },
        {
          text: 'Integrations',
          link: '/integrations'
        },
        {
          text: 'Community',
          link: '/community'
        },
        {
          text: 'Style Guide',
          link: '/styleguide'
        },
        {
          text: 'Contributing',
          link: '/contributing'
        },
        {
          text: 'Releasing',
          link: '/releasing'
        },
        {
          text: 'Changelog',
          link: '/changelog'
        },
        {
          text: 'FAQ',
          link: '/faq'
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
    ]
  }
});

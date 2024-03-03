import type {Config} from '@docusaurus/types';
import type * as Preset from '@docusaurus/preset-classic';
import { EnumChangefreq } from 'sitemap';

import remarkGithub from 'remark-github';
import remarkGfm from 'remark-gfm';

import { DISCORD_URL } from './constants';
import { GITHUB_URL } from './constants';
import { MASTODON_URL } from './constants';
import { TWITTER_URL } from './constants';

import lightCodeTheme from './src/themes/prismLight';
import darkCodeTheme from './src/themes/prismDark';

import { getTranslationProgress } from './src/api/crowdin.js';
const translationProgress = getTranslationProgress();

const config: Config = {
  title: 'Task',
  tagline: 'A task runner / simpler Make alternative written in Go ',
  url: 'https://taskfile.dev',
  baseUrl: '/',
  onBrokenLinks: 'throw',
  onBrokenMarkdownLinks: 'throw',
  favicon: 'img/favicon.ico',

  organizationName: 'go-task',
  projectName: 'task',
  deploymentBranch: 'gh-pages',

  i18n: {
    defaultLocale: 'en',
    locales: [
      'en',
      'es-ES',
      'fr-FR',
      'ja-JP',
      'pt-BR',
      'ru-RU',
      'tr-TR',
      'zh-Hans'
    ],
    localeConfigs: {
      en: {
        label: 'English',
        direction: 'ltr',
        htmlLang: 'en-US'
      },
      'es-ES': {
        label: `Español (${translationProgress['es-ES'] || 0}%)`,
        direction: 'ltr',
        htmlLang: 'es-ES'
      },
      'fr-FR': {
        label: `Français (${translationProgress['fr'] || 0}%)`,
        direction: 'ltr',
        htmlLang: 'fr-FR'
      },
      'ja-JP': {
        label: `日本語 (${translationProgress['ja'] || 0}%)`,
        direction: 'ltr',
        htmlLang: 'ja-JP'
      },
      'pt-BR': {
        label: `Português (${translationProgress['pt-BR'] || 0}%)`,
        direction: 'ltr',
        htmlLang: 'pt-BR'
      },
      'ru-RU': {
        label: `Pусский (${translationProgress['ru'] || 0}%)`,
        direction: 'ltr',
        htmlLang: 'ru-RU'
      },
      'tr-TR': {
        label: `Türkçe (${translationProgress['tr'] || 0}%)`,
        direction: 'ltr',
        htmlLang: 'tr-TR'
      },
      'zh-Hans': {
        label: `简体中文 (${translationProgress['zh-CN'] || 0}%)`,
        direction: 'ltr',
        htmlLang: 'zh-Hans'
      }
    }
  },

  presets: [
    [
      'classic',
      {
        docs: {
          routeBasePath: '/',
          sidebarPath: './sidebars.ts',
          remarkPlugins: [remarkGithub, remarkGfm],
          includeCurrentVersion: true,
          versions: {
            current: {
              label: `Next 🚧`,
              path: 'next',
              badge: false
            },
            latest: {
              label: 'Latest',
              badge: false
            }
          }
        },
        blog: {},
        theme: {
          customCss: [
            './src/css/custom.css',
            './src/css/carbon.css',
          ]
        },
        gtag: {
          trackingID: 'G-4RT25NXQ7N',
          anonymizeIP: true
        },
        sitemap: {
          changefreq: EnumChangefreq.WEEKLY,
          priority: 0.5,
          ignorePatterns: ['/tags/**']
        }
      } satisfies Preset.Options,
    ]
  ],

  scripts: [
    {
      src: '/js/carbon.js',
      async: true
    }
  ],

  themeConfig:{
    metadata: [
      {
        name: 'og:image',
        content: 'https://taskfile.dev/img/og-image.png'
      }
    ],
    navbar: {
      title: 'Task',
      logo: {
        alt: 'Task Logo',
        src: 'img/logo.svg'
      },
      items: [
        {
          type: 'doc',
          docId: 'installation',
          position: 'left',
          label: 'Installation'
        },
        {
          type: 'doc',
          docId: 'usage',
          position: 'left',
          label: 'Usage'
        },
        {
          type: 'doc',
          docId: 'api_reference',
          position: 'left',
          label: 'API'
        },
        {
          to: 'blog',
          label: 'Blog',
          position: 'left'
        },
        {
          type: 'doc',
          docId: 'donate',
          position: 'left',
          label: 'Donate'
        },
        {
          type: 'localeDropdown',
          position: 'left',
          dropdownItemsAfter: [
            {
              to: '/translate/',
              label: 'Help Us Translate'
            }
          ]
        },
        {
          type: 'docsVersionDropdown',
          position: 'right',
          dropdownActiveClassDisabled: true,
        },
        {
          href: GITHUB_URL,
          label: 'GitHub',
          position: 'right'
        },
        {
          href: TWITTER_URL,
          label: 'Twitter',
          position: 'right'
        },
        {
          href: MASTODON_URL,
          label: 'Mastodon',
          rel: 'me',
          position: 'right'
        },
        {
          href: DISCORD_URL,
          label: 'Discord',
          position: 'right'
        }
      ]
    },
    footer: {
      style: 'dark',
      links: [
        {
          title: 'Pages',
          items: [
            {
              label: 'Installation',
              to: '/installation/'
            },
            {
              label: 'Usage',
              to: '/usage/'
            },
            {
              label: 'Donate',
              to: '/donate/'
            }
          ]
        },
        {
          title: 'Community',
          items: [
            {
              label: 'GitHub',
              href: GITHUB_URL
            },
            {
              label: 'Twitter',
              href: TWITTER_URL
            },
            {
              label: 'Mastodon',
              href: MASTODON_URL,
              rel: 'me'
            },
            {
              label: 'Discord',
              href: DISCORD_URL
            },
            {
              label: 'OpenCollective',
              href: 'https://opencollective.com/task'
            }
          ]
        },
        {
          items: [
            {
              html: '<a target="_blank" href="https://www.netlify.com"><img src="https://www.netlify.com/v3/img/components/netlify-color-accent.svg" alt="Deploys by Netlify" /></a>'
            }
          ]
        }
      ]
    },
    prism: {
      theme: lightCodeTheme,
      darkTheme: darkCodeTheme,
      additionalLanguages: [
        "bash", // aka. shell
        "json",
        "powershell"
      ]
    },
    // NOTE(@andreynering): Don't worry, these keys are meant to be public =)
    algolia: {
      appId: '7IZIJ13AI7',
      apiKey: '34b64ae4fc8d9da43d9a13d9710aaddc',
      indexName: 'taskfile'
    }
  } satisfies Preset.ThemeConfig,
};

export default config;

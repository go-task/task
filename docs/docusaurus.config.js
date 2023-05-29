// @ts-check
// Note: type annotations allow type checking and IDEs autocompletion

const {
  DISCORD_URL,
  GITHUB_URL,
  MASTODON_URL,
  TWITTER_URL
} = require('./constants');
const lightCodeTheme = require('./src/themes/prismLight');
const darkCodeTheme = require('./src/themes/prismDark');

const { getTranslationProgress } = require('./src/api/crowdin.js');

const getConfig = async () => {
  const translationProgress = await getTranslationProgress();

  /** @type {import('@docusaurus/types').Config} */
  const config = {
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
      locales: ['en', 'fr-FR', 'pt-BR', 'ru-RU', 'es-ES', 'zh-Hans'],
      localeConfigs: {
        en: {
          label: 'English',
          direction: 'ltr',
          htmlLang: 'en-US'
        },
        'fr-FR': {
          label: `Français (${translationProgress['fr'] || 0}%)`,
          direction: 'ltr',
          htmlLang: 'fr-FR'
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
        'es-ES': {
          label: `Español (${translationProgress['es-ES'] || 0}%)`,
          direction: 'ltr',
          htmlLang: 'es-ES'
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
        /** @type {import('@docusaurus/preset-classic').Options} */
        ({
          docs: {
            routeBasePath: '/',
            sidebarPath: require.resolve('./sidebars.js')
          },
          blog: false,
          theme: {
            customCss: [require.resolve('./src/css/custom.css')]
          },
          gtag: {
            trackingID: 'G-4RT25NXQ7N',
            anonymizeIP: true
          },
          sitemap: {
            changefreq: 'weekly',
            priority: 0.5,
            ignorePatterns: ['/tags/**']
          }
        })
      ]
    ],

    themeConfig:
      /** @type {import('@docusaurus/preset-classic').ThemeConfig} */
      ({
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
            }
          ]
        },
        prism: {
          theme: lightCodeTheme,
          darkTheme: darkCodeTheme
        },
        // NOTE(@andreynering): Don't worry, these keys are meant to be public =)
        algolia: {
          appId: '7IZIJ13AI7',
          apiKey: '34b64ae4fc8d9da43d9a13d9710aaddc',
          indexName: 'taskfile'
        }
      })
  };

  return config;
};

module.exports = getConfig;

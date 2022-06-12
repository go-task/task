// @ts-check
// Note: type annotations allow type checking and IDEs autocompletion

const lightCodeTheme = require('prism-react-renderer/themes/github');
const darkCodeTheme = require('prism-react-renderer/themes/dracula');

const GITHUB_URL = 'https://github.com/go-task/task';
const DISCORD_URL = 'https://discord.gg/6TY36E39UK';

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
    locales: ['en']
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
          customCss: [
            require.resolve('./src/css/custom.css'),
            require.resolve('./src/css/carbon.css')
          ]
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
            href: GITHUB_URL,
            label: 'GitHub',
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
    }),

  scripts: [
    {
      src: '/js/carbon.js',
      async: true
    }
  ]
};

module.exports = config;

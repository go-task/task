import type { DefaultTheme } from 'vitepress';

// Navigation for the `/docs` section. next.ts is the source of both sidebars;
// cmd/release copies it over latest.ts alongside the content it describes. See
// the "Documentation channels" section of website/src/next/docs/contributing.md.
//
// Grouped by what the reader is trying to do: get going, learn Task, look
// something up, follow the project. The DocSearch crawler puts the active
// sidebar section into hierarchy.lvl0, so these labels are also the breadcrumbs
// on every search result.
export const sidebar: DefaultTheme.SidebarItem[] = [
  {
    text: 'Overview',
    link: '/docs/'
  },
  {
    text: 'Getting Started',
    items: [
      {
        text: 'Installation',
        link: '/docs/installation'
      },
      {
        text: 'Quick Start',
        link: '/docs/getting-started'
      },
      {
        text: 'Editors and Integrations',
        link: '/docs/integrations'
      }
    ]
  },
  {
    text: 'Guide',
    link: '/docs/guide/',
    items: [
      {
        text: 'Running tasks',
        link: '/docs/guide/running-tasks'
      },
      {
        text: 'Defining tasks',
        link: '/docs/guide/defining-tasks'
      },
      {
        text: 'Passing arguments',
        link: '/docs/guide/arguments'
      },
      {
        text: 'Variables',
        link: '/docs/guide/variables'
      },
      {
        text: 'Environment variables',
        link: '/docs/guide/environment'
      },
      {
        text: 'Required variables and prompts',
        link: '/docs/guide/required-variables'
      },
      {
        text: 'Dependencies and task calls',
        link: '/docs/guide/dependencies'
      },
      {
        text: 'Skipping work that is up to date',
        link: '/docs/guide/up-to-date'
      },
      {
        text: 'Conditional execution',
        link: '/docs/guide/conditional-execution'
      },
      {
        text: 'Loops',
        link: '/docs/guide/loops'
      },
      {
        text: 'Including other Taskfiles',
        link: '/docs/guide/includes'
      },
      {
        text: 'Remote Taskfiles',
        link: '/docs/remote-taskfiles'
      },
      {
        text: 'Output and logging',
        link: '/docs/guide/output'
      },
      {
        text: 'Platform-specific behaviour',
        link: '/docs/guide/platforms'
      },
      {
        text: 'Watch mode',
        link: '/docs/guide/watch'
      }
    ]
  },
  {
    text: 'Reference',
    collapsed: false,
    items: [
      {
        text: 'Taskfile Schema',
        link: '/docs/reference/schema'
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
        text: 'Environment',
        link: '/docs/reference/environment'
      },
      {
        text: 'Configuration',
        link: '/docs/reference/config'
      },
      {
        text: 'Package API',
        link: '/docs/reference/package'
      }
    ]
  },
  {
    text: 'Project',
    collapsed: true,
    items: [
      {
        text: 'Changelog',
        link: '/docs/changelog'
      },
      {
        text: 'FAQ',
        link: '/docs/faq'
      },
      {
        text: 'Taskfile Versions',
        link: '/docs/taskfile-versions'
      },
      {
        text: 'Community',
        link: '/docs/community'
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
        text: 'Security',
        collapsed: true,
        link: '/docs/security/',
        items: [
          {
            text: 'Incident Response Plan',
            link: '/docs/security/incident-response-plan'
          },
          {
            text: 'Threat Model',
            link: '/docs/security/threat-model'
          }
        ]
      }
    ]
  },
  {
    text: 'Contributing',
    collapsed: true,
    items: [
      {
        text: 'Contributing',
        link: '/docs/contributing'
      },
      {
        text: 'Style Guide',
        link: '/docs/styleguide'
      },
      {
        text: 'Releasing',
        link: '/docs/releasing'
      }
    ]
  }
];

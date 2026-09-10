import type { DefaultTheme } from 'vitepress';

// Navigation for the `/docs` section. next.ts is the source of both sidebars;
// cmd/release copies it over latest.ts alongside the content it describes. See
// the "Documentation channels" section of website/src/next/docs/contributing.md.
export const sidebar: DefaultTheme.SidebarItem[] = [
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
    text: 'Remote Taskfiles',
    link: '/docs/remote-taskfiles'
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
        text: 'Prefix Matching (#2947)',
        link: '/docs/experiments/prefix-matching'
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
  },
  {
    text: 'Changelog',
    link: '/docs/changelog'
  },
  {
    text: 'FAQ',
    link: '/docs/faq'
  }
];

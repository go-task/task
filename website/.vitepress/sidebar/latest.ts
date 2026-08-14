import type { DefaultTheme } from 'vitepress';

// Sidebar for the `/docs` section. `next.ts` navigates `src/docs`, and
// cmd/release copies it to `latest.ts` alongside the docs it describes, so
// that the published site keeps the navigation of the released version.
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

// Sidebar for the `/blog` section, split the same way as the docs: an entry
// added here only reaches taskfile.dev once cmd/release promotes this file,
// so a post announcing a feature ships with the release that carries it.
export const blogSidebar: DefaultTheme.SidebarItem[] = [
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
];

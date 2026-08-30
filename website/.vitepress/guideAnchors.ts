// Where each section of the old single-page guide went when it was
// split up. Netlify never sees the URL fragment, so a _redirects rule
// cannot route these; GuideRedirect.vue resolves them in the browser.
export const guideAnchors: Record<string, string> = {
  'running-taskfiles': '/docs/guide/running-tasks',
  'supported-file-names': '/docs/guide/running-tasks#supported-file-names',
  'running-a-taskfile-from-a-subdirectory':
    '/docs/guide/running-tasks#running-a-taskfile-from-a-subdirectory',
  'running-a-global-taskfile':
    '/docs/guide/running-tasks#running-a-global-taskfile',
  'running-a-taskfile-from-stdin':
    '/docs/guide/running-tasks#running-a-taskfile-from-stdin',
  'running-a-remote-taskfile':
    '/docs/remote-taskfiles#specifying-a-remote-entrypoint',
  'environment-variables': '/docs/guide/environment',
  task: '/docs/guide/environment#task',
  'env-files': '/docs/guide/environment#env-files',
  'including-other-taskfiles': '/docs/guide/includes',
  'remote-taskfiles': '/docs/guide/includes#remote-taskfiles',
  'os-specific-taskfiles': '/docs/guide/includes#os-specific-taskfiles',
  'directory-of-included-taskfile':
    '/docs/guide/includes#directory-of-included-taskfile',
  'optional-includes': '/docs/guide/includes#optional-includes',
  'internal-includes': '/docs/guide/includes#internal-includes',
  'flatten-includes': '/docs/guide/includes#flatten-includes',
  'exclude-tasks-from-being-included':
    '/docs/guide/includes#exclude-tasks-from-being-included',
  'vars-of-included-taskfiles':
    '/docs/guide/includes#vars-of-included-taskfiles',
  'namespace-aliases': '/docs/guide/includes#namespace-aliases',
  'internal-tasks': '/docs/guide/defining-tasks#internal-tasks',
  'task-directory': '/docs/guide/defining-tasks#task-directory',
  'task-dependencies': '/docs/guide/dependencies#task-dependencies',
  'fail-fast-dependencies': '/docs/guide/dependencies#fail-fast-dependencies',
  'platform-specific-tasks-and-commands':
    '/docs/guide/platforms#platform-specific-tasks-and-commands',
  'calling-another-task': '/docs/guide/dependencies#calling-another-task',
  'prevent-unnecessary-work': '/docs/guide/up-to-date',
  'by-fingerprinting-locally-generated-files-and-their-sources':
    '/docs/guide/up-to-date#by-fingerprinting-locally-generated-files-and-their-sources',
  'using-programmatic-checks-to-indicate-a-task-is-up-to-date':
    '/docs/guide/up-to-date#using-programmatic-checks-to-indicate-a-task-is-up-to-date',
  'using-programmatic-checks-to-cancel-the-execution-of-a-task-and-its-dependencies':
    '/docs/guide/conditional-execution#using-programmatic-checks-to-cancel-the-execution-of-a-task-and-its-dependencies',
  'conditional-execution-with-if':
    '/docs/guide/conditional-execution#conditional-execution-with-if',
  'task-level-if': '/docs/guide/conditional-execution#task-level-if',
  'command-level-if': '/docs/guide/conditional-execution#command-level-if',
  'using-templates-in-if-conditions':
    '/docs/guide/conditional-execution#using-templates-in-if-conditions',
  'using-if-with-for-loops':
    '/docs/guide/conditional-execution#using-if-with-for-loops',
  'if-vs-preconditions':
    '/docs/guide/conditional-execution#if-vs-preconditions',
  'limiting-when-tasks-run':
    '/docs/guide/conditional-execution#limiting-when-tasks-run',
  'ensuring-required-variables-are-set':
    '/docs/guide/required-variables#ensuring-required-variables-are-set',
  'ensuring-required-variables-have-allowed-values':
    '/docs/guide/required-variables#ensuring-required-variables-have-allowed-values',
  'using-variable-references-for-enum-values':
    '/docs/guide/required-variables#using-variable-references-for-enum-values',
  'prompting-for-missing-variables-interactively':
    '/docs/guide/required-variables#prompting-for-missing-variables-interactively',
  variables: '/docs/guide/variables',
  'dynamic-variables': '/docs/guide/variables#dynamic-variables',
  'referencing-other-variables':
    '/docs/guide/variables#referencing-other-variables',
  'parsing-json-yaml-into-map-variables':
    '/docs/guide/variables#parsing-json-yaml-into-map-variables',
  'secret-variables': '/docs/guide/variables#secret-variables',
  'looping-over-values': '/docs/guide/loops',
  'looping-over-a-static-list': '/docs/guide/loops#looping-over-a-static-list',
  'looping-over-a-matrix': '/docs/guide/loops#looping-over-a-matrix',
  'looping-over-your-task-s-sources-or-generated-files':
    '/docs/guide/loops#looping-over-your-task-s-sources-or-generated-files',
  'looping-over-variables': '/docs/guide/loops#looping-over-variables',
  'renaming-variables': '/docs/guide/loops#renaming-variables',
  'looping-over-tasks': '/docs/guide/loops#looping-over-tasks',
  'looping-over-dependencies': '/docs/guide/loops#looping-over-dependencies',
  'forwarding-cli-arguments-to-commands':
    '/docs/guide/arguments#forwarding-cli-arguments-to-commands',
  'wildcard-arguments': '/docs/guide/arguments#wildcard-arguments',
  'doing-task-cleanup-with-defer':
    '/docs/guide/dependencies#doing-task-cleanup-with-defer',
  help: '/docs/guide/defining-tasks#help',
  'display-summary-of-task':
    '/docs/guide/defining-tasks#display-summary-of-task',
  'task-aliases': '/docs/guide/defining-tasks#task-aliases',
  'overriding-task-name': '/docs/guide/defining-tasks#overriding-task-name',
  'warning-prompts': '/docs/guide/required-variables#warning-prompts',
  'silent-mode': '/docs/guide/output#silent-mode',
  'dry-run-mode': '/docs/guide/running-tasks#dry-run-mode',
  'ignore-errors': '/docs/guide/output#ignore-errors',
  'output-syntax': '/docs/guide/output#output-syntax',
  'ci-integration': '/docs/guide/output#ci-integration',
  'colored-output': '/docs/guide/output#colored-output',
  'error-annotations': '/docs/guide/output#error-annotations',
  'interactive-cli-application':
    '/docs/guide/running-tasks#interactive-cli-application',
  'short-task-syntax': '/docs/guide/defining-tasks#short-task-syntax',
  'set-and-shopt': '/docs/guide/platforms#set-and-shopt',
  'watch-tasks': '/docs/guide/watch'
};

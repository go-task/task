// Where each section of the old single-page guide went when it was
// split up. Netlify never sees the URL fragment, so a _redirects rule
// cannot route these; GuideRedirect.vue resolves them in the browser.
export const guideAnchors: Record<string, string> = {
  'running-taskfiles': '/docs/running-tasks',
  'supported-file-names': '/docs/running-tasks#supported-file-names',
  'running-a-taskfile-from-a-subdirectory':
    '/docs/running-tasks#running-a-taskfile-from-a-subdirectory',
  'running-a-global-taskfile': '/docs/running-tasks#running-a-global-taskfile',
  'running-a-taskfile-from-stdin':
    '/docs/running-tasks#running-a-taskfile-from-stdin',
  'running-a-remote-taskfile':
    '/docs/remote-taskfiles#specifying-a-remote-entrypoint',
  'environment-variables': '/docs/environment',
  task: '/docs/environment#task',
  'env-files': '/docs/environment#env-files',
  'including-other-taskfiles': '/docs/includes',
  'remote-taskfiles': '/docs/includes#remote-taskfiles',
  'os-specific-taskfiles': '/docs/includes#os-specific-taskfiles',
  'directory-of-included-taskfile':
    '/docs/includes#directory-of-included-taskfile',
  'optional-includes': '/docs/includes#optional-includes',
  'internal-includes': '/docs/includes#internal-includes',
  'flatten-includes': '/docs/includes#flatten-includes',
  'exclude-tasks-from-being-included':
    '/docs/includes#exclude-tasks-from-being-included',
  'vars-of-included-taskfiles': '/docs/includes#vars-of-included-taskfiles',
  'namespace-aliases': '/docs/includes#namespace-aliases',
  'internal-tasks': '/docs/defining-tasks#internal-tasks',
  'task-directory': '/docs/defining-tasks#task-directory',
  'task-dependencies': '/docs/dependencies#task-dependencies',
  'fail-fast-dependencies': '/docs/dependencies#fail-fast-dependencies',
  'platform-specific-tasks-and-commands':
    '/docs/platforms#platform-specific-tasks-and-commands',
  'calling-another-task': '/docs/dependencies#calling-another-task',
  'prevent-unnecessary-work': '/docs/up-to-date',
  'by-fingerprinting-locally-generated-files-and-their-sources':
    '/docs/up-to-date#by-fingerprinting-locally-generated-files-and-their-sources',
  'using-programmatic-checks-to-indicate-a-task-is-up-to-date':
    '/docs/up-to-date#using-programmatic-checks-to-indicate-a-task-is-up-to-date',
  'using-programmatic-checks-to-cancel-the-execution-of-a-task-and-its-dependencies':
    '/docs/conditional-execution#using-programmatic-checks-to-cancel-the-execution-of-a-task-and-its-dependencies',
  'conditional-execution-with-if':
    '/docs/conditional-execution#conditional-execution-with-if',
  'task-level-if': '/docs/conditional-execution#task-level-if',
  'command-level-if': '/docs/conditional-execution#command-level-if',
  'using-templates-in-if-conditions':
    '/docs/conditional-execution#using-templates-in-if-conditions',
  'using-if-with-for-loops':
    '/docs/conditional-execution#using-if-with-for-loops',
  'if-vs-preconditions': '/docs/conditional-execution#if-vs-preconditions',
  'limiting-when-tasks-run':
    '/docs/conditional-execution#limiting-when-tasks-run',
  'ensuring-required-variables-are-set':
    '/docs/required-variables#ensuring-required-variables-are-set',
  'ensuring-required-variables-have-allowed-values':
    '/docs/required-variables#ensuring-required-variables-have-allowed-values',
  'using-variable-references-for-enum-values':
    '/docs/required-variables#using-variable-references-for-enum-values',
  'prompting-for-missing-variables-interactively':
    '/docs/required-variables#prompting-for-missing-variables-interactively',
  variables: '/docs/variables',
  'dynamic-variables': '/docs/variables#dynamic-variables',
  'referencing-other-variables': '/docs/variables#referencing-other-variables',
  'parsing-json-yaml-into-map-variables':
    '/docs/variables#parsing-json-yaml-into-map-variables',
  'secret-variables': '/docs/variables#secret-variables',
  'looping-over-values': '/docs/loops',
  'looping-over-a-static-list': '/docs/loops#looping-over-a-static-list',
  'looping-over-a-matrix': '/docs/loops#looping-over-a-matrix',
  'looping-over-your-task-s-sources-or-generated-files':
    '/docs/loops#looping-over-your-task-s-sources-or-generated-files',
  'looping-over-variables': '/docs/loops#looping-over-variables',
  'renaming-variables': '/docs/loops#renaming-variables',
  'looping-over-tasks': '/docs/loops#looping-over-tasks',
  'looping-over-dependencies': '/docs/loops#looping-over-dependencies',
  'forwarding-cli-arguments-to-commands':
    '/docs/arguments#forwarding-cli-arguments-to-commands',
  'wildcard-arguments': '/docs/arguments#wildcard-arguments',
  'doing-task-cleanup-with-defer':
    '/docs/dependencies#doing-task-cleanup-with-defer',
  help: '/docs/defining-tasks#help',
  'display-summary-of-task': '/docs/defining-tasks#display-summary-of-task',
  'task-aliases': '/docs/defining-tasks#task-aliases',
  'overriding-task-name': '/docs/defining-tasks#overriding-task-name',
  'warning-prompts': '/docs/required-variables#warning-prompts',
  'silent-mode': '/docs/output#silent-mode',
  'dry-run-mode': '/docs/running-tasks#dry-run-mode',
  'ignore-errors': '/docs/output#ignore-errors',
  'output-syntax': '/docs/output#output-syntax',
  'ci-integration': '/docs/output#ci-integration',
  'colored-output': '/docs/output#colored-output',
  'error-annotations': '/docs/output#error-annotations',
  'interactive-cli-application':
    '/docs/running-tasks#interactive-cli-application',
  'short-task-syntax': '/docs/defining-tasks#short-task-syntax',
  'set-and-shopt': '/docs/platforms#set-and-shopt',
  'watch-tasks': '/docs/watch'
};

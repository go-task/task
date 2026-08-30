<script setup lang="ts">
import { computed, onUnmounted, ref } from 'vue';
import { VPHomeSponsors } from 'vitepress/theme';
import { sponsors } from '../sponsors';
import AdoptersCarousel from './AdoptersCarousel.vue';
import { data as example } from './homeExample.data';

const installCommands = {
  unix: 'sh -c "$(curl --location https://taskfile.dev/install.sh)" -- -d -b ~/.local/bin',
  windows: 'winget install Task.Task'
} as const;
type InstallPlatform = keyof typeof installCommands;

const installPlatform = ref<InstallPlatform>('unix');
const installCommand = computed(() => installCommands[installPlatform.value]);
const installPlatformLabel = computed(() =>
  installPlatform.value === 'windows' ? 'Windows' : 'macOS and Linux'
);
const copyState = ref<'idle' | 'copied' | 'failed'>('idle');
const commandEl = ref<HTMLElement>();
let copyResetTimer: number | undefined;

const copyLabel = computed(() => {
  if (copyState.value === 'copied') return 'Copied';
  if (copyState.value === 'failed') return 'Copy failed';
  return 'Copy';
});

const copyAnnouncement = computed(() => {
  if (copyState.value === 'copied') {
    return `${installPlatformLabel.value} install command copied to clipboard`;
  }
  if (copyState.value === 'failed') {
    return 'The install command could not be copied, so it has been selected';
  }
  return '';
});

const copyAriaLabel = computed(() => {
  if (copyState.value === 'copied') {
    return `${installPlatformLabel.value} install command copied`;
  }
  if (copyState.value === 'failed') {
    return `Copy ${installPlatformLabel.value} install command again`;
  }
  return `Copy ${installPlatformLabel.value} install command`;
});

function selectInstallPlatform(platform: InstallPlatform) {
  if (installPlatform.value === platform) return;

  window.clearTimeout(copyResetTimer);
  copyResetTimer = undefined;
  copyState.value = 'idle';
  installPlatform.value = platform;
  window.getSelection()?.removeAllRanges();
}

function resetCopyState() {
  window.clearTimeout(copyResetTimer);
  copyResetTimer = window.setTimeout(() => {
    copyState.value = 'idle';
  }, 2000);
}

async function copyInstallCommand() {
  try {
    await navigator.clipboard.writeText(installCommand.value);
    copyState.value = 'copied';
  } catch {
    copyState.value = 'failed';
    selectInstallCommand();
  }
  resetCopyState();
}

// Without the clipboard API — insecure context, denied permission — the least
// we can do is select the command so a manual copy is one keystroke away.
function selectInstallCommand() {
  const node = commandEl.value;
  if (!node) return;

  const range = document.createRange();
  range.selectNodeContents(node);
  const selection = window.getSelection();
  selection?.removeAllRanges();
  selection?.addRange(range);
}

onUnmounted(() => window.clearTimeout(copyResetTimer));
</script>

<template>
  <main class="home-content">
    <AdoptersCarousel />

    <section class="quick-start" aria-labelledby="quick-start-title">
      <div class="section-heading">
        <p class="eyebrow">
          <span aria-hidden="true">//</span> FROM ZERO TO YOUR FIRST TASK
        </p>
        <h2 id="quick-start-title">A useful Taskfile in a few minutes</h2>
        <p>
          Install one binary, add readable project commands, and give everyone
          the same entry point—locally and in CI.
        </p>
      </div>

      <div
        class="install-platforms"
        role="group"
        aria-label="Choose an operating system"
      >
        <span>Install on:</span>
        <button
          type="button"
          :aria-pressed="installPlatform === 'unix'"
          @click="selectInstallPlatform('unix')"
        >
          macOS / Linux
        </button>
        <button
          type="button"
          :aria-pressed="installPlatform === 'windows'"
          @click="selectInstallPlatform('windows')"
        >
          Windows
        </button>
      </div>

      <div class="install-command">
        <span aria-hidden="true" class="prompt">$</span>
        <code
          ref="commandEl"
          aria-live="polite"
          :aria-label="`${installPlatformLabel} install command`"
          >{{ installCommand }}</code
        >
        <button
          type="button"
          :aria-label="copyAriaLabel"
          data-umami-event="home-install-copy"
          @click="copyInstallCommand"
        >
          {{ copyLabel }}
        </button>
        <span class="visually-hidden" aria-live="polite">
          {{ copyAnnouncement }}
        </span>
      </div>
      <p class="install-note">
        Need another installation method? Task is also available through
        Homebrew, Scoop, npm, apt, dnf, apk, and more.
        <a href="/docs/installation" data-umami-event="home-install-options"
          >See every installation method</a
        >.
      </p>

      <div class="example" aria-label="Taskfile and terminal example">
        <div class="code-panel">
          <div class="panel-title">
            <span class="status-dot" aria-hidden="true"></span>
            Taskfile.yml
          </div>
          <div class="code-body" v-html="example.taskfile"></div>
        </div>

        <div class="code-panel terminal-panel">
          <div class="panel-title">
            <span class="status-dot" aria-hidden="true"></span>
            Terminal
          </div>
          <div class="code-body">
            <pre><code v-html="example.terminal"></code></pre>
          </div>
        </div>
      </div>

      <div class="example-actions">
        <a
          class="primary-link"
          href="/docs/getting-started"
          data-umami-event="home-example-get-started"
          >Build your first Taskfile <span aria-hidden="true">→</span></a
        >
        <a href="/docs/guide" data-umami-event="home-example-guide">
          Explore the guide
        </a>
      </div>
    </section>

    <section class="choose-path" aria-labelledby="choose-path-title">
      <div class="section-heading">
        <p class="eyebrow">
          <span aria-hidden="true">//</span> FIND THE RIGHT ANSWER
        </p>
        <h2 id="choose-path-title">Start broad or jump straight to syntax</h2>
        <p>
          The documentation has separate entry points for learning, exact
          lookup, and machine-readable access.
        </p>
      </div>

      <div class="path-grid">
        <article>
          <span class="path-number">01</span>
          <h3>I'm new to Task</h3>
          <p>
            Install Task, create a small Taskfile, then grow it one workflow at
            a time with focused examples.
          </p>
          <ul>
            <li><a href="/docs/installation">Install Task</a></li>
            <li>
              <a href="/docs/getting-started">Create your first Taskfile</a>
            </li>
            <li><a href="/docs/guide">Read the guide</a></li>
          </ul>
        </article>

        <article>
          <span class="path-number">02</span>
          <h3>I need exact syntax</h3>
          <p>
            Look up a Taskfile field, CLI flag, environment variable, or
            template function without reading a long tutorial.
          </p>
          <ul>
            <li><a href="/docs/reference/schema">Taskfile schema</a></li>
            <li><a href="/docs/reference/cli">CLI flags</a></li>
            <li><a href="/docs/reference/templating">Template functions</a></li>
          </ul>
        </article>

        <article>
          <span class="path-number">03</span>
          <h3>I'm browsing with an agent</h3>
          <p>
            Discover the documentation from a compact map, retrieve it as one
            text bundle, or jump directly to the Taskfile schema.
          </p>
          <ul>
            <li><a href="/llms.txt">LLM documentation map</a></li>
            <li><a href="/llms-full.txt">Full documentation bundle</a></li>
            <li><a href="/docs/reference/schema">Taskfile schema</a></li>
          </ul>
        </article>
      </div>
    </section>

    <div class="sponsors">
      <VPHomeSponsors
        v-if="sponsors"
        message="Task is free and open source, made possible by wonderful sponsors."
        :data="sponsors"
      />
    </div>
  </main>
</template>

<style scoped>
.home-content {
  border-top: 1px solid var(--vp-c-divider);
  margin-top: 4rem;
}

.quick-start,
.choose-path {
  max-width: 1152px;
  margin: 0 auto;
  padding: 6rem 24px 0;
}

.quick-start {
  padding-top: 4.5rem;
}

.section-heading {
  max-width: 720px;
  margin-bottom: 2rem;
}

.section-heading .eyebrow {
  margin: 0 0 0.75rem;
  color: var(--vp-c-brand-1);
  font-family: var(--vp-font-family-mono);
  font-size: 0.75rem;
  font-weight: 700;
  letter-spacing: 0.08em;
}

.section-heading h2 {
  margin: 0;
  border: 0;
  color: var(--vp-c-text-1);
  font-size: clamp(1.75rem, 4vw, 2.5rem);
  letter-spacing: -0.035em;
  line-height: 1.15;
}

.section-heading > p:last-child {
  max-width: 640px;
  margin: 1rem 0 0;
  color: var(--vp-c-text-2);
  font-size: 1.05rem;
  line-height: 1.7;
}

.install-platforms {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 0.4rem;
  max-width: 940px;
  margin-bottom: 0.6rem;
  color: var(--vp-c-text-2);
  font-size: 0.78rem;
  font-weight: 600;
}

.install-platforms button {
  border: 1px solid var(--vp-c-divider);
  border-radius: 999px;
  background: transparent;
  color: var(--vp-c-text-2);
  cursor: pointer;
  font: inherit;
  padding: 0.35rem 0.65rem;
}

.install-platforms button[aria-pressed='true'] {
  border-color: var(--vp-c-brand-1);
  background: var(--vp-c-brand-soft);
  color: var(--vp-c-brand-1);
}

.install-platforms button:hover,
.install-platforms button:focus-visible {
  border-color: var(--vp-c-brand-1);
  color: var(--vp-c-brand-1);
}

.install-command {
  display: grid;
  grid-template-columns: auto minmax(0, 1fr) auto;
  align-items: center;
  gap: 0.75rem;
  max-width: 940px;
  padding: 0.65rem 0.75rem 0.65rem 1rem;
  border: 1px solid var(--vp-c-divider);
  border-radius: 12px;
  background: var(--vp-code-block-bg);
  color: var(--vp-code-block-color);
}

.install-command .prompt,
.example :deep(.terminal-prompt) {
  color: var(--vp-c-brand-1);
  font-weight: 700;
}

.install-command code {
  overflow-x: auto;
  padding: 0;
  background: none;
  color: inherit;
  font-size: 0.85rem;
  white-space: nowrap;
}

.install-command button {
  min-width: 4.5rem;
  border: 1px solid var(--vp-c-divider);
  border-radius: 7px;
  background: var(--vp-c-bg-soft);
  color: var(--vp-c-text-1);
  cursor: pointer;
  font: inherit;
  font-size: 0.75rem;
  font-weight: 700;
  padding: 0.45rem 0.7rem;
}

.install-command button:hover,
.install-command button:focus-visible {
  border-color: var(--vp-c-brand-1);
  color: var(--vp-c-brand-1);
}

.install-note {
  margin: 0.75rem 0 2rem;
  color: var(--vp-c-text-2);
  font-size: 0.85rem;
}

.install-note a,
.example-actions a,
.path-grid a {
  color: var(--vp-c-brand-1);
  font-weight: 600;
  text-decoration: none;
}

.install-note a:hover,
.example-actions a:hover,
.path-grid a:hover {
  text-decoration: underline;
}

.example {
  display: grid;
  grid-template-columns: 1.1fr 0.9fr;
  overflow: hidden;
  border: 1px solid var(--vp-c-divider);
  border-radius: 16px;
  background: var(--vp-code-block-bg);
  box-shadow: var(--vp-shadow-3);
}

.code-panel + .code-panel {
  border-left: 1px solid var(--vp-c-divider);
}

.code-panel {
  display: flex;
  flex-direction: column;
  min-width: 0;
}

.code-body {
  display: flex;
  flex: 1;
  min-width: 0;
}

.panel-title {
  display: flex;
  align-items: center;
  gap: 0.6rem;
  height: 44px;
  padding: 0 1.25rem;
  border-bottom: 1px solid var(--vp-c-divider);
  color: var(--vp-c-text-2);
  font-family: var(--vp-font-family-mono);
  font-size: 0.72rem;
  font-weight: 600;
}

.status-dot {
  width: 8px;
  height: 8px;
  border-radius: 50%;
  background: var(--vp-c-brand-1);
  box-shadow: 0 0 0 4px var(--vp-c-brand-soft);
}

.code-panel :deep(pre) {
  flex: 1;
  min-width: 0;
  margin: 0;
  overflow: auto;
  padding: 1.5rem;
  background: transparent;
  color: var(--vp-code-block-color);
  font-family: var(--vp-font-family-mono);
  font-size: 0.82rem;
  line-height: 1.65;
}

.terminal-panel :deep(pre) {
  color: var(--vp-c-text-2);
}

.example-actions {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 1.25rem;
  margin-top: 1.5rem;
}

.example-actions .primary-link {
  display: inline-flex;
  gap: 0.5rem;
  align-items: center;
  border-radius: 8px;
  background: var(--vp-c-brand-3);
  color: var(--vp-button-brand-text);
  padding: 0.7rem 1rem;
}

.example-actions .primary-link:hover {
  background: var(--vp-button-brand-hover-bg);
  text-decoration: none;
}

.choose-path {
  padding-top: 7rem;
}

.path-grid {
  display: grid;
  grid-template-columns: repeat(3, 1fr);
  gap: 1rem;
}

.path-grid article {
  position: relative;
  display: flex;
  flex-direction: column;
  min-width: 0;
  padding: 1.5rem;
  border: 1px solid var(--vp-c-divider);
  border-radius: 14px;
  background: var(--vp-c-bg-soft);
}

.path-number {
  color: var(--vp-c-brand-1);
  font-family: var(--vp-font-family-mono);
  font-size: 0.75rem;
  font-weight: 700;
}

.path-grid h3 {
  margin: 0.85rem 0 0;
  color: var(--vp-c-text-1);
  font-size: 1.1rem;
}

.path-grid p {
  margin: 0.75rem 0 1rem;
  color: var(--vp-c-text-2);
  font-size: 0.9rem;
  line-height: 1.6;
}

.path-grid ul {
  display: grid;
  gap: 0.55rem;
  margin: auto 0 0;
  padding: 1rem 0 0;
  border-top: 1px solid var(--vp-c-divider);
  list-style: none;
}

.path-grid li::before {
  content: '→';
  margin-right: 0.5rem;
  color: var(--vp-c-text-3);
}

.sponsors {
  max-width: 1152px;
  margin: 0 auto;
  padding: 0 24px;
}

@media (max-width: 959px) {
  .quick-start,
  .choose-path {
    padding-top: 4.5rem;
  }

  .example,
  .path-grid {
    grid-template-columns: 1fr;
  }

  .code-panel + .code-panel {
    border-top: 1px solid var(--vp-c-divider);
    border-left: 0;
  }
}

@media (max-width: 639px) {
  .home-content {
    margin-top: 2.5rem;
  }

  .quick-start,
  .choose-path {
    padding-right: 16px;
    padding-left: 16px;
  }

  .install-command {
    grid-template-columns: auto minmax(0, 1fr);
  }

  .install-command button {
    grid-column: 1 / -1;
  }

  .code-panel :deep(pre) {
    padding: 1rem;
    font-size: 0.74rem;
  }
}
</style>

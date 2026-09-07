<script setup lang="ts">
import { nextTick, onMounted, onUnmounted, ref } from 'vue';

const platforms = [
  { id: 'all', label: 'All systems' },
  { id: 'macos', label: 'macOS' },
  { id: 'linux', label: 'Linux' },
  { id: 'windows', label: 'Windows' }
];
const selected = ref('all');
const methods = ref<HTMLElement>();
const count = ref<number>();

async function selectPlatform(platform: string) {
  selected.value = platform;
  await nextTick();
  count.value = Array.from(
    methods.value?.querySelectorAll<HTMLElement>('.install-method') ?? []
  ).filter((method) => method.offsetHeight > 0).length;
}

async function revealLinkedMethod() {
  if (!location.hash || selected.value === 'all') return;
  let id: string;
  try {
    id = decodeURIComponent(location.hash.slice(1));
  } catch {
    return;
  }
  const target = document.getElementById(id);
  const method = target?.closest<HTMLElement>('.install-method');
  if (!method || !methods.value?.contains(method) || method.offsetHeight > 0) {
    return;
  }
  await selectPlatform('all');
  target?.scrollIntoView();
}

function revealCurrentLink(event: MouseEvent) {
  if (
    event.button !== 0 ||
    event.metaKey ||
    event.ctrlKey ||
    event.shiftKey ||
    event.altKey
  ) {
    return;
  }
  const link =
    event.target instanceof Element
      ? event.target.closest<HTMLAnchorElement>('a[href]')
      : null;
  // Clicking the current anchor again does not emit a hashchange event.
  if (link?.href === location.href) void revealLinkedMethod();
}

onMounted(() => {
  window.addEventListener('click', revealCurrentLink, true);
  window.addEventListener('hashchange', revealLinkedMethod);
  window.addEventListener('popstate', revealLinkedMethod);
});
onUnmounted(() => {
  window.removeEventListener('click', revealCurrentLink, true);
  window.removeEventListener('hashchange', revealLinkedMethod);
  window.removeEventListener('popstate', revealLinkedMethod);
});
</script>

<template>
  <div ref="methods" class="installation-methods" :data-platform="selected">
    <div class="install-picker">
      <span id="install-platform-label" class="install-picker-label">
        Choose your system
      </span>
      <div
        class="install-platform-buttons"
        role="group"
        aria-labelledby="install-platform-label"
      >
        <button
          v-for="platform in platforms"
          :key="platform.id"
          type="button"
          :aria-pressed="selected === platform.id"
          @click="selectPlatform(platform.id)"
        >
          {{ platform.label }}
        </button>
      </div>
      <span class="install-sr-only" role="status">
        {{ count === undefined ? '' : `${count} installation methods shown.` }}
      </span>
    </div>
    <slot />
  </div>
</template>

<style scoped>
:global(.installation-page .installation-shortcuts) {
  display: grid;
  grid-template-columns: repeat(3, minmax(0, 1fr));
  gap: 12px;
  margin: 28px 0 32px;
}

:global(.installation-page .installation-shortcuts a) {
  display: flex;
  flex-direction: column;
  padding: 18px 16px;
  border: 1px solid var(--vp-c-divider);
  border-radius: 12px;
  background: var(--vp-c-bg-soft);
  color: var(--vp-c-text-1);
  text-decoration: none;
  transition: border-color 0.15s;
}

:global(.installation-page .installation-shortcuts a:hover) {
  border-color: var(--vp-c-brand-1);
}

:global(.installation-page .installation-shortcuts a:focus-visible) {
  outline: 2px solid var(--vp-c-brand-1);
  outline-offset: 3px;
}

:global(.installation-page .installation-shortcuts span) {
  margin-bottom: 12px;
  color: var(--vp-c-text-2);
  font-family: var(--vp-font-family-mono);
  font-size: 12px;
}

:global(.installation-page .installation-shortcuts strong) {
  font-size: 14px;
  font-weight: 600;
  line-height: 22px;
}

:global(.installation-page .installation-shortcuts small) {
  margin-top: 4px;
  color: var(--vp-c-text-2);
  font-size: 12px;
  line-height: 20px;
}

.install-picker {
  margin: 28px 0 32px;
}

.install-picker-label {
  display: block;
  margin-bottom: 12px;
  color: var(--vp-c-text-2);
  font-size: 13px;
  font-weight: 600;
}

.install-platform-buttons {
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
  width: fit-content;
  padding: 5px;
  border: 1px solid var(--vp-c-divider);
  border-radius: 12px;
  background: var(--vp-c-bg-soft);
}

.install-platform-buttons button {
  min-height: 40px;
  padding: 8px 18px;
  border-radius: 8px;
  color: var(--vp-c-text-2);
  font-size: 14px;
  font-weight: 600;
  transition:
    background-color 0.15s,
    color 0.15s;
}

.install-platform-buttons button:hover {
  color: var(--vp-c-text-1);
  background: var(--vp-c-default-soft);
}

.install-platform-buttons button[aria-pressed='true'] {
  color: var(--vp-c-text-1);
  background: var(--vp-c-bg);
  box-shadow: 0 1px 4px rgb(0 0 0 / 10%);
}

.install-platform-buttons button:focus-visible {
  outline: 2px solid var(--vp-c-brand-1);
  outline-offset: 2px;
}

.install-sr-only {
  position: absolute;
  width: 1px;
  height: 1px;
  overflow: hidden;
  clip-path: inset(50%);
  white-space: nowrap;
}

.installation-methods :deep(.install-method) {
  position: relative;
  margin: 16px 0;
  padding: 24px;
  border: 1px solid var(--vp-c-divider);
  border-radius: 12px;
  background: var(--vp-c-bg);
}

.installation-methods :deep(.install-method h3) {
  margin: 0 0 4px;
  padding: 0;
  font-size: 19px;
  line-height: 28px;
}

.installation-methods :deep(.install-method h3 a:not(.header-anchor)) {
  color: var(--vp-c-text-1);
  text-decoration: none;
}

.installation-methods :deep(.install-method h3 a:hover) {
  color: var(--vp-c-brand-1);
}

.installation-methods :deep(.install-meta) {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  justify-content: space-between;
  gap: 8px 16px;
  margin: 8px 0 20px;
}

.installation-methods :deep(.install-method .install-platforms) {
  margin: 0;
  color: var(--vp-c-text-2);
  font-size: 12px;
  line-height: 20px;
  font-weight: 500;
}

.installation-methods :deep(.install-links p) {
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
  margin: 0;
}

.installation-methods :deep(.install-links a) {
  padding: 3px 9px;
  border: 1px solid transparent;
  border-radius: 6px;
  background: color-mix(in srgb, var(--vp-c-brand-1) 12%, transparent);
  color: color-mix(in srgb, var(--vp-c-brand-1) 45%, var(--vp-c-text-1));
  font-size: 12px;
  font-weight: 500;
  line-height: 20px;
  text-decoration: none;
}

.installation-methods :deep(.install-links a:hover) {
  border-color: var(--vp-c-brand-1);
}

.installation-methods :deep(.install-links a:focus-visible) {
  outline: 2px solid var(--vp-c-brand-1);
  outline-offset: 2px;
}

.installation-methods :deep(.install-method p) {
  font-size: 14px;
  line-height: 24px;
}

.installation-methods :deep(.install-method > :last-child) {
  margin-bottom: 0;
}

.installation-methods :deep(.install-method div[class*='language-']) {
  margin: 16px 0;
  border: 1px solid var(--vp-c-divider);
  border-radius: 8px;
}

.installation-methods :deep(.install-method div[class*='language-'] pre) {
  padding: 16px 0;
}

.installation-methods :deep(.install-method div[class*='language-'] code) {
  padding: 0 18px;
  font-size: 13px;
}

.installation-methods :deep(.install-method .vp-code-group) {
  margin-top: 16px;
}

.installation-methods :deep(.install-method .vp-code-group .tabs) {
  margin: 0;
  border-radius: 8px 8px 0 0;
}

.installation-methods
  :deep(.install-method .vp-code-group div[class*='language-']) {
  margin: 0;
  border-radius: 0 0 8px 8px;
}

.installation-methods :deep(.install-hosting) {
  margin: 24px 0 40px;
  color: var(--vp-c-text-2);
  font-size: 13px;
  line-height: 22px;
}

.installation-methods[data-platform='macos']
  :deep(.install-method:not([data-platforms~='macos'])),
.installation-methods[data-platform='linux']
  :deep(.install-method:not([data-platforms~='linux'])),
.installation-methods[data-platform='windows']
  :deep(.install-method:not([data-platforms~='windows'])) {
  display: none;
}

@media (max-width: 639px) {
  :global(.installation-page .installation-shortcuts) {
    grid-template-columns: 1fr;
    gap: 8px;
  }

  :global(.installation-page .installation-shortcuts a) {
    display: grid;
    grid-template-columns: 24px 1fr;
    column-gap: 12px;
    padding: 12px 16px;
  }

  :global(.installation-page .installation-shortcuts span) {
    grid-row: span 2;
    align-self: center;
    margin: 0;
  }

  :global(.installation-page .installation-shortcuts small) {
    margin-top: 0;
  }

  .install-platform-buttons {
    display: grid;
    grid-template-columns: repeat(2, minmax(0, 1fr));
    width: 100%;
  }

  .installation-methods :deep(.install-method) {
    padding: 20px 16px;
  }
}

@media print {
  .install-picker {
    display: none;
  }

  .installation-methods :deep(.install-method) {
    display: block !important;
    break-inside: avoid;
  }
}
</style>

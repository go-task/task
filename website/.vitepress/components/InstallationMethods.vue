<script setup lang="ts">
import { nextTick, onMounted, onUnmounted, ref } from 'vue';

const platforms = [
  { id: 'all', label: 'All' },
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
          <svg
            class="install-platform-icon"
            viewBox="0 0 24 24"
            fill="none"
            stroke="currentColor"
            stroke-width="1.7"
            stroke-linecap="round"
            stroke-linejoin="round"
            aria-hidden="true"
          >
            <template v-if="platform.id === 'all'">
              <rect x="3" y="4" width="18" height="13" rx="2" />
              <path d="M8 21h8m-4-4v4" />
            </template>
            <path
              v-else-if="platform.id === 'macos'"
              d="M9 9V6a3 3 0 1 0-3 3h12a3 3 0 1 0-3-3v12a3 3 0 1 0 3-3H6a3 3 0 1 0 3 3V9"
            />
            <template v-else-if="platform.id === 'linux'">
              <rect x="3" y="4" width="18" height="16" rx="2" />
              <path d="m7 9 3 3-3 3m6 0h4" />
            </template>
            <template v-else>
              <rect x="3" y="3" width="18" height="18" rx="1" />
              <path d="M12 3v18M3 12h18" />
            </template>
          </svg>
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
  display: flex;
  flex-wrap: wrap;
  gap: 4px 20px;
  margin: 12px 0 24px;
}

:global(.installation-page .installation-shortcuts a) {
  padding: 4px 0;
  color: var(--vp-c-text-2);
  font-size: 13px;
  line-height: 20px;
  text-decoration: none;
}

:global(.installation-page .installation-shortcuts a:hover) {
  color: var(--vp-c-brand-1);
  text-decoration: underline;
  text-underline-offset: 4px;
}

:global(.installation-page .installation-shortcuts a:focus-visible) {
  outline: 2px solid var(--vp-c-brand-1);
  outline-offset: 3px;
}

.install-picker {
  margin: 24px 0 0;
}

.install-picker-label {
  display: block;
  margin-bottom: 8px;
  color: var(--vp-c-text-2);
  font-size: 13px;
  font-weight: 600;
}

.install-platform-buttons {
  display: flex;
  gap: 3px;
  width: fit-content;
  max-width: 100%;
  padding: 4px;
  border: 1px solid var(--vp-c-divider);
  border-radius: 12px;
  background: var(--vp-c-bg-soft);
}

.install-platform-buttons button {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 6px;
  min-height: 40px;
  padding: 8px 16px;
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
  color: color-mix(in srgb, var(--vp-c-brand-1) 40%, var(--vp-c-text-1));
  background: color-mix(in srgb, var(--vp-c-brand-1) 18%, var(--vp-c-bg));
  box-shadow: inset 0 0 0 1px
    color-mix(in srgb, var(--vp-c-brand-1) 40%, transparent);
}

.install-platform-icon {
  width: 16px;
  height: 16px;
  flex-shrink: 0;
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
  padding: 20px;
  border: 1px solid var(--vp-c-divider);
  border-radius: 12px;
  background: var(--vp-c-bg);
}

.installation-methods :deep(.install-method h3) {
  margin: 0 0 4px;
  padding: 0;
  font-size: 18px;
  line-height: 26px;
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
  margin: 4px 0 12px;
}

.installation-methods :deep(.install-method .install-platforms) {
  margin: 0;
  color: var(--vp-c-text-2);
  font-size: 12px;
  line-height: 20px;
  font-weight: 500;
}

.installation-methods :deep(.install-method .install-links p) {
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
  margin: 12px 0;
  font-size: 14px;
  line-height: 22px;
}

.installation-methods :deep(.installation-shortcuts + h2) {
  margin-top: 24px;
  padding-top: 0;
  border-top: 0;
}

.installation-methods :deep(.install-method > :last-child) {
  margin-bottom: 0;
}

.installation-methods :deep(.install-method div[class*='language-']) {
  margin: 12px 0;
  border: 1px solid var(--vp-c-divider);
  border-radius: 8px;
}

.installation-methods :deep(.install-method div[class*='language-'] pre) {
  padding: 14px 0;
  overflow-x: auto;
  overflow-y: hidden;
}

.installation-methods :deep(.install-method div[class*='language-'] code) {
  padding: 0 18px;
  font-size: 13px;
  line-height: 22px;
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
  .install-platform-buttons {
    width: 100%;
  }

  .install-platform-buttons button {
    flex: 1;
    padding: 8px 10px;
    font-size: 13px;
  }

  .installation-methods :deep(.install-method) {
    padding: 16px;
  }
}

@media (max-width: 379px) {
  .install-platform-icon {
    display: none;
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

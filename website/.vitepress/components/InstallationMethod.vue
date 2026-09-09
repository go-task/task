<script setup lang="ts">
defineProps<{
  platforms: string;
  platformLabel: string;
}>();
</script>

<template>
  <div class="install-method" :data-platforms="platforms">
    <slot name="heading" />
    <div class="install-meta">
      <p class="install-platforms">{{ platformLabel }}</p>
      <div v-if="$slots.links" class="install-links">
        <slot name="links" />
      </div>
    </div>
    <slot />
  </div>
</template>

<style scoped>
.install-method {
  position: relative;
  margin: 16px 0;
  padding: 20px;
  border: 1px solid var(--vp-c-divider);
  border-radius: 12px;
  background: var(--vp-c-bg);
}

.install-method :deep(h3) {
  position: relative;
  margin: 0 0 4px;
  padding: 0 0 0 30px;
  font-size: 18px;
  line-height: 26px;
}

/* Decorative CSS masks keep logos out of heading names and the page outline. */
.install-method :deep(h3::before) {
  position: absolute;
  top: 3px;
  left: 0;
  width: 20px;
  height: 20px;
  background-color: var(--install-icon-color, var(--vp-c-text-2));
  content: '';
  -webkit-mask: var(--install-icon, url('../theme/icons/install/package.svg'))
    center / contain no-repeat;
  mask: var(--install-icon, url('../theme/icons/install/package.svg')) center /
    contain no-repeat;
}

.install-method :deep(#homebrew) {
  --install-icon: url('../theme/icons/install/homebrew.svg');
  --install-icon-color: #fbb040;
}

.install-method :deep(#apk) {
  --install-icon: url('../theme/icons/install/alpinelinux.svg');
  --install-icon-color: #0d597f;
}

.install-method :deep(#snap) {
  --install-icon: url('../theme/icons/install/snapcraft.svg');
  --install-icon-color: #e95420;
}

.install-method :deep(#npm) {
  --install-icon: url('../theme/icons/install/npm.svg');
  --install-icon-color: #cb3837;
}

.install-method :deep(#mise) {
  --install-icon: url('../theme/icons/install/mise.svg');
  --install-icon-color: var(--vp-c-text-1);
}

.install-method :deep(#pip) {
  --install-icon: url('../theme/icons/install/python.svg');
  --install-icon-color: #3776ab;
}

.install-method :deep(#chocolatey) {
  --install-icon: url('../theme/icons/install/chocolatey.svg');
  --install-icon-color: #80b5e3;
}

.install-method :deep(#arch) {
  --install-icon: url('../theme/icons/install/archlinux.svg');
  --install-icon-color: #1793d1;
}

.install-method :deep(#fedora-community) {
  --install-icon: url('../theme/icons/install/fedora.svg');
  --install-icon-color: #51a2da;
}

.install-method :deep(#freebsd) {
  --install-icon: url('../theme/icons/install/freebsd.svg');
  --install-icon-color: #ab2b28;
}

.install-method :deep(#nix) {
  --install-icon: url('../theme/icons/install/nixos.svg');
  --install-icon-color: #5277c3;
}

.install-method :deep(h3 a:not(.header-anchor)) {
  color: var(--vp-c-text-1);
  text-decoration: none;
}

.install-method :deep(h3 a:hover) {
  color: var(--vp-c-brand-1);
}

.install-method :deep(.install-meta) {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  justify-content: space-between;
  gap: 8px 16px;
  margin: 4px 0 12px;
}

.install-method :deep(.install-platforms) {
  margin: 0;
  color: var(--vp-c-text-2);
  font-size: 12px;
  line-height: 20px;
  font-weight: 500;
}

.install-method :deep(.install-links p) {
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
  margin: 0;
}

.install-method :deep(.install-links a) {
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

.install-method :deep(.install-links a:hover) {
  border-color: var(--vp-c-brand-1);
}

.install-method :deep(.install-links a:focus-visible) {
  outline: 2px solid var(--vp-c-brand-1);
  outline-offset: 2px;
}

.install-method :deep(p) {
  margin: 12px 0;
  font-size: 14px;
  line-height: 22px;
}

.install-method > :deep(:last-child) {
  margin-bottom: 0;
}

.install-method :deep(div[class*='language-']) {
  margin: 12px 0;
  border: 1px solid var(--vp-c-divider);
  border-radius: 8px;
}

/* Compact commands leave no room for the overlaid language label. */
.install-method :deep(div[class*='language-'] > span.lang) {
  display: none;
}

.install-method :deep(div[class*='language-'] pre) {
  padding: 14px 0;
}

.install-method :deep(div[class*='language-'] code) {
  padding: 0 18px;
  font-size: 13px;
  line-height: 22px;
}

/* Keep the 40px copy control inside the compact, 50px single-line block. */
.install-method :deep(div[class*='language-'] > button.copy) {
  top: 5px;
  right: 6px;
}

.install-method :deep(.vp-code-group) {
  margin-top: 16px;
}

.install-method :deep(.vp-code-group .tabs) {
  margin: 0;
  border-radius: 8px 8px 0 0;
}

.install-method :deep(.vp-code-group div[class*='language-']) {
  margin: 0;
  border-radius: 0 0 8px 8px;
}

@media (max-width: 639px) {
  .install-method {
    padding: 16px;
  }
}

@media print {
  .install-method {
    display: block !important;
    break-inside: avoid;
  }
}
</style>

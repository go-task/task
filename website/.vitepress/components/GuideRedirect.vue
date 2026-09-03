<script setup lang="ts">
import { onMounted, onUnmounted } from 'vue';
import { guideAnchors } from '../guideAnchors';

// The guide used to be a single page, so links to it in issues, blog posts and
// Stack Overflow answers point at anchors that now live on other pages. Netlify
// never receives the fragment, so this has to be resolved in the browser.
//
// location.replace rather than the VitePress router: the router leaves the new
// fragment unscrolled, and it pushes a history entry, so going back would land
// on the guide with the old hash still set and redirect again.
function resolve() {
  const hash = window.location.hash.slice(1);
  if (!hash) return;

  const target = guideAnchors[decodeURIComponent(hash).toLowerCase()];
  if (target) window.location.replace(target);
}

onMounted(() => {
  resolve();
  window.addEventListener('hashchange', resolve);
});

onUnmounted(() => window.removeEventListener('hashchange', resolve));
</script>

<template>
  <span hidden />
</template>

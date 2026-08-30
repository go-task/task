<script setup lang="ts">
import { VPButton } from 'vitepress/theme';
import { data as stars } from '../githubStars.data';

// VitePress renders the `home-hero-actions-after` slot outside its own
// `.actions` row, so the hero CTAs live here rather than in the front matter:
// it is the only way to line the star count up with the other two buttons.
</script>

<template>
  <div class="hero-actions">
    <div class="hero-action">
      <VPButton theme="brand" text="Get Started" href="/docs/getting-started" />
    </div>
    <div class="hero-action">
      <VPButton
        theme="alt"
        text="Browse the Reference"
        href="/docs/reference/schema"
      />
    </div>
    <div class="hero-action">
      <VPButton
        theme="alt"
        :text="`★ ${stars.label}`"
        :aria-label="`Task on GitHub, ${stars.count} stars`"
        href="https://github.com/go-task/task"
        data-umami-event="home-hero-github"
      />
    </div>
  </div>
</template>

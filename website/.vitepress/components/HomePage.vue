<script setup lang="ts">
import { VPHomeSponsors } from 'vitepress/theme';
import { sponsors } from '../sponsors';
</script>

<template>
  <div class="content">
    <div class="content-container">
      <main class="main">
        <VPHomeSponsors
          v-if="sponsors"
          message="Task is free and open source, made possible by wonderful sponsors."
          :data="sponsors"
        />
      </main>
    </div>
  </div>
</template>

<style scoped></style>

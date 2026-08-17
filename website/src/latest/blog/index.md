---
title: Blog
description: Latest news and updates from the Task team
editLink: false
---

<script setup>
import { data as posts } from '../../../.vitepress/blog.data';
</script>

<BlogPost
  v-for="post in posts"
  :key="post.url"
  :title="post.title"
  :url="post.url"
  :date="post.date.string"
  :author="post.author"
  :description="post.excerpt"
  :tags="post.tags"
/>

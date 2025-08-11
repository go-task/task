---
layout: page
---

<script setup>
import {
  VPTeamPage,
  VPTeamPageTitle,
  VPTeamMembers
} from 'vitepress/theme'

const members = [
  {
    avatar: 'https://www.github.com/andreynering.png',
    name: 'Andrey Nering',
    icon: '/img/flag-brazil.svg',
    title: 'Creator & Maintainer',
    sponsor: 'https://github.com/sponsors/andreynering',
    links: [
      { icon: 'github', link: 'https://github.com/andreynering' },
      { icon: 'discord', link: 'https://discord.com/users/310141681926275082' },
      { icon: 'x', link: 'https://x.com/andreynering' },
      { icon: 'bluesky', link: 'https://bsky.app/profile/andreynering.bsky.social' },
      { icon: 'mastodon', link: 'https://mastodon.social/@andreynering' }
    ]
  },
  {
    avatar: 'https://www.github.com/pd93.png',
    name: 'Pete Davison',
    icon: '/img/flag-wales.svg',
    title: 'Maintainer',
    sponsor: 'https://github.com/sponsors/pd93',
    links: [
      { icon: 'github', link: 'https://github.com/pd93' },
      { icon: 'bluesky', link: 'https://bsky.app/profile/pd93.uk' }
    ]
  },
  {
    avatar: 'https://www.github.com/vmaerten.png',
    name: 'Valentin Maerten',
    icon: '/img/flag-france.svg',
    title: 'Maintainer',
    sponsor: 'https://github.com/sponsors/vmaerten',
    links: [
      { icon: 'github', link: 'https://github.com/vmaerten' },
      { icon: 'x', link: 'https://x.com/v_maerten' },
      { icon: 'bluesky', link: 'https://bsky.app/profile/vmaerten.bsky.social' }
    ]
  }

]
</script>

<VPTeamPage>
  <VPTeamPageTitle>
    <template #title>
      Our Team
    </template>
    <template #lead>
      The development of Task is guided by an international
      team, some of whom have chosen to be featured below.
    </template>
  </VPTeamPageTitle>
  <VPTeamMembers :members />
</VPTeamPage>

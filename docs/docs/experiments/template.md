---
# This is a template for an experiments documentation
# Copy this page and fill in the details as necessary
title: '--- Template ---'
sidebar_position: -1 # Always push to the top
draft: true # Hide in production
---

# \{Name of Experiment\} (#\{Issue\})

:::caution

All experimental features are subject to breaking changes and/or removal _at any
time_. We strongly recommend that you do not use these features in a production
environment. They are intended for testing and feedback only.

:::

:::warning

This experiment breaks the following functionality:

- \{list any existing functionality that will be broken by this experiment\}
- \{if there are no breaking changes, remove this admonition\}

:::

:::info

To enable this experiment, set the environment variable: `TASK_X_{feature}=1`.
Check out [our guide to enabling experiments ][enabling-experiments] for more
information.

:::

\{Short description of the feature\}

\{Short explanation of how users should migrate to the new behavior\}

<!-- prettier-ignore-start -->
[enabling-experiments]: /experiments/#enabling-experiments
<!-- prettier-ignore-end -->

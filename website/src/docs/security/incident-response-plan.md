---
title: Incident Response Plan
outline: deep
---

# Incident Response Plan

This document outlines our incident response plan in the event that a
vulnerability is reported to the Task project. This serves as a high-level,
public guide and is published as part of our commitment to transparency.

Below are the security principles that we aim to adhere to as a project:

- **Transparency**: All incidents and fixes are documented here for the
  community.
- **Stewardship**: Take responsibility for protecting users and the project.
- **Protection**: Act to minimize harm and provide guidance.

## Scope

This plan applies to the core Task repository and all _official_ Task projects.
For example, the Visual Studio Code extension and officially supported
installation methods. In the event that a vulnerability is reported with a
community-managed installation method, we will work with the community and make
a "best-effort" attempt to help resolve the issue.

## Steps

### 🔍 1. Detect

- All security issues should be **privately reported** as described in our
  [security documentation][security-docs].
- Maintainers should also regularly monitor and respond to:
  - Pull requests from dependency scanners such as Dependabot.
  - GitHub notifications and vulnerability alerts.
  - Messages in community channels such as Discord.

### 🩺 2. Triage

- Upon first receipt of a security issue, one of our team will immediately
  notify the other maintainers via a secure and private channel. This ensures
  that all maintainers are able to contribute to the issue where possible.
- A maintainer should respond to the reporter in a timely manner in order to
  acknowledge receipt of the issue.
- The issue must then be triaged into one of the following categories:
  - ‼️**Critical**: Has a serious and immediate impact on users or affects
    critical infrastructure related to the project.
  - ❗**High**: Has the potential to seriously impact users of a distributed
    asset.
  - 🟰**Medium**: Has the potential to impact users, but is obscure or low-risk.
  - ➖**Low**: No direct or immediate impact to users, but requires attention.
- Open a draft
  [GitHub Security Advisory (GHSA)](https://github.com/go-task/task/security/advisories)
  in the Task repository.
  - Optionally create a CVE. This can be skipped for low/medium impact issues at
    the discretion of the maintainers.

### 🩹 3. Mitigate

- Act calmly and communicate decisions.
- Stop the bleed.
  - Before attempting to fix the issue, perform any actions that stop the
    problem from becoming worse. For example:
    - Rotate any affected secrets.
    - Rebuild any affected services (website, etc.).
  - It may be difficult to do some of this in cases where packages are
    maintained by the community if we are not yet ready to disclose the
    vulnerability publicly. This should be decided on a case-by-case basis.
- Address the root cause.
  - Plan and document a fix.
  - Patch the issue.
  - Test the fix.
  - Release new versions.

### 📢 4. Disclose

- Publish the GitHub Security Advisory (GHSE). Make sure to include:
  - The affected version(s)/services.
  - The impact of the issue.
  - The root cause.
  - The steps taken to resolve.
- Optionally, create a blog post and/or share the information via our socials
  and public communication channels.

### 🧠 5. Learn

- Document the disclosure in a permanent location.
- Make and document any changes that can be made to prevent similar issues from
  arising in the future.

[security-docs]: ../security

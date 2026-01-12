# GitHub Repository Setup Guide

This document provides instructions for configuring the GitHub repository settings to maintain a high-quality open-source project.

## Repository About Section

Configure via: **Settings** → **General** → **About** (top right of repository page)

### Description
```
kubectl-inspired CLI for managing Dynatrace platform resources from your terminal
```

### Website
```
https://github.com/dynatrace-oss/dtctl
```

### Topics (Tags)
Add the following topics to improve discoverability:

```
dynatrace
cli
kubectl
kubernetes
devops
observability
golang
command-line
workflow-automation
dql
dashboards
monitoring
platform-engineering
developer-tools
```

### Features Checkboxes
- ✅ Releases
- ✅ Packages (if publishing to GitHub Packages)
- ❌ Environments (not needed for CLI tool)
- ❌ Deployments (not needed for CLI tool)

## Repository Settings

### General Settings

Navigate to: **Settings** → **General**

#### Features
- ✅ **Wikis**: Disabled (use docs/ directory instead)
- ✅ **Issues**: Enabled
- ✅ **Sponsorships**: Disabled (or enabled if you want to accept sponsorships)
- ✅ **Preserve this repository**: Consider enabling for important projects
- ✅ **Discussions**: Enabled (for community Q&A)
- ✅ **Projects**: Enabled (optional, for roadmap tracking)

#### Pull Requests
- ✅ **Allow squash merging**: Enabled (recommended)
  - Default to: "Default to pull request title"
- ✅ **Allow merge commits**: Enabled
- ✅ **Allow rebase merging**: Enabled
- ✅ **Always suggest updating pull request branches**: Enabled
- ✅ **Allow auto-merge**: Enabled
- ✅ **Automatically delete head branches**: Enabled (keeps repo clean)

#### Archives
- ❌ **Include Git LFS objects in archives**: Disabled (not using LFS)

### Branch Protection Rules

Navigate to: **Settings** → **Branches** → **Add branch protection rule**

#### For `main` branch:

**Branch name pattern**: `main`

**Protect matching branches**:
- ✅ **Require a pull request before merging**
  - ✅ Require approvals: 1 (or 2 for stricter review)
  - ✅ Dismiss stale pull request approvals when new commits are pushed
  - ❌ Require review from Code Owners (optional, if you have CODEOWNERS file)
- ✅ **Require status checks to pass before merging**
  - ✅ Require branches to be up to date before merging
  - Required status checks:
    - `build`
    - `test`
    - `lint`
    - `security`
- ✅ **Require conversation resolution before merging**
- ✅ **Require signed commits** (recommended for security)
- ❌ **Require linear history** (optional, depends on merge strategy preference)
- ✅ **Do not allow bypassing the above settings** (maintainers should follow rules too)
- ✅ **Restrict who can push to matching branches**
  - Add: Maintainers/Admins only
- ✅ **Allow force pushes**: Disabled
- ✅ **Allow deletions**: Disabled

### Code Security and Analysis

Navigate to: **Settings** → **Security & analysis**

#### Security Features
- ✅ **Dependency graph**: Enabled (auto-enabled for public repos)
- ✅ **Dependabot alerts**: Enabled
- ✅ **Dependabot security updates**: Enabled (auto-creates PRs for vulnerable dependencies)
- ✅ **Dependabot version updates**: Optional (configure via `.github/dependabot.yml` if desired)
- ✅ **Code scanning**: Enabled
  - Set up CodeQL analysis for Go
- ✅ **Secret scanning**: Enabled (auto-enabled for public repos)
- ✅ **Secret scanning push protection**: Enabled (prevents committing secrets)

#### Private vulnerability reporting
- ✅ **Enable private vulnerability reporting**: Enabled
  - This allows security researchers to privately disclose vulnerabilities

### Actions Permissions

Navigate to: **Settings** → **Actions** → **General**

#### Actions permissions
- ✅ **Allow all actions and reusable workflows** (or restrict as needed)

#### Workflow permissions
- 🔘 **Read and write permissions** (default)
- ✅ **Allow GitHub Actions to create and approve pull requests** (for Dependabot)

### Pages (Optional)

If you want to host documentation:

Navigate to: **Settings** → **Pages**

- **Source**: Deploy from a branch
- **Branch**: `gh-pages` (or `main` with `/docs` folder)
- **Custom domain**: Optional

### Notifications

Navigate to: **Settings** → **Notifications**

Configure email notifications for:
- ✅ Issues
- ✅ Pull requests
- ✅ Releases
- ✅ Discussions
- ✅ Security alerts

## Additional Files to Consider

### .github/CODEOWNERS (Optional)

Create this file to automatically request reviews from code owners:

```
# Default owners for everything in the repo
* @maintainer-username

# Specific areas
/pkg/client/ @network-expert-username
/pkg/resources/ @api-expert-username
/docs/ @documentation-lead-username
```

### .github/dependabot.yml (Optional)

Automate dependency updates:

```yaml
version: 2
updates:
  - package-ecosystem: "gomod"
    directory: "/"
    schedule:
      interval: "weekly"
    open-pull-requests-limit: 10
    labels:
      - "dependencies"
      - "go"
    reviewers:
      - "maintainer-username"

  - package-ecosystem: "github-actions"
    directory: "/"
    schedule:
      interval: "weekly"
    labels:
      - "dependencies"
      - "github-actions"
```

## Repository Insights

### Community Standards

Navigate to: **Insights** → **Community Standards**

Ensure all items are green:
- ✅ Description
- ✅ README
- ✅ Code of conduct
- ✅ Contributing guidelines
- ✅ License
- ✅ Security policy
- ✅ Issue templates
- ✅ Pull request template

### Pulse and Contributors

Monitor these regularly:
- **Pulse**: Weekly activity summary
- **Contributors**: Track contributor statistics
- **Traffic**: Monitor clones, views, and referrers
- **Forks**: See fork network

## GitHub Discussions Setup

Navigate to: **Discussions** tab → **Start a discussion**

### Suggested Categories

1. **Announcements** 📢
   - Description: Official announcements and releases
   - Format: Announcement

2. **General** 💬
   - Description: General discussions about dtctl
   - Format: Open-ended discussion

3. **Ideas** 💡
   - Description: Share ideas for new features
   - Format: Open-ended discussion

4. **Q&A** ❓
   - Description: Ask the community for help
   - Format: Question / Answer

5. **Show and tell** 🎨
   - Description: Share your dtctl workflows and use cases
   - Format: Open-ended discussion

## Releases Configuration

### Release Drafts

When creating releases (automated via GoReleaser):
- ✅ Use semantic versioning (v1.2.3)
- ✅ Include changelog (auto-generated)
- ✅ Attach binaries (tar.gz, zip)
- ✅ Mark pre-releases appropriately
- ✅ Set as "latest release" for stable versions

### Release Notes Template

Consider adding `.github/release.yml`:

```yaml
changelog:
  exclude:
    labels:
      - ignore-for-release
      - dependencies
  categories:
    - title: Breaking Changes 🚨
      labels:
        - breaking-change
    - title: New Features 🎉
      labels:
        - enhancement
        - feature
    - title: Bug Fixes 🐛
      labels:
        - bug
    - title: Documentation 📚
      labels:
        - documentation
    - title: Other Changes
      labels:
        - "*"
```

## Social Preview Image (Optional)

Create a social preview image (1280x640px) to display when sharing the repository:

Navigate to: **Settings** → **General** → **Social preview**

Suggested content:
- Project logo
- Name: "dtctl"
- Tagline: "kubectl for Dynatrace"
- Visual: Terminal with sample commands

## Labels Configuration

### Recommended Labels

Navigate to: **Issues** → **Labels**

#### Type Labels
- `bug` 🐛 (red)
- `enhancement` ✨ (light blue)
- `documentation` 📚 (blue)
- `question` ❓ (purple)

#### Priority Labels
- `priority: critical` (dark red)
- `priority: high` (orange)
- `priority: medium` (yellow)
- `priority: low` (light gray)

#### Status Labels
- `status: needs-triage` (gray)
- `status: in-progress` (yellow)
- `status: blocked` (red)
- `status: ready-for-review` (green)

#### Effort Labels
- `good first issue` (green)
- `help wanted` (green)
- `size: small` (light green)
- `size: medium` (yellow)
- `size: large` (orange)

#### Component Labels
- `area: cli` (blue)
- `area: api` (blue)
- `area: docs` (blue)
- `area: tests` (blue)

## Checklist for Repository Quality

Use this checklist to ensure your repository meets high standards:

### Essential (Must Have)
- ✅ Clear README with badges, installation, and usage
- ✅ LICENSE file (Apache 2.0)
- ✅ CONTRIBUTING.md with guidelines
- ✅ CODE_OF_CONDUCT.md
- ✅ SECURITY.md with vulnerability reporting
- ✅ Issue templates (bug report, feature request)
- ✅ Pull request template
- ✅ CI/CD workflows (build, test, lint, security)
- ✅ Release workflow
- ✅ .gitignore properly configured
- ✅ Branch protection on main
- ✅ Dependabot security updates enabled

### Recommended (Should Have)
- ✅ GitHub Discussions enabled
- ✅ Repository topics/tags configured
- ✅ About section filled in
- ✅ CITATION.cff for academic use
- ✅ Comprehensive documentation in docs/
- ✅ Auto-delete merged branches enabled
- ✅ Signed commits required
- ✅ Code scanning enabled

### Nice to Have (Optional)
- ⬜ CODEOWNERS file
- ⬜ Dependabot version updates
- ⬜ GitHub Pages for documentation
- ⬜ Social preview image
- ⬜ Custom issue labels
- ⬜ Project boards for roadmap
- ⬜ Wiki (if extensive docs needed)
- ⬜ Sponsorship configuration
- ⬜ Multiple language support in README

## Monitoring and Maintenance

### Regular Tasks
- **Weekly**: Review Dependabot PRs
- **Weekly**: Triage new issues
- **Monthly**: Review and update documentation
- **Monthly**: Check CI/CD workflow efficiency
- **Quarterly**: Review security alerts
- **Quarterly**: Update dependencies manually if needed
- **Yearly**: Review and update policies (CoC, Security, Contributing)

### Metrics to Track
- Stars and watchers
- Fork count
- Open/closed issues ratio
- PR merge time
- Test coverage
- Security vulnerabilities
- Community engagement (discussions, comments)

---

**Last Updated**: 2026-01-12

**Maintainer**: Update this guide when repository settings change

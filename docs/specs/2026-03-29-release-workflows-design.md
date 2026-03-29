# pulumi-quant CI/CD Workflows

## Overview

Self-contained GitHub Actions workflows for building, testing, and publishing the pulumi-quant provider. Replaces the 16 auto-generated boilerplate workflows that depend on Pulumi's internal tooling.

## Workflows

### 1. `ci.yml` — PR/push validation

**Trigger:** Pull requests and pushes to `main`.

**Jobs:**

- **build-and-test**: Build tfgen + resource provider binaries, run `go test ./provider/`

### 2. `release.yml` — Full release pipeline

**Trigger:** Push tag matching `v*.*.*`.

**Permissions:** `contents: write`, `id-token: write`

**Jobs:**

#### a. test
- Build provider, run `go test ./provider/`
- Gate for all subsequent jobs

#### b. build-provider (matrix: 6 platforms)
- Cross-compile `pulumi-resource-quant` for:
  - linux-amd64, linux-arm64
  - darwin-amd64, darwin-arm64
  - windows-amd64, windows-arm64
- Package as `pulumi-resource-quant-v{VERSION}-{os}-{arch}.tar.gz`
- Upload as build artifacts

#### c. publish-github-release (needs: test, build-provider)
- Create GitHub release from tag
- Attach all 6 platform tarballs
- Auto-generate release notes from commits

#### d. publish-npm (needs: test)
- Generate Node.js SDK via `pulumi-tfgen-quant nodejs`
- `yarn install && yarn build`
- `npm publish --access public` (OIDC trusted publishing, no token)
- Requires: Node 22+, npm 11.5.1+, `id-token: write`
- `registry-url: https://registry.npmjs.org` in setup-node
- Do NOT set `NODE_AUTH_TOKEN` (OIDC only activates when absent)

#### e. Future: publish-pypi, publish-nuget
- Placeholder jobs documented but not implemented
- Can be added when needed using same OIDC pattern

### 3. `upgrade.yml` — Upstream provider update

**Trigger:** `repository_dispatch` event type `upstream-release` from terraform-provider-quant, or manual `workflow_dispatch` with version input.

**Steps:**

1. Checkout pulumi-quant
2. `go get github.com/quantcdn/terraform-provider-quant/v5@{version}`
3. `go mod tidy`
4. Build tfgen, regenerate schema + all SDKs
5. Create branch `upgrade/terraform-provider-quant-{version}`
6. Commit and open PR

### 4. Addition to terraform-provider-quant `release.yml`

Add a job after the existing GoReleaser release that sends a `repository_dispatch` to `quantcdn/pulumi-quant`:

```yaml
notify-pulumi:
  needs: goreleaser
  runs-on: ubuntu-latest
  steps:
    - uses: peter-evans/repository-dispatch@v3
      with:
        token: ${{ secrets.PULUMI_QUANT_DISPATCH_TOKEN }}
        repository: quantcdn/pulumi-quant
        event-type: upstream-release
        client-payload: '{"version": "${{ github.ref_name }}"}'
```

Requires a GitHub PAT with `repo` scope stored as `PULUMI_QUANT_DISPATCH_TOKEN` in terraform-provider-quant secrets.

## Version Strategy

- Version extracted from git tag: `v0.1.0` -> `0.1.0`
- Injected via Go ldflags at build time
- SDK `package.json` version set via `sed` before publish
- `pluginDownloadURL: github://api.github.com/quantcdn/pulumi-quant` tells Pulumi to fetch binaries from GitHub releases

## Binary Naming Convention

Pulumi expects: `pulumi-resource-quant-v{VERSION}-{os}-{arch}.tar.gz`

The `github://api.github.com/quantcdn/pulumi-quant` download URL convention means Pulumi will look for assets on the GitHub release matching the SDK version.

## What Gets Deleted

All 16 boilerplate workflow files replaced by 3 clean files:
- `build_provider.yml`, `build_sdk.yml`, `license.yml`, `lint.yml`, `main-post-build.yml`, `main.yml`, `prerelease.yml`, `prerequisites.yml`, `publish.yml`, `pull-request.yml`, `release.yml`, `run-acceptance-tests.yml`, `test.yml`, `upgrade-bridge.yml`, `upgrade-provider.yml`, `verify-release.yml`

Also remove boilerplate GitHub Actions in `.github/actions/` that are no longer referenced.

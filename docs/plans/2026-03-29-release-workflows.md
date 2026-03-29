# Release Workflows Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Replace 16 boilerplate GitHub Actions workflows with 3 self-contained workflows that build, test, and publish the pulumi-quant provider.

**Architecture:** Three workflows — `ci.yml` (PR validation), `release.yml` (full build/test/publish pipeline on tag), `upgrade.yml` (upstream TF provider update automation). Plus a dispatch notification added to terraform-provider-quant's release workflow.

**Tech Stack:** GitHub Actions, Go cross-compilation, npm OIDC trusted publishing, `peter-evans/repository-dispatch`

---

### Task 1: Delete boilerplate workflows and actions

**Files:**
- Delete: all 16 files in `/Users/stuart/apps/pulumi-quant/.github/workflows/`
- Delete: all 8 directories in `/Users/stuart/apps/pulumi-quant/.github/actions/`

- [ ] **Step 1: Remove all boilerplate workflow files**

```bash
cd /Users/stuart/apps/pulumi-quant
rm -rf .github/workflows/*.yml .github/actions/
```

- [ ] **Step 2: Verify clean state**

```bash
ls .github/workflows/ .github/actions/ 2>&1
```

Expected: workflows directory empty, actions directory not found.

- [ ] **Step 3: Commit**

```bash
git add -A
git commit -m "chore: remove boilerplate CI workflows and actions"
```

---

### Task 2: Create CI workflow

**Files:**
- Create: `/Users/stuart/apps/pulumi-quant/.github/workflows/ci.yml`

- [ ] **Step 1: Write ci.yml**

```yaml
name: CI

on:
  pull_request:
  push:
    branches: [main]

permissions:
  contents: read

jobs:
  build-and-test:
    name: Build & Test
    runs-on: ubuntu-latest
    timeout-minutes: 15
    steps:
      - uses: actions/checkout@v4

      - uses: actions/setup-go@v5
        with:
          go-version-file: provider/go.mod
          cache-dependency-path: provider/go.sum

      - name: Build tfgen
        run: |
          cd provider
          go build -ldflags "-X github.com/quantcdn/pulumi-quant/provider/pkg/version.Version=0.0.0-dev" \
            -o ../bin/pulumi-tfgen-quant ./cmd/pulumi-tfgen-quant

      - name: Build resource provider
        run: |
          cd provider
          go build -ldflags "-X github.com/quantcdn/pulumi-quant/provider/pkg/version.Version=0.0.0-dev" \
            -o ../bin/pulumi-resource-quant ./cmd/pulumi-resource-quant

      - name: Run provider tests
        run: cd provider && go test -v ./...
```

- [ ] **Step 2: Validate YAML syntax**

```bash
python3 -c "import yaml; yaml.safe_load(open('.github/workflows/ci.yml'))" && echo "Valid YAML"
```

Expected: "Valid YAML"

- [ ] **Step 3: Commit**

```bash
git add .github/workflows/ci.yml
git commit -m "ci: add PR and push validation workflow"
```

---

### Task 3: Create release workflow

**Files:**
- Create: `/Users/stuart/apps/pulumi-quant/.github/workflows/release.yml`

- [ ] **Step 1: Write release.yml**

```yaml
name: Release

on:
  push:
    tags:
      - "v*.*.*"

env:
  PROVIDER: quant
  GO_VERSION_FILE: provider/go.mod

permissions:
  contents: write
  id-token: write

jobs:
  # --- Extract version from tag ---
  version:
    name: Extract version
    runs-on: ubuntu-latest
    outputs:
      version: ${{ steps.version.outputs.version }}
    steps:
      - id: version
        run: echo "version=${GITHUB_REF_NAME#v}" >> "$GITHUB_OUTPUT"

  # --- Build & test ---
  test:
    name: Test
    runs-on: ubuntu-latest
    timeout-minutes: 15
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with:
          go-version-file: provider/go.mod
          cache-dependency-path: provider/go.sum
      - name: Run tests
        run: cd provider && go test -v ./...

  # --- Cross-compile provider binary ---
  build-provider:
    name: Build provider (${{ matrix.os }}-${{ matrix.arch }})
    needs: [version, test]
    runs-on: ubuntu-latest
    strategy:
      matrix:
        include:
          - { os: linux,   arch: amd64, goos: linux,   goarch: amd64 }
          - { os: linux,   arch: arm64, goos: linux,   goarch: arm64 }
          - { os: darwin,  arch: amd64, goos: darwin,  goarch: amd64 }
          - { os: darwin,  arch: arm64, goos: darwin,  goarch: arm64 }
          - { os: windows, arch: amd64, goos: windows, goarch: amd64 }
          - { os: windows, arch: arm64, goos: windows, goarch: arm64 }
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with:
          go-version-file: provider/go.mod
          cache-dependency-path: provider/go.sum

      - name: Build binary
        env:
          VERSION: ${{ needs.version.outputs.version }}
          GOOS: ${{ matrix.goos }}
          GOARCH: ${{ matrix.goarch }}
          CGO_ENABLED: "0"
        run: |
          cd provider
          BINARY_NAME="pulumi-resource-${{ env.PROVIDER }}"
          if [ "$GOOS" = "windows" ]; then BINARY_NAME="${BINARY_NAME}.exe"; fi
          go build -ldflags "-s -w -X github.com/quantcdn/pulumi-quant/provider/pkg/version.Version=${VERSION}" \
            -o "../bin/${BINARY_NAME}" ./cmd/pulumi-resource-${{ env.PROVIDER }}

      - name: Package tarball
        env:
          VERSION: ${{ needs.version.outputs.version }}
        run: |
          TARBALL="pulumi-resource-${{ env.PROVIDER }}-v${VERSION}-${{ matrix.os }}-${{ matrix.arch }}.tar.gz"
          tar -czf "${TARBALL}" -C bin .
          echo "TARBALL=${TARBALL}" >> "$GITHUB_ENV"

      - uses: actions/upload-artifact@v4
        with:
          name: provider-${{ matrix.os }}-${{ matrix.arch }}
          path: ${{ env.TARBALL }}

  # --- Create GitHub release ---
  publish-github-release:
    name: GitHub Release
    needs: [version, build-provider]
    runs-on: ubuntu-latest
    steps:
      - uses: actions/download-artifact@v4
        with:
          path: artifacts
          pattern: provider-*
          merge-multiple: true

      - uses: softprops/action-gh-release@v2
        with:
          generate_release_notes: true
          files: artifacts/*.tar.gz

  # --- Publish Node.js SDK to npm ---
  publish-npm:
    name: Publish npm
    needs: [version, test]
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4

      - uses: actions/setup-go@v5
        with:
          go-version-file: provider/go.mod
          cache-dependency-path: provider/go.sum

      - uses: actions/setup-node@v4
        with:
          node-version: "22"
          registry-url: "https://registry.npmjs.org"

      - name: Update npm
        run: npm install -g npm@latest

      - name: Build tfgen
        env:
          VERSION: ${{ needs.version.outputs.version }}
        run: |
          cd provider
          go build -ldflags "-X github.com/quantcdn/pulumi-quant/provider/pkg/version.Version=${VERSION}" \
            -o ../bin/pulumi-tfgen-${{ env.PROVIDER }} ./cmd/pulumi-tfgen-${{ env.PROVIDER }}

      - name: Generate Node.js SDK
        run: ./bin/pulumi-tfgen-${{ env.PROVIDER }} nodejs --out sdk/nodejs

      - name: Set SDK version
        env:
          VERSION: ${{ needs.version.outputs.version }}
        run: |
          cd sdk/nodejs
          npm version "${VERSION}" --no-git-tag-version --allow-same-version

      - name: Build SDK
        run: |
          cd sdk/nodejs
          yarn install --no-lockfile
          yarn run build

      - name: Publish to npm
        run: cd sdk/nodejs && npm publish --access public
        # Do NOT set NODE_AUTH_TOKEN — OIDC trusted publishing requires it absent

  # --- Placeholder: future SDK publishing ---
  # publish-pypi:
  #   name: Publish PyPI
  #   needs: [version, test]
  #   runs-on: ubuntu-latest
  #   steps:
  #     - Build tfgen, generate python SDK, pip install build, python -m build
  #     - pypa/gh-action-pypi-publish with OIDC trusted publishing
  #
  # publish-nuget:
  #   name: Publish NuGet
  #   needs: [version, test]
  #   runs-on: ubuntu-latest
  #   steps:
  #     - Build tfgen, generate dotnet SDK, dotnet pack, dotnet nuget push
```

- [ ] **Step 2: Validate YAML syntax**

```bash
python3 -c "import yaml; yaml.safe_load(open('.github/workflows/release.yml'))" && echo "Valid YAML"
```

Expected: "Valid YAML"

- [ ] **Step 3: Commit**

```bash
git add .github/workflows/release.yml
git commit -m "ci: add release workflow with cross-platform builds and npm OIDC publishing"
```

---

### Task 4: Create upgrade workflow

**Files:**
- Create: `/Users/stuart/apps/pulumi-quant/.github/workflows/upgrade.yml`

- [ ] **Step 1: Write upgrade.yml**

```yaml
name: Upgrade Upstream Provider

on:
  repository_dispatch:
    types: [upstream-release]
  workflow_dispatch:
    inputs:
      version:
        description: "Upstream terraform-provider-quant version (e.g. v5.0.3)"
        required: true

permissions:
  contents: write
  pull-requests: write

jobs:
  upgrade:
    name: Upgrade terraform-provider-quant
    runs-on: ubuntu-latest
    timeout-minutes: 15
    steps:
      - uses: actions/checkout@v4

      - uses: actions/setup-go@v5
        with:
          go-version-file: provider/go.mod
          cache-dependency-path: provider/go.sum

      - uses: actions/setup-node@v4
        with:
          node-version: "22"

      - name: Determine version
        id: version
        run: |
          if [ "${{ github.event_name }}" = "repository_dispatch" ]; then
            VERSION="${{ github.event.client_payload.version }}"
          else
            VERSION="${{ inputs.version }}"
          fi
          echo "version=${VERSION}" >> "$GITHUB_OUTPUT"
          echo "Upgrading to ${VERSION}"

      - name: Update Go dependency
        run: |
          cd provider
          go get "github.com/quantcdn/terraform-provider-quant/v5@${{ steps.version.outputs.version }}"
          go mod tidy

      - name: Build tfgen
        run: |
          cd provider
          go build -ldflags "-X github.com/quantcdn/pulumi-quant/provider/pkg/version.Version=0.0.0-dev" \
            -o ../bin/pulumi-tfgen-quant ./cmd/pulumi-tfgen-quant

      - name: Regenerate schema
        run: ./bin/pulumi-tfgen-quant schema --out provider/cmd/pulumi-resource-quant

      - name: Regenerate Node.js SDK
        run: ./bin/pulumi-tfgen-quant nodejs --out sdk/nodejs

      - name: Regenerate Python SDK
        run: ./bin/pulumi-tfgen-quant python --out sdk/python

      - name: Regenerate Go SDK
        run: ./bin/pulumi-tfgen-quant go --out sdk/go

      - name: Regenerate .NET SDK
        run: ./bin/pulumi-tfgen-quant dotnet --out sdk/dotnet

      - name: Build and test provider
        run: |
          cd provider
          go build -ldflags "-X github.com/quantcdn/pulumi-quant/provider/pkg/version.Version=0.0.0-dev" \
            -o ../bin/pulumi-resource-quant ./cmd/pulumi-resource-quant
          go test -v ./...

      - name: Create pull request
        uses: peter-evans/create-pull-request@v7
        with:
          branch: upgrade/terraform-provider-quant-${{ steps.version.outputs.version }}
          title: "chore: upgrade terraform-provider-quant to ${{ steps.version.outputs.version }}"
          body: |
            Automated upgrade of upstream `terraform-provider-quant` to `${{ steps.version.outputs.version }}`.

            - Updated `provider/go.mod` dependency
            - Regenerated Pulumi schema
            - Regenerated all SDKs (TypeScript, Python, Go, .NET)
            - Provider builds and tests pass
          commit-message: "chore: upgrade terraform-provider-quant to ${{ steps.version.outputs.version }}"
```

- [ ] **Step 2: Validate YAML syntax**

```bash
python3 -c "import yaml; yaml.safe_load(open('.github/workflows/upgrade.yml'))" && echo "Valid YAML"
```

Expected: "Valid YAML"

- [ ] **Step 3: Commit**

```bash
git add .github/workflows/upgrade.yml
git commit -m "ci: add upstream provider upgrade workflow with auto-PR"
```

---

### Task 5: Add dispatch notification to terraform-provider-quant

**Files:**
- Modify: `/Users/stuart/apps/terraform-provider-quant/.github/workflows/release.yml`

- [ ] **Step 1: Add notify-pulumi job to release.yml**

Add this job after the existing `goreleaser` job:

```yaml
  notify-pulumi:
    name: Notify pulumi-quant
    needs: goreleaser
    runs-on: ubuntu-latest
    steps:
      - name: Dispatch upstream-release event
        uses: peter-evans/repository-dispatch@v3
        with:
          token: ${{ secrets.PULUMI_QUANT_DISPATCH_TOKEN }}
          repository: quantcdn/pulumi-quant
          event-type: upstream-release
          client-payload: '{"version": "${{ github.ref_name }}"}'
```

The full file should look like:

```yaml
# Terraform Provider release workflow.
name: Release

on:
  push:
    tags:
      - 'v*'

permissions:
  contents: write

jobs:
  goreleaser:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v6
        with:
          fetch-depth: 0
      - uses: actions/setup-go@v6
        with:
          go-version-file: 'go.mod'
          cache: true
      - name: Import GPG key
        uses: crazy-max/ghaction-import-gpg@01dd5d3ca463c7f10f7f4f7b4f177225ac661ee4 # v6.1.0
        id: import_gpg
        with:
          gpg_private_key: ${{ secrets.GPG_PRIVATE_KEY }}
          passphrase: ${{ secrets.PASSPHRASE }}
      - name: Run GoReleaser
        uses: goreleaser/goreleaser-action@286f3b13b1b49da4ac219696163fb8c1c93e1200 # v6.0.0
        with:
          args: release --clean
        env:
          GITHUB_TOKEN: ${{ secrets.GITHUB_TOKEN }}
          GPG_FINGERPRINT: ${{ steps.import_gpg.outputs.fingerprint }}

  notify-pulumi:
    name: Notify pulumi-quant
    needs: goreleaser
    runs-on: ubuntu-latest
    steps:
      - name: Dispatch upstream-release event
        uses: peter-evans/repository-dispatch@v3
        with:
          token: ${{ secrets.PULUMI_QUANT_DISPATCH_TOKEN }}
          repository: quantcdn/pulumi-quant
          event-type: upstream-release
          client-payload: '{"version": "${{ github.ref_name }}"}'
```

- [ ] **Step 2: Commit and push**

```bash
cd /Users/stuart/apps/terraform-provider-quant
git checkout -b ci/notify-pulumi-on-release
git add .github/workflows/release.yml
git commit -m "ci: notify pulumi-quant on new releases via repository_dispatch"
git push -u origin ci/notify-pulumi-on-release
```

- [ ] **Step 3: Create PR**

```bash
gh pr create --title "ci: notify pulumi-quant on new releases" --body "Adds a repository_dispatch event to notify quantcdn/pulumi-quant when a new terraform-provider-quant release is published. Requires PULUMI_QUANT_DISPATCH_TOKEN secret (GitHub PAT with repo scope)."
```

- [ ] **Step 4: Document the secret requirement**

The `PULUMI_QUANT_DISPATCH_TOKEN` secret must be created in terraform-provider-quant's repo settings. It needs a GitHub PAT (fine-grained) with:
- Repository access: `quantcdn/pulumi-quant`
- Permissions: Contents (read/write)

---

### Task 6: Push pulumi-quant changes and verify

**Files:**
- No new files

- [ ] **Step 1: Push all commits to pulumi-quant**

```bash
cd /Users/stuart/apps/pulumi-quant
git push origin main
```

- [ ] **Step 2: Verify CI workflow runs on push**

```bash
gh run list --repo quantcdn/pulumi-quant --limit 1
```

Expected: A workflow run from the push to main.

- [ ] **Step 3: Test release workflow with a tag**

```bash
cd /Users/stuart/apps/pulumi-quant
git tag v0.1.1
git push origin v0.1.1
```

- [ ] **Step 4: Monitor release**

```bash
gh run watch --repo quantcdn/pulumi-quant
```

Expected: All jobs pass — test, build-provider (6 platforms), publish-github-release, publish-npm.

- [ ] **Step 5: Verify npm publish**

```bash
npm view @quantcdn/pulumi-quant version
```

Expected: `0.1.1`

- [ ] **Step 6: Verify GitHub release assets**

```bash
gh release view v0.1.1 --repo quantcdn/pulumi-quant --json assets --jq '.assets[].name'
```

Expected: 6 tarballs (linux/darwin/windows x amd64/arm64).

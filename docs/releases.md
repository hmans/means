# Releases

Means uses [Sweet Release 0.3.0](https://github.com/chattocorp/sweet-release/releases/tag/sweet%400.3.0)
to manage change notes, release PRs, versions, changelogs, and tags. GoReleaser builds
binaries and uploads them to a draft GitHub Release when a `v*` tag is pushed.
Maintainers review and publish the draft on GitHub.

Run `mise install` before using these commands. The `mise sweet` task runs the
pinned Sweet version with Go; it does not require a global `sweet` installation.
The Go commands pin the commit behind `sweet@0.3.0`, because that release tag is
not a Go module version tag. The shared workflows download the published binary.

## GitHub setup

Add a repository Actions secret named `RELEASE_PR_TOKEN`. Use a fine-grained
personal token with access to `hmans/means`, Contents read/write, and Pull requests
read/write. The same token creates release PRs and pushes tags, so those actions
can trigger subsequent GitHub workflows. The built-in `GITHUB_TOKEN` does not
trigger CI for those PRs or the tag-driven release workflow.

The shared workflows are pinned to Sweet Release commit
`ccee4355cc84229bf23b3f6dfd9658dd21672cba`; their CLI input is `sweet@0.3.0`.
Update the workflow references in `bump.yml` and `ci.yml` together when upgrading.

## Record a change

Run `mise sweet change .` and commit the generated `.sweet/*.md` file with the
code. For non-interactive use:

```sh
printf 'Describe the user-visible change.\n' | mise sweet change . --type fix --stdin
mise sweet status
```

Features bump the minor version. Fixes and performance changes bump the patch
version. Chores and refactors do not cause a release. Set `breaking: true` in a
note's frontmatter for breaking changes; these require a minor bump before 1.0
and a major bump after that. Policies live in `.sweet.config.yml`.

## Prepare a release

Every push to `main` starts the shared `bump.yml` workflow. It updates `.sweet.yml`,
writes `CHANGELOG.md`, consumes change notes, and opens or updates a release PR.
Review and merge that PR through the normal review process. With no releasable
notes, no release PR is needed.

CI runs Go race tests and vet for PRs and `main` pushes. After checks pass on
`main`, the shared `release.yml` workflow examines that push's commit range and
creates tags for committed version changes. Main runs are not canceled by newer
pushes. Root-package tags retain the `v<version>` format.
PRs also run the shared release planner with read-only permissions and tag pushes
disabled, so changes to release configuration are checked before merge.

For a local preview, use a clean checkout with full Git history:

```sh
git fetch origin --tags
mise sweet status
mise sweet bump .
git diff
mise test
mise run release:snapshot
```

The bump commands modify the checkout. Use a separate worktree for previews;
let the automated PR own release preparation on `main`. Tags point to the commit
that introduced the version, even if later commits exist. Existing versions are
not released again just because the automation is installed.

## Build and publish

The release workflow checks that the tag matches `.sweet.yml` and extracts the
matching changelog entry with `sweet changelog`. It runs Go tests and vet, then
builds Linux, macOS, and Windows binaries for amd64 and arm64. Archives use
tar.gz on Linux/macOS and ZIP on Windows, with `checksums.txt` for SHA-256 hashes.
The linker sets the version reported by `means version`.

The workflow uploads a draft named `means <version>`. Review its notes and assets
on GitHub, then publish it. Versions with prerelease suffixes are marked as
prereleases. The workflow uses the built-in `GITHUB_TOKEN` with `contents: write`;
the separate PR/tag steps use `RELEASE_PR_TOKEN` as described above.

If release PR preparation fails, rerun all jobs in the Bump run. If tagging fails,
rerun the original CI run to preserve its commit range. If the tag already exists
but the publishing workflow did not start, dispatch it explicitly:

```sh
gh workflow run release.yml --ref main -f tag=v0.1.0
```

If a failed upload left a partial draft, delete that draft before retrying. Keep
the Git tag. Do not rerun publishing for an already published release.

For local checks, use `mise run release:check` to validate the GoReleaser config
and `mise run release:snapshot` to run tests, vet, and all six builds. Snapshots
write archives into the ignored `dist/` directory and do not publish or create
tags. Their version is `0.0.0-SNAPSHOT-<commit>`; ordinary source builds retain
the `dev` version.

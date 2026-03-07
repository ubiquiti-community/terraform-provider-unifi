#!/usr/bin/env bash
# update-release-notes.sh
#
# Updates GitHub release descriptions for releases that only contain
# "Full Changelog" links in their descriptions.
#
# Usage:
#   GITHUB_TOKEN=<token> ./scripts/update-release-notes.sh
#
# Requirements:
#   - gh CLI installed and authenticated, OR GH_TOKEN/GITHUB_TOKEN environment variable set

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
NOTES_DIR="${SCRIPT_DIR}/release-notes"

RELEASES=(
  "v0.41.19"
  "v0.41.18"
  "v0.41.13"
  "v0.41.5"
  "v0.41.4-rc3"
  "v0.41.4-rc2"
  "v0.41.4-beta2"
)

echo "Updating release notes for ubiquiti-community/terraform-provider-unifi..."
echo ""

for tag in "${RELEASES[@]}"; do
  notes_file="${NOTES_DIR}/${tag}.md"
  if [ ! -f "$notes_file" ]; then
    echo "⚠️  Skipping $tag: notes file not found at $notes_file"
    continue
  fi

  echo "Updating $tag..."
  if GH_TOKEN="${GITHUB_TOKEN:-}" gh release edit "$tag" \
    --repo ubiquiti-community/terraform-provider-unifi \
    --notes-file "$notes_file"; then
    echo "  ✓ Updated successfully"
  else
    echo "  ✗ Failed to update $tag"
  fi
done

echo ""
echo "Done!"

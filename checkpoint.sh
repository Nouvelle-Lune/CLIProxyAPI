#!/usr/bin/env bash
set -euo pipefail

PREFIX="cliproxy-checkpoint"
DEFAULT_BASELINE="baseline"
DEFAULT_WIP="wip"

usage() {
  cat <<'USAGE'
Usage:
  ./checkpoint.sh baseline [name]
      Save the current working tree as a baseline snapshot and keep the tree unchanged.

  ./checkpoint.sh save [name]
      Save the current half-finished working tree as a WIP snapshot and keep the tree unchanged.

  ./checkpoint.sh restore-baseline [name]
      Restore a baseline snapshot. Current changes are autosaved before switching.

  ./checkpoint.sh restore-wip [name]
      Restore a WIP snapshot. Current changes are autosaved before switching.

  ./checkpoint.sh restore <baseline|wip|autosave> <name>
      Restore a specific snapshot kind and name.

  ./checkpoint.sh list
      List snapshots managed by this script.

  ./checkpoint.sh drop <baseline|wip|autosave> <name>
      Drop the latest matching snapshot.

Examples:
  ./checkpoint.sh baseline before-remote-merge
  ./checkpoint.sh save merge-halfway
  ./checkpoint.sh restore-baseline before-remote-merge
  ./checkpoint.sh restore-wip merge-halfway
USAGE
}

die() {
  printf 'checkpoint.sh: %s\n' "$*" >&2
  exit 1
}

ensure_repo() {
  local root
  root=$(git rev-parse --show-toplevel 2>/dev/null) || die "not inside a Git repository"
  cd "$root"
}

validate_name() {
  local name=$1
  [[ -n "$name" ]] || die "snapshot name must not be empty"
  [[ "$name" =~ ^[A-Za-z0-9._-]+$ ]] || die "snapshot name may only contain letters, numbers, dots, underscores, and hyphens"
}

validate_kind() {
  case "$1" in
    baseline|wip|autosave) ;;
    *) die "snapshot kind must be one of: baseline, wip, autosave" ;;
  esac
}

message_for() {
  local kind=$1
  local name=$2
  printf '%s:%s:%s' "$PREFIX" "$kind" "$name"
}

has_changes() {
  [[ -n "$(git status --porcelain --untracked-files=all)" ]]
}

save_snapshot() {
  local kind=$1
  local name=$2
  validate_kind "$kind"
  validate_name "$name"

  if ! has_changes; then
    die "working tree is clean; there is no file state to snapshot"
  fi

  local message
  message=$(message_for "$kind" "$name")

  # Stash first, then immediately re-apply, so saving a checkpoint never interrupts ongoing edits.
  git stash push --include-untracked --message "$message" >/dev/null
  git stash apply --index stash@{0} >/dev/null

  printf 'Saved %s snapshot: %s\n' "$kind" "$name"
}

find_snapshot_hash() {
  local kind=$1
  local name=$2
  local expected
  expected=$(message_for "$kind" "$name")

  git stash list --format='%H%x09%gs' | while IFS=$'\t' read -r hash subject; do
    case "$subject" in
      *": $expected")
        printf '%s\n' "$hash"
        return 0
        ;;
    esac
  done
}

find_snapshot_ref() {
  local kind=$1
  local name=$2
  local expected
  expected=$(message_for "$kind" "$name")

  git stash list --format='%gd%x09%gs' | while IFS=$'\t' read -r ref subject; do
    case "$subject" in
      *": $expected")
        printf '%s\n' "$ref"
        return 0
        ;;
    esac
  done
}

autosave_current_changes() {
  if ! has_changes; then
    return 0
  fi

  local name
  name="before-restore-$(date +%Y%m%d-%H%M%S)"

  # Restore is intentionally non-lossy: the current tree is stashed before applying another snapshot.
  git stash push --include-untracked --message "$(message_for autosave "$name")" >/dev/null
  printf 'Autosaved current changes as autosave snapshot: %s\n' "$name"
}

restore_snapshot() {
  local kind=$1
  local name=$2
  validate_kind "$kind"
  validate_name "$name"

  local hash
  hash=$(find_snapshot_hash "$kind" "$name")
  [[ -n "$hash" ]] || die "snapshot not found: $kind $name"

  autosave_current_changes
  git stash apply --index "$hash" >/dev/null

  printf 'Restored %s snapshot: %s\n' "$kind" "$name"
}

list_snapshots() {
  local found=0
  local entries
  entries=$(git stash list --format='%gd%x09%cr%x09%gs')

  while IFS=$'\t' read -r ref age subject; do
    [[ -n "$ref" ]] || continue
    case "$subject" in
      *": $PREFIX:"*)
        found=1
        printf '%s\t%s\t%s\n' "$ref" "$age" "${subject##*: }"
        ;;
    esac
  done <<< "$entries"

  if [[ "$found" -eq 0 ]]; then
    printf 'No checkpoints found.\n'
  fi
}

drop_snapshot() {
  local kind=$1
  local name=$2
  validate_kind "$kind"
  validate_name "$name"

  local ref
  ref=$(find_snapshot_ref "$kind" "$name")
  [[ -n "$ref" ]] || die "snapshot not found: $kind $name"

  git stash drop "$ref" >/dev/null
  printf 'Dropped %s snapshot: %s\n' "$kind" "$name"
}

main() {
  ensure_repo

  local command=${1:-}
  case "$command" in
    baseline)
      save_snapshot baseline "${2:-$DEFAULT_BASELINE}"
      ;;
    save)
      save_snapshot wip "${2:-$DEFAULT_WIP}"
      ;;
    restore-baseline)
      restore_snapshot baseline "${2:-$DEFAULT_BASELINE}"
      ;;
    restore-wip)
      restore_snapshot wip "${2:-$DEFAULT_WIP}"
      ;;
    restore)
      [[ $# -eq 3 ]] || die "restore requires <baseline|wip|autosave> <name>"
      restore_snapshot "$2" "$3"
      ;;
    list)
      list_snapshots
      ;;
    drop)
      [[ $# -eq 3 ]] || die "drop requires <baseline|wip|autosave> <name>"
      drop_snapshot "$2" "$3"
      ;;
    -h|--help|help|'')
      usage
      ;;
    *)
      usage >&2
      die "unknown command: $command"
      ;;
  esac
}

main "$@"

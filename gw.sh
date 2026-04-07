#!/usr/bin/env bash
# handy shell functions

# Git worktree helpers
#
# Usage:
#   gw start  [-d subdir] <branch> [base]         — create worktree + focus + open in VS Code
#   gw focus  [--force] [--new-window] <branch>   — start focused session
#   gw park   [branch]                            — save exit note, clear active session
#   gw finish [branch]                            — park, then remove worktree and clean up
#   gw list                                       — list active worktrees
#   gw log    [range] [flags]                     — show session log (delegates to gw-log)
#
# Session state lives in ~/.gw/ — never touches the repo.
# Worktree-local files (git-excluded):
#   .gw-focus    — session goal, picked up by AI tooling
#   .gw-activity — EARLY activity ID for this worktree, set at gw start
#
# Hooks (executable scripts in ~/.gw/hooks/):
#   on-start.d/*  — run after worktree creation     (args: branch worktree_path)
#   on-focus.d/*  — run when entering focus         (args: branch worktree_path)
#   on-park.d/*   — run when parking a session      (args: branch worktree_path note)
#
# Requires: jq, yq, gw-log (Go binary, go build ./cmd/gw-log/)

# ── Guards ───────────────────────────────────────────────────────────────────

_gw_check_deps() {
  local missing=()
  command -v jq &>/dev/null || missing+=(jq)
  command -v yq &>/dev/null || missing+=(yq)
  if [[ ${#missing[@]} -gt 0 ]]; then
    echo "gw: missing dependencies: ${missing[*]} (brew install ${missing[*]})" >&2
    return 1
  fi
}
_gw_check_deps || { [[ "${BASH_SOURCE[0]}" == "$0" ]] && exit 1 || return 1; }

# ── State helpers ────────────────────────────────────────────────────────────

GW_STATE_DIR="${HOME}/.gw"

# Read a value from ~/.gw/config.yaml
_gw_read_config() {
  local key="$1" cfg="${GW_STATE_DIR}/config.yaml"
  [[ -f "$cfg" ]] || return 1
  yq -r ".${key} // \"\"" "$cfg"
}

# Resolve sessions directory: env > config > default
_gw_sessions_dir() {
  if [[ -n "${GW_SESSIONS_DIR:-}" ]]; then
    echo "$GW_SESSIONS_DIR"
  else
    local val
    val="$(_gw_read_config sessions_dir)" && [[ -n "$val" ]] && echo "$val" \
      || echo "${GW_STATE_DIR}/sessions"
  fi
}

_gw_active_file()  { echo "${GW_STATE_DIR}/active"; }
_gw_session_dir()  { echo "$(_gw_sessions_dir)/${1//\//-}"; }

_gw_set_active()   { mkdir -p "$GW_STATE_DIR"; echo "$1" > "$(_gw_active_file)"; }
_gw_clear_active() { rm -f "$(_gw_active_file)"; }
_gw_get_active()   { local f="$(_gw_active_file)"; [[ -f "$f" ]] && cat "$f" || echo ""; }

_gw_worktree_path() {
  local branch="$1"
  local repo_root
  repo_root="$(git rev-parse --show-toplevel)" || return 1
  local repo_name
  repo_name="$(basename "$(git worktree list --porcelain | head -1 | sed 's/^worktree //')")"
  echo "$(dirname "$repo_root")/${repo_name}-${branch//\//-}"
}

# Exclude a filename from git in the given worktree
_gw_git_exclude() {
  local worktree_path="$1" filename="$2"
  local git_common_dir
  git_common_dir="$(git -C "$worktree_path" rev-parse --git-common-dir)"
  mkdir -p "${git_common_dir}/info"
  grep -qxF "$filename" "${git_common_dir}/info/exclude" 2>/dev/null \
    || echo "$filename" >> "${git_common_dir}/info/exclude"
}

# ── Hook dispatcher ──────────────────────────────────────────────────────────

# Run all executable scripts in ~/.gw/hooks/<event>.d/ in sorted order.
# Args after event are forwarded to each hook.
_gw_run_hooks() {
  local event="$1"; shift
  local hook_dir="${GW_STATE_DIR}/hooks/${event}.d"
  [[ -d "$hook_dir" ]] || return 0
  local hook
  for hook in "$hook_dir"/*; do
    [[ -x "$hook" ]] && "$hook" "$@"
  done
}

# ── Box printer ──────────────────────────────────────────────────────────────

# Print a bordered box with a title and content from a file
_gw_print_box() {
  local title="$1" file="$2"
  local -a lines
  while IFS= read -r line; do lines+=("$line"); done < "$file"
  [[ ${#lines[@]} -eq 0 ]] && return

  # Compute inner width: max(longest line, 40) + 4 for padding
  local max_len=40
  for line in "${lines[@]}"; do
    (( ${#line} > max_len )) && max_len=${#line}
  done
  local inner=$(( max_len + 4 ))  # "  " on each side

  # Top border: ┌─ title ───...───┐
  local title_seg="─ ${title} "
  local fill_len=$(( inner - ${#title_seg} ))
  local fill=""
  local i
  for (( i=0; i<fill_len; i++ )); do fill+="─"; done
  echo ""
  echo "  ┌${title_seg}${fill}┐"

  # Content lines: │  text       │
  for line in "${lines[@]}"; do
    local pad_len=$(( inner - 2 - ${#line} - 2 ))
    local pad=""
    for (( i=0; i<pad_len; i++ )); do pad+=" "; done
    echo "  │  ${line}${pad}  │"
  done

  # Bottom border: └───...───┘
  fill=""
  for (( i=0; i<inner; i++ )); do fill+="─"; done
  echo "  └${fill}┘"
  echo ""
}

# ── Main dispatcher ──────────────────────────────────────────────────────────

gw() {
  local cmd="${1:-}"
  [[ $# -gt 0 ]] && shift

  case "$cmd" in
    start)  _gw_start  "$@" ;;
    focus)  _gw_focus  "$@" ;;
    park)   _gw_park   "$@" ;;
    finish) _gw_finish "$@" ;;
    list)   git worktree list ;;
    log)    gw-log --sessions-dir="$(_gw_sessions_dir)" read "$@" ;;
    *)
      echo "usage: gw <start|focus|park|finish|list|log>" >&2
      echo "  start  [-d subdir] <branch> [base]          — create worktree + focus + open" >&2
      echo "  focus  [--force] [--new-window] <branch>    — begin session" >&2
      echo "  park   [branch]                             — save exit note and release session" >&2
      echo "  finish [branch]                              — park + remove worktree" >&2
      echo "  list                                        — list active worktrees" >&2
      echo "  log    [range] [--first=D] [--last=D]        — show session log" >&2
      return 1
      ;;
  esac
}

# ── gw start ─────────────────────────────────────────────────────────────────

_gw_start() {
  local subdir=""
  if [[ "${1:-}" == "-d" ]]; then
    subdir="$2"
    shift 2
  fi

  local branch="$1"
  local base="${2:-HEAD}"

  if [[ -z "$branch" ]]; then
    echo "usage: gw start [-d subdir] <branch> [base]" >&2
    return 1
  fi

  local repo_root
  repo_root="$(git rev-parse --show-toplevel)" || return 1
  local repo_name
  repo_name="$(basename "$repo_root")"

  local dir_suffix="${branch//\//-}"
  local worktree_path="$(dirname "$repo_root")/${repo_name}-${dir_suffix}"
  local open_path="$worktree_path${subdir:+/$subdir}"

  if [[ -d "$worktree_path" ]]; then
    echo "worktree already exists: $worktree_path" >&2
    echo "opening in VS Code..." >&2
    if [[ -n "$subdir" && ! -d "$open_path" ]]; then
      echo "warning: $open_path doesn't exist yet, opening worktree root" >&2
      open_path="$worktree_path"
    fi
    code "$open_path"
    return 0
  fi

  if git show-ref --verify --quiet "refs/heads/$branch"; then
    git worktree add "$worktree_path" "$branch"
  else
    git worktree add -b "$branch" "$worktree_path" "$base"
  fi || return 1

  echo "worktree ready: $worktree_path"

  # Run on-start hooks (e.g. EARLY activity selection)
  _gw_run_hooks on-start "$branch" "$worktree_path"

  # Git-exclude any files hooks may have written
  _gw_git_exclude "$worktree_path" ".gw-activity"

  # Focus immediately — new worktree always gets a new window
  _gw_focus --new-window "$branch"
}

# ── gw focus ─────────────────────────────────────────────────────────────────

_gw_focus() {
  local force=0
  local new_window=0

  while [[ "${1:-}" == --* ]]; do
    case "$1" in
      --force)      force=1; shift ;;
      --new-window) new_window=1; shift ;;
      *) echo "unknown flag: $1" >&2; return 1 ;;
    esac
  done

  local branch="${1:-}"
  if [[ -z "$branch" ]]; then
    echo "usage: gw focus [--force] [--new-window] <branch>" >&2
    return 1
  fi

  # Block if a different session is active and unparked
  local active
  active="$(_gw_get_active)"
  if [[ -n "$active" && "$active" != "$branch" && "$force" -eq 0 ]]; then
    echo "⚠  active session: $active (unparked)" >&2
    echo ""
    echo "   run: gw park $active" >&2
    echo "   or:  gw focus --force $branch" >&2
    return 1
  fi

  local worktree_path
  worktree_path="$(_gw_worktree_path "$branch")" || return 1

  if [[ ! -d "$worktree_path" ]]; then
    echo "no worktree found for: $branch — run gw start first" >&2
    return 1
  fi

  local session_dir
  session_dir="$(_gw_session_dir "$branch")"
  mkdir -p "$session_dir"

  # Surface last park note
  if [[ -f "${session_dir}/park" ]]; then
    _gw_print_box "last parked" "${session_dir}/park"
  fi

  # Prompt for session goal
  echo ""
  printf '  session outcome (past tense — "fixed X", "shipped Y"): '
  local goal
  read -r goal

  if [[ -n "$goal" ]]; then
    gw-log --sessions-dir="$(_gw_sessions_dir)" write --type=focus --branch="$branch" --note="$goal"

    # Write to worktree root for AI context pickup
    echo "$goal" > "${worktree_path}/.gw-focus"
    _gw_git_exclude "$worktree_path" ".gw-focus"
  fi

  echo ""
  _gw_set_active "$branch"
  _gw_run_hooks on-focus "$branch" "$worktree_path"

  if [[ "$new_window" -eq 1 ]]; then
    code "$worktree_path"
  fi
}

# ── gw park ──────────────────────────────────────────────────────────────────

_gw_park() {
  local branch="${1:-}"

  # Infer from active session, then from cwd
  if [[ -z "$branch" ]]; then
    branch="$(_gw_get_active)"
  fi
  if [[ -z "$branch" ]]; then
    branch="$(git rev-parse --abbrev-ref HEAD 2>/dev/null)"
  fi
  if [[ -z "$branch" ]]; then
    echo "usage: gw park <branch>" >&2
    return 1
  fi

  local session_dir
  session_dir="$(_gw_session_dir "$branch")"
  mkdir -p "$session_dir"

  echo ""
  printf "  where did you leave off / next action? "
  local note
  read -r note

  if [[ -n "$note" ]]; then
    gw-log --sessions-dir="$(_gw_sessions_dir)" write --type=park --branch="$branch" --note="$note"
  fi

  # Remove AI context file
  local worktree_path
  worktree_path="$(_gw_worktree_path "$branch")"
  [[ -f "${worktree_path}/.gw-focus" ]] && rm -f "${worktree_path}/.gw-focus"

  _gw_clear_active
  _gw_run_hooks on-park "$branch" "$worktree_path" "$note"
  echo ""
  echo "  parked: $branch"
}

# ── gw finish ────────────────────────────────────────────────────────────────

_gw_finish() {
  local branch="${1:-}"
  local repo_root
  repo_root="$(git rev-parse --show-toplevel)" || return 1

  if [[ -z "$branch" ]]; then
    branch="$(git rev-parse --abbrev-ref HEAD)"
    local main_worktree
    main_worktree="$(git worktree list --porcelain | head -1 | sed 's/^worktree //')"
    if [[ "$repo_root" == "$main_worktree" ]]; then
      echo "you're in the main worktree — specify which branch to remove" >&2
      return 1
    fi
  fi

  local dir_suffix="${branch//\//-}"
  local repo_name
  repo_name="$(basename "$(git worktree list --porcelain | head -1 | sed 's/^worktree //')")"
  local worktree_path="$(dirname "$repo_root")/${repo_name}-${dir_suffix}"

  if [[ ! -d "$worktree_path" ]]; then
    echo "no worktree found at: $worktree_path" >&2
    return 1
  fi

  # Park the session first (prompts for closing note, runs on-park hooks)
  local active
  active="$(_gw_get_active)"
  if [[ "$active" == "$branch" || -d "$(_gw_session_dir "$branch")" ]]; then
    echo "  ── closing session ──"
    _gw_park "$branch"
  fi

  if [[ "$repo_root" == "$worktree_path" ]]; then
    local main_worktree
    main_worktree="$(git worktree list --porcelain | head -1 | sed 's/^worktree //')"
    echo "stepping out to $main_worktree"
    cd "$main_worktree" || return 1
  fi

  git worktree remove "$worktree_path" && echo "removed: $worktree_path"

  # Clean up session state
  local session_dir
  session_dir="$(_gw_session_dir "$branch")"
  if [[ -d "$session_dir" ]]; then
    rm -f "${session_dir}/focus" "${session_dir}/park"
    [[ -f "${session_dir}/log" ]] && echo "session log: ${session_dir}/log"
  fi
}

# ── Entrypoint (direct execution) ───────────────────────────────────────────

# When executed (not sourced), dispatch to gw immediately.
if [[ "${BASH_SOURCE[0]}" == "$0" ]]; then
  gw "$@"
fi

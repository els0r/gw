# gw — git worktree workflow

`gw` manages git worktrees with built-in session tracking. Each branch gets an isolated worktree and a structured log of what you intended (focus) and what actually happened (park).

Borrows from Cal Newport's [Deep Work](https://calnewport.com/deep-work-rules-for-focused-success-in-a-distracted-world/) principles: single-tasking is enforced (only one active session at a time), every session starts with a stated outcome, and context switches are tracked and made visible so you can measure focus quality over time.

## Quick start

```bash
source ~/.gw/gw.sh      # add to .zshrc / .bashrc

# build the log binary
cd ~/.gw && go build -o ~/go/bin/gw-log ./cmd/gw-log/
```

## Commands

| Command | What it does |
|---|---|
| `gw start <branch> [base]` | Create worktree, start session, open VS Code |
| `gw focus <branch>` | Begin a focused session — prompts for intended outcome |
| `gw park [branch]` | Record where you left off, release session |
| `gw finish [branch]` | Park + remove worktree |
| `gw list` | List active worktrees |
| `gw log [range] [flags]` | Show session log |

## Session log

Every focus and park writes a structured entry to `~/.gw/sessions/<branch>/log`. The `gw log` command renders these grouped by activity.

```
gw log                          # today (default)
gw log yesterday
gw log "this week"
gw log "last week"
gw log --first=2026-03-30 --last=2026-03-31
gw log --sort=asc               # most recent activity first
```

Output per activity:
- **Name** — derived from the branch (strips `feature-`/`fix-`/… prefixes, title-cases)
- **Total duration** — sum of focus-to-park times
- **Context switches** — number of focus entries, color-coded: green (1), orange (3–4), red (5+)
- **Entries** — focus notes (grey, intent) with park notes below (outcome), showing elapsed time

## Hooks

Executable scripts in `~/.gw/hooks/` run at lifecycle events:

| Directory | Trigger | Args |
|---|---|---|
| `on-start.d/` | After worktree creation | `branch worktree_path` |
| `on-focus.d/` | When entering focus | `branch worktree_path` |
| `on-park.d/` | When parking | `branch worktree_path note` |

Ships with hooks for [EARLY](https://early.app) time tracking and Slack status.

## Configuration

Hooks read credentials from `~/.gw/config`, a plain `KEY=value` file. Supported keys:

| Key | Used by | Purpose |
|---|---|---|
| `EARLY_API_KEY` | `on-start`, `on-focus`, `on-park` | EARLY API key (base64) |
| `EARLY_API_SECRET` | `on-start`, `on-focus`, `on-park` | EARLY API secret (base64) |
| `SLACK_TOKEN` | `on-focus`, `on-park` | Slack OAuth token for status updates |

Example:

```
EARLY_API_KEY=your_api_key_here
EARLY_API_SECRET=your_api_secret_here
SLACK_TOKEN=xoxp-...
```

The config file is git-ignored. EARLY tokens are cached in `~/.gw/early_token` (23h TTL, `chmod 600`).

## State

All state lives in `~/.gw/`, never in the repo:

```
~/.gw/
├── active              # current session branch
├── config              # API keys (git-ignored)
├── hooks/              # lifecycle hooks
└── sessions/<branch>/
    ├── log             # append-only structured log
    ├── focus           # last focus note
    └── park            # last park note
```

## License

MIT

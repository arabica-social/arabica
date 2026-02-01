# Using jj (Jujutsu) with Cells

This guide explains how to use jj (Jujutsu) version control with the cells work management system. Cells integrates with jj to provide isolated workspaces and change tracking for each unit of work.

## Quick Reference

```bash
# Check current change
jj log -r @                     # Show current change
jj status                       # Show working copy status
jj diff                         # Show uncommitted changes

# Describe your work
jj desc -m "feat: your message" # Set change description

# View history
jj log                          # Show commit log
jj log -r 'all()'               # Show all changes including hidden

# Working with changes
jj new                          # Create new change on top of current
jj squash                       # Squash current change into parent
jj edit <change-id>             # Edit an existing change

# Rebasing
jj rebase -r @ -d <dest>        # Rebase current change
jj rebase -s <source> -d <dest> # Rebase a subtree
```

## How Cells Uses jj

### Workspace Isolation

When you run `cells run <id>`, cells:

1. Creates a dedicated jj workspace at `.worktrees/cell-<id>`
2. Each workspace has its own working copy
3. Sets the change description using conventional commits
4. Binds the cell to the workspace's change-id

This allows multiple agents to work on different cells simultaneously without conflicts.

### Automatic Description

`cells run` automatically sets your jj change description using conventional commit format:

```
<type>: <cell title>

<cell description>

Cell: <cell-id>
```

The commit type is inferred from:
- Cell labels (e.g., `bug` → `fix`, `feature` → `feat`)
- Title keywords (e.g., "Fix ..." → `fix`, "Add ..." → `feat`)

### Change Binding

Cells tracks which jj change corresponds to each cell via the `change_id` field. This provides:
- Traceability between tasks and code
- Ability to verify work was committed
- Integration with `cells sync` to check for orphaned cells

## Conventional Commits

Use conventional commit format for all change descriptions:

```
<type>[optional scope]: <description>

[optional body]

[optional footer(s)]
```

### Types

| Type | When to Use |
|------|-------------|
| `feat` | New feature or capability |
| `fix` | Bug fix |
| `docs` | Documentation only |
| `style` | Formatting, no code change |
| `refactor` | Code change that neither fixes nor adds |
| `perf` | Performance improvement |
| `test` | Adding or updating tests |
| `chore` | Maintenance tasks |
| `ci` | CI/CD changes |
| `build` | Build system changes |

### Examples

```bash
# Feature
jj desc -m "feat: add user authentication

Implement JWT-based authentication with refresh tokens.

Cell: 01KG8Y8JVT37"

# Bug fix
jj desc -m "fix: prevent null pointer in user lookup

Handle case where user record is deleted mid-session.

Cell: 01KG8Y8JVT38"

# Documentation
jj desc -m "docs: add API endpoint documentation

Cell: 01KG8Y8JVT39"
```

## Common Workflows

### Starting Work on a Cell

```bash
# Let cells create the workspace and set description
cells run <cell-id>

# You're now in the workspace with description set
# Just start coding!
```

### Making Progress

```bash
# Check what you've changed
jj status
jj diff

# jj auto-commits on every change, so just keep working
# Optionally update description as you go
jj desc -m "feat: updated description of progress"
```

### Creating Checkpoints

```bash
# Create a new change for the next phase of work
jj new -m "feat: continue implementation"

# Previous work is now in parent change
```

### Squashing Work

When you have multiple changes for one cell:

```bash
# View your changes
jj log

# Squash current into parent
jj squash

# Or squash with a new message
jj squash -m "feat: complete feature implementation"
```

### Rebasing onto Main

When your work is done and needs to merge:

```bash
# First, identify the main branch
jj log -r 'trunk()'

# Rebase your changes onto trunk
jj rebase -r @ -d trunk()

# Or rebase a whole subtree
jj rebase -s <first-change> -d trunk()
```

### Handling Conflicts

```bash
# If rebase causes conflicts
jj status                       # Shows conflicted files

# Option 1: Resolve in editor
# Edit the conflicted files, remove conflict markers

# Option 2: Use jj resolve
jj resolve                      # Interactive resolution

# After resolving
jj squash                       # Squash resolution into conflicted change
```

## Workspace Management

### Listing Workspaces

```bash
# Via cells
cells workspace list

# Via jj
jj workspace list
```

### Cleaning Up

```bash
# Clean workspaces for completed cells
cells workspace clean

# Clean a specific cell's workspace
cells workspace clean <cell-id>

# Clean orphaned workspaces
cells workspace clean-orphaned

# Or manually via jj
jj workspace forget <workspace-name>
rm -rf .worktrees/<workspace-name>
```

### Switching Workspaces

```bash
# Just cd to the workspace
cd .worktrees/cell-01KG8Y8J

# Or start fresh with cells run
cells run <cell-id>
```

## Best Practices

### 1. Let Cells Manage Workspaces

Don't manually create workspaces for cells. Use `cells run` to ensure:
- Proper naming convention
- Change binding
- Description setting
- Cell status updates

### 2. Keep Changes Atomic

Each cell should correspond to one logical change:
- If you create multiple changes, squash before completing
- Use `jj new` for work-in-progress checkpoints, then squash

### 3. Write Good Descriptions

```bash
# Bad
jj desc -m "updates"

# Good
jj desc -m "feat: add retry logic to API client

Implement exponential backoff with jitter for transient failures.
Max 3 retries with 1s initial delay.

Cell: 01KG8Y8JVT37"
```

### 4. Rebase Before Completing

Before marking a cell complete:

```bash
# Ensure your change is based on latest
jj rebase -r @ -d trunk()

# Verify it applies cleanly
jj status

# Then complete the cell
cells complete <cell-id> --cleanup
```

### 5. Use `jj log` Liberally

```bash
# See what you're working on
jj log -r @

# See the full picture
jj log -r '::@'

# See all changes in all workspaces
jj log -r 'all()'
```

## Troubleshooting

### "not a jj repository"

You're not in a jj-managed directory. Either:
- Navigate to your repo root
- Use `cells run` which handles workspace navigation

### "workspace already exists"

The workspace already exists. Use:
```bash
cells run <cell-id>  # Reuses existing workspace
```

### "conflicting changes"

After a rebase:
```bash
jj status                 # See conflicts
# Edit files to resolve
jj squash                 # Apply resolution
```

### "change not found"

The change-id in the cell no longer exists:
```bash
cells sync                # Check all bindings
cells unbind <cell-id>    # Clear stale binding
```

## jj vs git

| Operation | jj | git |
|-----------|-----|-----|
| Commit | Automatic | `git commit` |
| Describe | `jj desc -m` | `git commit --amend` |
| New change | `jj new` | `git commit` |
| Rebase | `jj rebase` | `git rebase` |
| Squash | `jj squash` | `git rebase -i` |
| View log | `jj log` | `git log` |
| View diff | `jj diff` | `git diff` |

Key differences:
- jj has no staging area - all changes are always "staged"
- jj auto-commits on every file change
- jj changes are mutable until pushed
- jj supports multiple concurrent workspaces natively

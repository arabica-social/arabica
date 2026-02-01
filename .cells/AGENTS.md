# Cells: Agent Guide

This guide explains how to use the cells work management system. Cells are atomic, dependency-aware units of work designed for coordination between humans and AI agents.

## Quick Reference

```bash
# View available work
cells list                      # List active cells
cells list --status open        # Show only unclaimed cells
cells show <id>                 # View cell details

# Claim and work on a cell
cells claim <id>                # Claim a cell (assigns to you)
cells start <id>                # Mark as in_progress
cells complete <id>             # Mark as completed

# Start an agent session for a cell
cells run <id>                  # Create workspace and launch agent
cells run <id> --agent opencode # Use specific agent harness

# Dependencies
cells dep tree <id>             # View dependency tree
cells dep add <id> --needs <x>  # Add dependency
```

## Core Concepts

### What is a Cell?

A cell is a discrete unit of work with:
- **Title**: Brief description of the task
- **Description**: Detailed requirements and acceptance criteria
- **Status**: Current state (open, claimed, in_progress, blocked, completed, cancelled)
- **Priority**: Urgency level (critical, high, normal, low)
- **Dependencies**: Other cells that must complete first
- **Change binding**: Optional link to a jj/git change

### Status Lifecycle

```
open → claimed → in_progress → completed
  │       │           │
  │       │           └──→ blocked ──→ (back to claimed/in_progress when unblocked)
  │       │
  └───────┴──────────────→ cancelled
```

| Status | Meaning |
|--------|---------|
| open | Available for anyone to claim |
| claimed | Assigned but work not started |
| in_progress | Actively being worked on |
| blocked | Waiting on dependencies |
| completed | Work finished successfully |
| cancelled | Work abandoned |

## Agent Workflow

### 1. Finding Work

When starting a work session, find available cells:

```bash
# List all active (non-completed, non-cancelled) cells
cells list

# Find unclaimed work
cells list --status open

# Find work assigned to you
cells list --mine

# See high-priority items
cells list --priority critical
cells list --priority high
```

### 2. Starting an Agent Session

The easiest way to work on a cell is to use `cells run`:

```bash
cells run <cell-id>
```

This command:
1. Creates a dedicated jj workspace for the cell
2. Claims the cell and binds it to the workspace's change
3. Launches your configured agent (claude-code, opencode, etc.) in the workspace
4. Passes the cell's context as the initial prompt

Options:
- `--agent <name>`: Use a specific agent harness (e.g., `--agent opencode`)
- `--no-claim`: Skip claiming/binding (for continuing previous work)

The agent runs in your current terminal. When it exits, you're back in your original directory.

### 3. Manual Workflow (Alternative)

If you prefer to work without `cells run`:

```bash
# Claim the cell
cells claim <cell-id>

# Start work
cells start <cell-id>

# ... do your work ...

# Complete the cell
cells complete <cell-id>
```

### 4. Checking Dependencies

Before working, verify the cell isn't blocked:

```bash
cells show <cell-id>        # Check "Blocked by" field
cells dep tree <cell-id>    # View full dependency tree
```

If blocked, either:
- Work on the blocking cells first
- Wait for another agent to complete them
- Discuss with the manager agent about reprioritization

### 5. Adding Notes

Document progress, blockers, or important decisions:

```bash
cells note <cell-id> "Implemented the basic structure, need to add tests"
cells note <cell-id> "Blocked: waiting for API design decision"
```

Notes create a timestamped log visible to all agents and humans.

### 6. Completing Work

When the task is done:

```bash
cells complete <cell-id>
```

**Before completing, verify**:
- All acceptance criteria in the description are met
- Tests pass (if applicable)
- Code is committed (if applicable)

### 7. Handling Blockers

If you discover a dependency that wasn't captured:

```bash
# Add the dependency
cells dep add <your-cell> --needs <blocking-cell>
```

This automatically transitions your cell to `blocked` status.

When the blocking cell completes, you can resume:

```bash
cells unblock <cell-id>
cells start <cell-id>
```

## Multi-Agent Coordination

### Communication Protocol

Agents communicate through cells, not direct messages:

1. **Manager → Worker**: Creates cells, sets priorities, assigns work
2. **Worker → Manager**: Updates status, adds notes, completes cells
3. **Worker → Worker**: Dependencies between cells

### Avoiding Conflicts

- **Always claim before working**: Prevents duplicate work
- **Use short IDs**: First 6-8 characters usually sufficient (e.g., `01KG8V76`)
- **Check status before operations**: Another agent may have modified the cell

### Creating New Cells

If you discover work that should be tracked:

```bash
cells new "Implement error handling" \
  -d "Add proper error handling to the API endpoints" \
  -p high \
  -l backend -l api
```

For work that depends on your current task:

```bash
# Create the new cell
cells new "Add unit tests for error handling"

# Make it depend on your current work
cells dep add <new-cell-id> --needs <your-cell-id>
```

## jj/Git Integration

Cells can be bound to version control changes:

```bash
# Bind to current jj change
cells bind <cell-id>

# Bind to specific change
cells bind <cell-id> --change <change-id>

# Remove binding
cells unbind <cell-id>

# Verify all bindings are valid
cells sync
```

**Best Practice**: Bind a cell to its change before starting work. This creates traceability between tasks and code.

When using `cells run`, the cell is automatically bound to the workspace's change.

## Agent Harness Configuration

The agent harness configuration lives in `.cells/config.json`:

```json
{
  "workspaces_dir": ".worktrees",
  "agent": {
    "default": "claude-code",
    "harnesses": {
      "claude-code": {
        "command": "claude",
        "args": ["-p", "{prompt}"]
      },
      "opencode": {
        "command": "opencode",
        "args": ["-p", "{prompt}"]
      }
    }
  }
}
```

The `{prompt}` placeholder is replaced with the cell's context (title, description, notes).

## Priority Levels

| Priority | Use When |
|----------|----------|
| critical | Production issues, security vulnerabilities, blocking all other work |
| high | Important features, significant bugs, time-sensitive work |
| normal | Standard development tasks (default) |
| low | Nice-to-haves, minor improvements, tech debt |

Cells are sorted by priority in list output. Always check for critical/high priority work first.

## Error Handling

### Common Issues

**"invalid transition"**: Check the current status. Some transitions aren't allowed:
- Can't claim an in_progress cell
- Can't complete a blocked cell
- Can't modify completed/cancelled cells

**"cell not found"**: The ID prefix might be ambiguous. Use more characters.

**"ambiguous ID prefix"**: Multiple cells match. Use a longer prefix or full ID.

### Recovery

If something goes wrong:

```bash
# View current state
cells show <cell-id>

# Cancel and start fresh
cells cancel <cell-id>
cells new "Retry: <original title>"
```

## Best Practices

1. **Atomic tasks**: Keep cells small and focused. One clear objective per cell.

2. **Clear descriptions**: Include acceptance criteria. Another agent should understand what "done" means.

3. **Update status promptly**: Don't leave cells in stale states. Other agents rely on accurate status.

4. **Document decisions**: Use notes liberally. Future agents (or humans) will thank you.

5. **Respect dependencies**: Don't work around blocked status. The dependency exists for a reason.

6. **Release what you can't finish**: If you're stuck or need to stop, release the cell so others can help.

7. **Check before claiming**: Verify the cell is actually open and unblocked.

8. **Use `cells run` for isolated work**: Each cell gets its own workspace, preventing conflicts with other work.

## File Locations

- `.cells/cells.jsonl` - All cell data (JSONL format)
- `.cells/config.json` - Repository configuration
- `.cells/AGENTS.md` - This guide

The `.cells/` directory should be version controlled so all agents share the same state.

## Workspaces

When using `cells run`, workspaces are created at:

```
<repo>/
├── .cells/              # Cell data and config
├── .worktrees/          # Default workspaces directory
│   ├── cell-01HX.../    # Workspace for cell 01HX...
│   └── cell-01HY.../    # Workspace for cell 01HY...
└── (main workspace)
```

Each workspace is an independent jj workspace with its own working copy. This allows multiple agents to work on different cells simultaneously without interference.

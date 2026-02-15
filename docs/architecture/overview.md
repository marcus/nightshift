# Nightshift Architecture Overview

## Introduction
Nightshift is an autonomous, budget-aware software maintenance tool designed to run during off-peak hours (overnight). It leverages Large Language Models (LLMs) to perform routine tasks such as finding dead code, fixing documentation drift, closing test gaps, and auditing security issues.

The primary philosophy of Nightshift is **Zero Risk**: all changes are submitted as Pull Requests or branches, never directly to the primary branch, allowing human developers to review and merge only what they trust.

## Scope and Features
- **Autonomous Maintenance**: Automatically identifies and fixes common technical debt.
- **Budget-Awareness**: Tracks token usage against daily/weekly allotments to avoid unexpected costs.
- **Multi-Project Support**: Can manage multiple repositories from a single configuration.
- **Agentic Workflow**: Uses a Plan-Implement-Review loop to ensure high-quality outputs.
- **Extensible Architecture**: Supports multiple LLM providers (Claude, Codex) and various task types.

## Use Cases
1. **Dead Code Elimination**: Identify and remove unused functions, variables, or imports.
2. **Documentation Synchronization**: Update READMEs or KDocs when implementation changes.
3. **Security Patching**: Automatically apply fixes for known vulnerabilities found in audits.
4. **Test Gap Filling**: Generate unit tests for uncovered code paths.
5. **Bus Factor Analysis**: Analyze commit history to identify critical code areas owned by single individuals.

## High-Level Architecture
Nightshift is built as a modular Go application. The core components are decoupled to allow for flexibility in AI agents, budget tracking, and task sourcing.

### Component Diagram
![Component Diagram](components.puml)

### Key Modules:
- **CLI (`cmd/nightshift`)**: The user interface for configuration, manual runs, and status monitoring.
- **Orchestrator (`internal/orchestrator`)**: The "brain" that coordinates the AI agents through the execution loop.
- **Scheduler (`internal/scheduler`)**: Manages when tasks are executed, supporting cron, intervals, and time windows.
- **Budget Manager (`internal/budget`)**: Ensures the tool stays within resource limits.
- **Task System (`internal/tasks`)**: Manages the queue of work to be done, sourced from GitHub, local files, or internal analyzers.
- **Agents (`internal/agents`)**: Wrappers around LLM APIs that provide a standardized execution interface.

## Configuration
Nightshift uses a flexible configuration system that allows for global defaults and per-project overrides.

Detailed configuration documentation can be found in [Configuration Overview](configuration.md).

### Key Configurable Areas:
- **Scheduling**: When and how often the tool runs.
- **Budget**: Daily/weekly token limits and usage modes.
- **Projects**: Which repositories to manage and their relative priorities.
- **Tasks**: Selection of maintenance tasks (finding dead code, updating docs, etc.).
- **Providers**: AI engine settings for Claude and Codex.
- **Integrations**: External data sources like GitHub Issues.

## Scheduling
Scheduling is managed by the `internal/scheduler` package, which provides flexible execution triggers and safety constraints.

### Scheduling Modes:
- **Cron**: Supports standard cron expressions (e.g., `0 2 * * *` for 2 AM daily) via the `robfig/cron/v3` library.
- **Interval**: Supports fixed duration intervals (e.g., `1h`, `30m`).
- **Time Window**: An optional constraint that limits execution to specific hours (e.g., `22:00` to `06:00`). This is used to ensure "nightshift" truly runs overnight, even if triggered by an interval.

### Scheduling Logic:
![Scheduling Logic](scheduling.puml)

1.  **Trigger**: Either a cron tick or an interval timer fires.
2.  **Window Check**: The scheduler checks if the current time falls within the allowed `Window` (if configured).
3.  **Job Execution**: If inside the window (or no window is set), the scheduler triggers the Orchestrator's run loop.
4.  **Next Run Calculation**: The scheduler calculates and updates the next scheduled run time.

## Business Logic: The Plan-Implement-Review Loop
The Orchestrator follows a rigorous process for every task to ensure reliability.

### Sequence Diagram
![Task Execution Sequence](task_execution_sequence.puml)

1. **Planning**: An agent analyzes the task and the codebase to create a step-by-step execution plan and identify relevant files.
2. **Implementation**: The agent modifies the files in a local sandbox/workspace based on the plan.
3. **Review**: A separate review step (often using a different prompt or model perspective) verifies the changes against the original task and project standards.
4. **Iteration**: If the review fails, the agent receives feedback and attempts to fix the implementation (up to a configurable `MaxIterations`).
5. **Finalization**: If successful, the Orchestrator pushes a branch and creates a Pull Request via the GitHub integration.

## Internal Structure
The internal class and interface relationships emphasize composability.

### Internal Structure Diagram
![Internal Structure](internal_structure.puml)

- **`Agent` Interface**: Allows swapping between Claude, Codex, or other future providers.
- **`BudgetSource` Interface**: Enables different ways of measuring usage (API-based, local tracking, or external stats).
- **`Task` & `Queue`**: Decouples task discovery from task execution.

## Git & Repository Management
Nightshift follows a **Local-First** approach to repository management. It does not autonomously clone or checkout repositories from remote sources. Instead, it operates on existing local clones configured by the user.

### Repository Interaction:
- **Project Discovery**: Nightshift identifies targets through explicit paths or glob patterns in the `projects` configuration.
- **Local Operations**: All analysis and modifications happen directly within the configured local repository paths.
- **Sandboxing**: Changes are implemented in a sandboxed environment to prevent accidental side effects on the primary branch.
- **GitHub Integration**: Uses the `gh` CLI to interact with GitHub-specific features like issue tracking and Pull Request management.
- **Zero Risk**: Modifications are submitted as new branches and Pull Requests, ensuring the primary branch remains untouched and requires human approval for merging.

## Architectural Principles
1. **Idempotency**: Runs should be repeatable and safe to restart.
2. **Transparency**: Detailed logs and metadata are injected into PRs for full traceability.
3. **Safety First**: Uses sandboxed environments and Git branches to prevent accidental data loss or corruption.
4. **Economic Efficiency**: Prioritizes tasks based on cost-benefit tiers and remaining budget.

# AI Agents Guide

Welcome to the autonomous development workspace for the **Go Modulith Template**. This project is optimized for collaboration with high-performance AI agents.

## Meet Your Agent: Antigravity

**Antigravity** is your lead agentic coding assistant. Designed for advanced engineering tasks, Antigravity excels at:

- **Complex Refactoring**: Handling multi-module changes with precision.
- **Architectural Design**: Maintaining the integrity of the Modulith architecture.
- **Visual Excellence**: Delivering premium, high-fidelity web interfaces (when applicable).
- **Proactive Problem Solving**: Researching issues deeply before proposing implementation plans.

## Collaboration Workflow

To get the most out of your AI agents, follow this standard workflow:

### 1. The Request
Be specific about the desired outcome. You can reference specific files, modules, or documentation using `@` symbols in the chat.

### 2. Research & Discovery
The agent will explore the codebase, run tests, and analyze dependencies to understand the full context of your request.

### 3. Implementation Plan
For non-trivial tasks, the agent will present an **Implementation Plan**. 
> [!IMPORTANT]
> Always review the plan carefully before approving. This ensures alignment on the technical approach and prevents regressions.

### 4. Execution & Tracking
Once approved, the agent will execute the plan, tracking progress via a `task.md` document.

### 5. Verification
The agent will verify all changes using the project's verification suite (`just pre-commit`, `just test-all`) before finishing the turn.

## Communication Tips

- **Use Just Commands**: Reference `just` commands for specific tasks.
- **Direct Feedback**: If an implementation detail doesn't meet your expectations, provide direct feedback early in the planning phase.
- **Context is King**: The more context you provide (errors, logs, screenshots), the more effective the agent will be.

---
*Optimized for productivity. Built for excellence.*

# agk-orchestrator Project Charter

## Project Intent
Provide an easy way to create different personas/agents with different models and tools depending on the use case, using a TOML configuration format. The system should enable rapid agent creation and deployment for various operational scenarios.

## Governance Principles

### Configuration-First Approach
- All agent definitions and workflows are configured in TOML files
- Configuration files are validated for required fields (provider, model, base_url) and referential integrity
- No direct code changes to agent configuration files are allowed without going through the PR process

### Quality & Reliability
- All agent configurations must pass validation checks before deployment
- Agents must respond within 5 seconds for simple queries and within 10 seconds for complex tool-calling workflows
- System/node downtime is the primary risk with automated monitoring and alerting to detect and respond to failures

### Change Management
- All agent changes must be reviewed and approved through the standard PR process with conventional commits
- Changes to agent configurations must be documented in the CHANGELOG
- Changes must be proposed through a PR and reviewed by at least one other team member

### Risk Mitigation
- System/node downtime is the primary risk
- In the case of system/node downtime, the system should automatically trigger a recovery process and notify the operations team
- Automated monitoring and alerting is in place to detect and respond to node failures

### Documentation
- All agent configurations must be documented in the AGENTS.md file
- Changes to agent configurations must be reflected in the CHANGELOG
- All agent changes must be documented in the CHANGELOG

## Operational Workflow
1. Create agent configuration in TOML format
2. Submit PR with configuration changes
3. Review and approve changes through standard PR process
4. Changes are automatically validated and deployed through Bazel build and test process
5. Changes are documented in CHANGELOG

## Tooling & Dependencies
- Uses AgenticGoKit v1beta with OpenAI-compatible adapters for MLX models
- Agent configurations can use prompt templates with {agent}, {model}, {provider}, {base_url} variables
- Agents can be configured to use tool-calling loops with max_iterations and max_concurrent settings
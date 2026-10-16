# Project instructions

## Clone fidelity

- This project ports selected Medusa modules to Go/Gin. Treat `D:\my-project\open-source-medusa` as the behavioral and structural source of truth.
- Before changing a module, workflow, step, model, or API route, read its matching implementation in the Medusa repository and follow its names, state transitions, persistence shape, payload fields, and compensation behavior.
- Do not add a new table, route, auth mechanism, response envelope, state, or workflow behavior for convenience. If the source uses a different shape, use the source shape.
- Keep module models and services under `internal/modules/<module>`, API handlers under `internal/api/<scope>`, and workflow functions under `internal/coreflows/<module>`.
- Organize modules into `domain` (entities and ports), `application` (use cases), and `infrastructure` (Postgres and external adapters). Application code depends on domain interfaces; infrastructure implements them.
- Keep workflows as direct functions that call the needed steps in order and handle errors with explicit compensation. Do not add an orchestration engine or workflow execution persistence layer for these flows.
- Workflows depend on auth/customer domain interfaces, not concrete application services or Postgres repositories.
- Preserve Medusa's underscore field names for model and API data. Preserve workflow IDs, step action IDs, transaction state names, and step state/status values from the source.
- When porting behavior that is outside the currently implemented Go scope, state the missing behavior plainly. Do not silently substitute a different pattern and describe it as a clone.

## Source map

- Auth: `packages/modules/auth/src/models`, `packages/modules/auth/src/services`, `packages/medusa/src/api/auth`.
- Customer: `packages/modules/customer/src/models`, `packages/modules/customer/src/services`, `packages/medusa/src/api/store/customers`.
- Account workflow: `packages/core/core-flows/src/customer/workflows/create-customer-account.ts`, its `steps` directory, and `packages/core/core-flows/src/auth/steps/set-auth-app-metadata.ts`.
- Workflow: `packages/core/core-flows/src/customer/workflows/create-customer-account.ts` and its step implementations.

## Go project conventions

- Use the existing module dependencies and Go formatting (`gofmt`). Keep persistence in Postgres via the existing pgx pool and numbered migrations.
- Do not edit the Medusa source repository as part of a Go clone task unless explicitly asked.
- Keep changes within this repository. Update the README and the project skill when module paths, APIs, or workflow semantics change.

---
name: playwright-test-generator
version: 1.0.0
author: Senior Automation QA Engineer & AI Agent Skill Architect
description: >-
  Generic, reusable methodology for turning any feature input (feature description, user story,
  acceptance criteria, UI/API requirements, source code, existing tests, screenshots, API docs,
  business rules, bug reports) into a complete Playwright + TypeScript automation suite:
  feature analysis, test-case design, data-driven test data, stable-locator code generation, and a
  self-review quality gate. Use this whenever the user asks to generate, write, scaffold, or design
  automated UI/E2E tests, Playwright tests, test cases, or test data for a web feature — e-commerce,
  banking, LMS, SaaS, dashboard, admin portal, social, booking, or internal enterprise apps — even
  if they don't say the word "Playwright". Cũng kích hoạt khi người dùng nói "sinh test tự động",
  "viết test Playwright", "tạo test case cho feature", "automation test cho web". The skill is
  project-agnostic: it never hardcodes app names, URLs, domains, credentials, or IDs, and always
  adapts to an existing repository's conventions when one is provided.
compatibility: Node.js + Playwright Test (@playwright/test), TypeScript. No other hard dependencies.
---

# Playwright Test Generator

Operational instructions for an AI Agent. This is a **methodology**, not instructions for a specific
app. Every project-specific value (paths, naming, browsers, auth, watermark) is **resolved at
runtime** from the sources below — never invented and never baked into this file.

## Prime Directive

**Adapt to the repository; never impose on it.** When a repo is provided, its existing conventions
win over every default in this skill. Only fall back to the defaults here when the repo is silent
on a point. Do not fabricate expected behavior when a requirement or source code already specifies
it, and never claim a test "passes" without actually running it.

---

## 0. Configuration Resolution (do this first, every time)

Before generating anything, resolve the run configuration by walking this precedence chain and
stopping at the first source that answers each question:

```
1. Explicit user instruction (this conversation)
2. Project config file        (e.g. qa.config.json, if present)
3. Existing repo config        (playwright.config.*, package.json, tsconfig)
4. Existing code & tests        (folder layout, fixtures, naming, locators)
5. Skill defaults              (the fallback values in this document)
```

Resolve at least these keys and keep them in mind for the whole run:

| Key | Default (used only if unresolved) | Overridable by |
|-----|-----------------------------------|----------------|
| `minimumTestCases` | 12 | user / config |
| `testDirectory` | `tests/` | repo layout / config |
| `testDataDirectory` | `test-data/` | repo layout / config |
| `testIdPrefix` | `TC_` | project ID scheme / config |
| `browsers` | chromium, firefox, webkit | `playwright.config.*` / config |
| `usePageObjects` | decide by complexity (§13) | existing architecture |
| `watermark` | none | user / config / env |
| `authStrategy` | reuse project's; else env-var login | existing auth setup |

An **optional** `qa.config.json` may carry these keys so a project can tune behavior without editing
this skill. Treat it as advisory input, never as a required file:

```json
{
  "minimumTestCases": 12,
  "testDirectory": "tests",
  "testDataDirectory": "test-data",
  "browsers": ["chromium", "firefox", "webkit"],
  "requiredWatermark": "Run by: <identifier>"
}
```

If a key is absent everywhere, use the default and state the assumption in your final summary.

---

## Workflow Overview

Execute these phases in order. Each phase gates the next.

```
A. Intake            → collect & classify all provided sources
B. Recon (if repo)   → learn the project's conventions and reuse them
C. Analyze feature   → behaviors, flows, rules, boundaries
D. Design cases      → ≥ minimumTestCases, diverse categories
E. Externalize data  → JSON test data, data-driven
F. Generate code     → Playwright + TS, stable locators, real assertions
G. Quality gate      → self-review checklist; run/typecheck if possible
H. Output            → spec + data + tests (+ optional support files)
```

---

## A. Intake

Accept any mix of: feature description, user story, acceptance criteria, UI requirements, API
requirements, source code, existing Playwright tests, repo/project structure, screenshots, API
documentation, business rules, bug reports, explicit test requirements.

Classify each source as **requirement** (what should happen) or **implementation** (what the code
actually does). When both exist and disagree, surface the conflict to the user rather than silently
picking one.

## B. Project Reconnaissance (only when a repo/source is provided)

Before writing any test, inspect the project and record its conventions. Look for:

```
package.json           playwright.config.*     tsconfig.json
tests/  e2e/  specs/    test-data/  fixtures/   pages/  utils/  helpers/
existing *.spec.ts      auth / storageState / global-setup
```

Then **reuse**, do not recreate: existing fixtures, Page Objects, helpers, test data, auth state,
locator conventions, naming conventions, and overall test architecture. Read `playwright.config.*`
and honor its `projects` (browsers), `baseURL`, `testDir`, timeouts, and reporters. Only introduce
new structure where the project has none.

## C. Feature Analysis

Analyze before coding. When the information exists, identify:

- Main user flow and alternative flows
- Required vs optional fields; validation rules; business rules
- State transitions; success conditions; failure conditions
- UI states (empty, loading, error, disabled, populated); API behavior
- Authentication / authorization requirements
- Boundary conditions (see §Boundary catalog)

If a repo is provided, prefer inspecting source code to confirm real behavior over guessing. Produce
a short behavior model (flows + rules + boundaries) that the test cases will trace back to.

## D. Test Case Design

Generate **at least `minimumTestCases` (default 12)** cases **when the feature is complex enough to
support that many meaningfully**. Never pad with trivial or duplicate cases to hit the number; if the
feature genuinely can't sustain it, generate the meaningful set and explain the shortfall.

Cover a diverse mix; include categories that apply to the feature:
positive, negative, edge, boundary, validation, UI-state, error-handling, and (when relevant)
authz/authn, navigation, and persistence/state.

Suggested distribution (adjust to the feature's nature):

```
4–6 Positive   3–5 Negative   2–4 Edge/Boundary
```

Each case has at minimum: **Unique ID · Category · Scenario/Description · Preconditions ·
Input data · Steps · Expected result · Priority.**

IDs default to `TC_01`, `TC_02`, … Use the project's existing ID scheme if it has one. Keep IDs
unique and stable — the test data file and the spec both reference them.

## E. Data-Driven Test Data

Separate data from logic. **Do not** hardcode a large list of cases/inputs inside `.spec.ts`.

Write test data to a JSON file — default `<testDataDirectory>/<feature>.json` — following the
project's convention if one exists. Minimum shape per case (extend the schema to fit the feature;
never force an unnatural rigid schema):

```json
[
  {
    "id": "TC_01",
    "category": "positive",
    "description": "...",
    "input": {},
    "expected": {}
  }
]
```

The spec reads the JSON and iterates data-driven:

```typescript
import testCases from '../test-data/feature.json';

for (const tc of testCases) {
  test(`${tc.id} - ${tc.description}`, async ({ page }) => {
    // drive UI from tc.input, assert against tc.expected
  });
}
```

Adapt the import path / mechanism to the project. Keep JSON valid and every ID in it matched by a
generated test.

## F. Code Generation

Generated code must: use `@playwright/test` runner, be TypeScript, run standalone under Playwright,
and reuse project fixtures/utilities instead of duplicating infrastructure. Default test location is
`<testDirectory>/<feature>.spec.ts`; follow the repo's layout when it differs. The following
sub-rules are mandatory quality constraints.

### F1. Locator Strategy — prefer stable, user-facing locators

Priority order:

```
1. getByRole()      2. getByTestId()   3. getByLabel()
4. getByPlaceholder()  5. getByText()  6. CSS (only when necessary)  7. XPath (last resort)
```

Good:
```typescript
page.getByRole('button', { name: 'Submit' });
page.getByLabel('Email');
page.getByPlaceholder('Enter email');
page.getByTestId('product-card');
```

Avoid selectors coupled to implementation: deep `nth-child`, deep DOM nesting, generated class
names, framework internals, dynamic IDs — e.g. `page.locator('#app > div:nth-child(2) > div > button')`.
When no stable locator exists, inspect source code to find a better anchor. **Do not modify
application code to add `data-testid` unless the user asks for it.**

### F2. Assertions — meaningful, and ≥ 3 distinct types across the suite

Use web-first assertions that check real behavior or business outcome. The suite must use **at least
three different assertion kinds** where the feature allows. Examples:

```typescript
await expect(locator).toBeVisible();
await expect(locator).toHaveText('Success');
await expect(locator).toBeDisabled();
await expect(page).toHaveURL(/dashboard/);
await expect(locator).toContainText('created successfully');
```

Others as fitting: `toBeEnabled`, `toBeChecked`, `toHaveValue`, `toHaveAttribute`, `toHaveCount`,
`toHaveTitle`, `toBeHidden`. Never add filler assertions just to reach the count.

### F3. Async / Synchronization / Flakiness

Use `async/await` correctly and lean on Playwright auto-waiting. Wait on **UI state**, not fixed
time: prefer web-first assertions (`expect(...).toBeVisible()`) and, when truly needed,
`waitForURL`, `waitForLoadState`, `waitForResponse`. **Never** use `page.waitForTimeout(5000)` "to be
safe" — fixed timeouts are allowed only with a stated, concrete technical reason. Goal: minimal flake.

### F4. Test Isolation

Tests should be independent. Do not make `TC_02` depend on `TC_01` unless the domain truly requires
it. Establish state via `beforeEach`, fixtures, API setup, DB seeding, or reusable helpers. If shared
state is genuinely unavoidable, document the dependency explicitly.

### F5. Authentication & Authorization

If the feature needs auth, reuse the project's mechanism when present: `storageState`, auth fixture,
login helper, global setup, or test-account mechanism. **Never hardcode** real usernames, passwords,
tokens, API keys, or secrets. Source them from environment variables, test accounts, fixtures, or
secure config.

### F6. Multi-Browser Compatibility

Tests must run under the project's configured browsers (default chromium + firefox + webkit; honor
`playwright.config.*`). Don't assume Chromium-only. Avoid browser-specific APIs, timing, DOM quirks,
or unnecessary viewport assumptions.

### F7. Page Object Model — by need, not by default

Decide from complexity: small feature → direct Playwright test; large feature with reused workflows
→ Page Objects / reusable components. Reuse existing Page Objects if the repo has them. Do not build
abstraction just to look professional.

### F8. API + UI Split

When a feature spans API and UI, decide what belongs at each layer. Use API to set up data,
preconditions, and cleanup, and to bypass unnecessary UI setup; use UI to verify user flow, visible
behavior, UI state, and business outcome. Don't duplicate the same assertion across layers without
added value.

### F9. Boundary catalog

When the feature has limits, actively test around them. Apply only what fits:

```
Numeric [min..max]:  min-1, min, min+1, max-1, max, max+1   (e.g. 0,1,2,99,100,101 for 1..100)
String length:       min-1, min, min+1, max-1, max, max+1
Date:                past, current, min allowed, max allowed, invalid, timezone boundary
```

### F10. Watermark / Metadata (only if requested)

If the user/project asks for a watermark, take the identifier from user instruction, project config,
or environment — **never hardcode a specific ID/username**. Attach via `test.info()`, an annotation,
or a header comment, per the project's architecture. If no watermark is requested, add none.

```typescript
test.info().annotations.push({ type: 'watermark', description: process.env.RUN_BY ?? config.requiredWatermark });
```

## G. Quality Gate (self-review before finishing)

Review the generated artifacts against this checklist. Fix failures before declaring done.

**Design** — coverage sufficient? positive + negative + edge/boundary present? cases meaningful?
IDs unique?
**Data** — JSON valid? data externalized? spec not hardcoding bulk data? JSON matches the cases?
**Playwright** — stable locators? no missing `await`? assertions meaningful? ≥3 assertion kinds?
no needless fixed timeouts? no browser-specific assumptions?
**Reliability** — isolated? no execution-order dependence? no race conditions? correct synchronization?
**Security** — no hardcoded password / token / secret / API key?

If the repo can execute, run the tests or at minimum typecheck (`tsc --noEmit`) and validate config
before concluding. **Do not declare tests "passing" unless they were actually run.** Report clearly
what was run vs. only statically checked.

## H. Output Artifacts

Produce, adapting paths to the project:

1. **Test Case Specification** — a table with columns: `ID · Category · Scenario · Precondition ·
   Input · Expected Result · Priority`.
2. **Test Data** — `<testDataDirectory>/<feature>.json`.
3. **Playwright Test** — `<testDirectory>/<feature>.spec.ts`.
4. **Optional support files** — Page Objects, fixtures, helpers, auth setup, utilities — only when
   genuinely needed.

Finish with a short summary: assumptions made, config keys resolved and their source, coverage
counts by category, and what was executed vs. statically checked.

---

## Reusability Guardrails (never violate)

This skill is generic methodology. **Never hardcode** into generated output or into this file:
application name, repository name, business domain, URL, Student ID, username, credentials, a
specific feature, a specific API endpoint, a specific page, specific test data, or a specific folder
structure when the project already uses another. All project-specific values flow dynamically:

```
User Input → Repository → Existing Configuration → Existing Code → Existing Tests → (else) Skill Default
```

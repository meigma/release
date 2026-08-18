# Actions delivery constraints

**Current as of 2026-08-18.**

## Direct answer

Choose **(c) both layered**:

1. **A reusable workflow is the supported default for consumers.** Release publication needs job-scoped permissions, protected environments, multi-job dependencies, matrix fan-out, concurrency, artifacts, and workflow outputs. Only a reusable workflow can package those GitHub-level controls and job boundaries. GitHub explicitly limits a reusable-workflow call job to `uses`, `with`, `secrets`, `strategy`, `needs`, `if`, `concurrency`, and `permissions`; the called workflow can then define ordinary jobs with `runs-on`, environments, steps, and timeouts. [GitHub Docs: supported keywords](https://docs.github.com/en/actions/reference/workflows-and-actions/reusing-workflow-configurations#supported-keywords-for-jobs-that-call-a-reusable-workflow)

2. **Also publish a thin composite action as a per-job adapter to the CLI.** It can standardize CLI installation/version selection, invoke the CLI, map outputs, emit annotations, and accept a locally built executable for dogfooding. It also gives consumers with bespoke orchestration a supported step-level integration. Composite actions work on Linux, macOS, and Windows and can call other actions; Docker actions are Linux-only, while a JavaScript wrapper would add a Node implementation and bundled-distribution surface without gaining job-level capabilities. [GitHub Docs: action types](https://docs.github.com/en/actions/concepts/workflows-and-actions/custom-actions) [GitHub Docs: composite `uses` steps](https://docs.github.com/en/actions/reference/workflows-and-actions/metadata-syntax#runsstepsuses)

3. **Keep the CLI directly runnable and installable, but do not make copied `run:` snippets the primary delivery contract.** Plain steps leave every consumer responsible for reproducing permissions, gates, matrices, job outputs, artifact transfer, and installation verification. That defeats the main value of replacing the existing workflow logic.

The decisive tradeoff is **job orchestration versus step reuse**, not the implementation language of the wrapper. A reusable workflow can represent release policy and job boundaries; no custom action can. A composite action can be reused inside those jobs and in caller-designed jobs; a reusable workflow cannot be inserted as a step.

A layered path of **reusable workflow → composite action → CLI** buys a complete turnkey workflow plus an escape hatch. Its costs are one more public input/output contract, more log nesting, explicit secret forwarding, and the risk of action/CLI version skew. The July 2026 `$/` self-repository syntax removes the internal action-pin skew: an internal `uses: $/path/to/action` resolves to the repository and exact commit of the currently running reusable workflow or action, including when the external caller pinned that workflow to a full SHA. It is GitHub.com-only and requires Actions Runner **2.336.0 or newer**. [GitHub changelog, 2026-07-30](https://github.blog/changelog/2026-07-30-reference-same-repository-actions-with-self-repository-syntax/)

## 1. Capability matrix

Legend:

- **Owns**: the reusable unit can declare the feature itself.
- **Caller**: the caller workflow/job must declare it.
- **No**: the shape cannot represent the feature as a native GitHub Actions construct.

| Capability | Reusable workflow (`workflow_call`) | Composite action | JavaScript action | Docker action | Plain installed-CLI step |
|---|---|---|---|---|---|
| **Declare `permissions`** | **Owns, but cannot elevate.** A call job supports `permissions`; called/nested workflows may maintain or reduce the caller’s `GITHUB_TOKEN` permissions, never increase them. For cross-org OIDC, the external caller must explicitly grant `id-token: write`. [Reuse reference](https://docs.github.com/en/actions/reference/workflows-and-actions/reusing-workflow-configurations#supported-keywords-for-jobs-that-call-a-reusable-workflow) [OIDC reference](https://docs.github.com/en/actions/reference/security/oidc#reusable-workflows) | **Caller.** `action.yml` has no `permissions` key. All child steps receive the caller job’s token permissions. [Metadata syntax](https://docs.github.com/en/actions/reference/workflows-and-actions/metadata-syntax) [Workflow permissions](https://docs.github.com/en/actions/reference/workflows-and-actions/workflow-syntax#permissions) | **Caller.** Same job token; no action-metadata permission declaration. | **Caller.** Same job token; no action-metadata permission declaration. | **Caller.** Set at workflow or job level. |
| **Secrets** | Defines explicit `on.workflow_call.secrets`; caller maps names with `jobs.<id>.secrets`. `secrets: inherit` is available only when caller and callee are in the same organization or enterprise. Secrets pass only to the directly called workflow at each nesting edge. External organizations therefore need explicit secret mapping unless both are in one enterprise. [Reuse workflows](https://docs.github.com/en/actions/how-tos/reuse-automations/reuse-workflows#passing-inputs-and-secrets-to-a-reusable-workflow) | The `secrets` context is deliberately unavailable inside composite actions; the caller must pass every non-`GITHUB_TOKEN` secret as an input or environment variable. [Contexts reference](https://docs.github.com/en/actions/reference/workflows-and-actions/contexts#secrets-context) | Caller passes secrets as `with` inputs or `env`; an action can read only secrets explicitly included by the workflow. [Secrets](https://docs.github.com/en/actions/concepts/security/secrets) | Same as JavaScript; Docker inputs must also be threaded through action metadata/arguments as required. [Metadata syntax](https://docs.github.com/en/actions/reference/workflows-and-actions/metadata-syntax#inputs) | Caller references `${{ secrets.NAME }}` in `env` or command inputs. |
| **OIDC token minting** | The **job** requesting a token requires `id-token: write`. For a reusable workflow outside the caller’s organization/enterprise, the caller workflow or call job must grant it explicitly; the callee cannot elevate the ceiling. The OIDC token contains `job_workflow_ref` and `job_workflow_sha` for reusable-workflow identity. [OIDC reference](https://docs.github.com/en/actions/reference/security/oidc#workflow-permissions-for-the-requesting-the-oidc-token) | Can request the token through a nested action, toolkit call, or `ACTIONS_ID_TOKEN_REQUEST_URL`/`ACTIONS_ID_TOKEN_REQUEST_TOKEN`, but only if the caller job granted `id-token: write`. It cannot grant that permission itself. [OIDC request methods](https://docs.github.com/en/actions/reference/security/oidc#methods-for-requesting-the-oidc-token) | Same; `@actions/core.getIDToken()` is the native JavaScript path. | Same ambient job variables can be consumed inside the container when the caller job grants permission. | CLI can use the ambient request URL/token, but the caller job still grants `id-token: write`. |
| **Is OIDC job-scoped?** | **Yes.** GitHub documents both workflow-level permission, which applies to jobs, and a single-job permission. The token’s claims include job/check-run and reusable-workflow identity. [OIDC reference](https://docs.github.com/en/actions/reference/security/oidc#setting-permissions) | **Yes; inherited from caller job.** | **Yes; inherited from caller job.** | **Yes; inherited from caller job.** | **Yes.** |
| **`environment` and required-reviewer gate** | A normal job inside the called workflow can declare `environment`; the call job itself cannot because `environment` is not a supported call-job keyword. A job referencing an environment must pass protection rules before running or accessing environment secrets. A required-reviewer rule may name up to **6** people/teams, and one approval is sufficient; approval wait is capped at **30 days**. [Supported call keywords](https://docs.github.com/en/actions/reference/workflows-and-actions/reusing-workflow-configurations#supported-keywords-for-jobs-that-call-a-reusable-workflow) [Environment rules](https://docs.github.com/en/actions/how-tos/deploy/configure-and-manage-deployments/manage-environments) [Actions limits](https://docs.github.com/en/actions/reference/limits) | Caller’s ordinary job declares `environment`; the gate completes before the action runs and before environment secrets become available. The action cannot declare a job environment. | Caller job. | Caller job. | Caller job. |
| **Environment-secret caveat** | `on.workflow_call` does not support an `environment` declaration. Environment secrets cannot be passed through that key; if a called job declares an environment, the environment secret wins over a caller-passed secret with the same name. [Reuse workflows warning](https://docs.github.com/en/actions/how-tos/reuse-automations/reuse-workflows#using-inputs-and-secrets-in-a-reusable-workflow) | After the caller job’s environment gate, pass the secret explicitly as an input/env; the composite still has no `secrets` context. | Explicit input/env. | Explicit input/env. | Direct job `secrets` context. |
| **Matrix fan-out** | **Owns.** A call job supports `strategy`; matrix values can be forwarded as workflow inputs. A matrix can generate at most **256 jobs per workflow run**. Matrix reusable-workflow outputs use the last successfully completing invocation that actually set a value. [Matrix reuse](https://docs.github.com/en/actions/how-tos/reuse-automations/reuse-workflows#using-a-matrix-strategy-with-a-reusable-workflow) [Actions limits](https://docs.github.com/en/actions/reference/limits) [Matrix output behavior](https://docs.github.com/en/actions/how-tos/reuse-automations/reuse-workflows#using-outputs-from-a-reusable-workflow) | **Caller.** A matrix expands the caller job; the composite executes once per expanded job. | Caller. | Caller. | Caller. |
| **`concurrency`** | **Owns.** Both the call job and jobs inside the called workflow can use job concurrency; a called workflow can also have workflow-level concurrency. Permissions documentation warns that reusing the same group based on `github.workflow` with `cancel-in-progress: true` in caller and callee can cancel the caller because the callee sees the caller workflow’s name. Default queueing permits one running and one pending member per group; `queue: max` permits up to **100 pending jobs/runs** and cannot be combined with `cancel-in-progress: true`. [Reuse reference](https://docs.github.com/en/actions/reference/workflows-and-actions/reusing-workflow-configurations#supported-keywords-for-jobs-that-call-a-reusable-workflow) [Concurrency docs](https://docs.github.com/en/actions/how-tos/write-workflows/choose-when-workflows-run/control-workflow-concurrency) | **Caller.** An action cannot create a job/workflow concurrency group. | Caller. | Caller. | Caller. |
| **`if` gating** | Call job supports `if`; called jobs and called steps may also have their own conditions. [Reuse reference](https://docs.github.com/en/actions/reference/workflows-and-actions/reusing-workflow-configurations#supported-keywords-for-jobs-that-call-a-reusable-workflow) | Caller step supports `if`; composite child steps also support `runs.steps[*].if`. [Metadata syntax](https://docs.github.com/en/actions/reference/workflows-and-actions/metadata-syntax#runsstepsif) | Caller step controls main invocation; metadata additionally supports `pre-if` and `post-if`. [Metadata syntax](https://docs.github.com/en/actions/reference/workflows-and-actions/metadata-syntax#runspre-if) | Caller step controls invocation; Docker metadata has pre/post conditions. | Ordinary step `if`. |
| **Multi-job orchestration / `needs`** | **Yes.** The called workflow may contain multiple ordinary jobs and dependencies; the call job itself supports `needs`. A workflow is called at job level, not as a step. [Reuse workflows](https://docs.github.com/en/actions/how-tos/reuse-automations/reuse-workflows#calling-a-reusable-workflow) | **No.** A composite combines steps and is consumed as a single caller step. [Custom actions](https://docs.github.com/en/actions/concepts/workflows-and-actions/custom-actions#composite-actions) | No; one action invocation, with optional pre/post code in the same job. | No; one action invocation, with optional pre/post containers in the same job. | No reusable orchestration by itself; the consumer writes the jobs. |
| **Artifact upload/download scope** | Called jobs participate in the caller’s workflow execution. `upload-artifact`/`download-artifact` can move files between dependent jobs in that workflow; a download from a different workflow or run requires an explicit token and run identifier. [Artifact docs](https://docs.github.com/en/actions/tutorials/store-and-share-data#downloading-artifacts-during-a-workflow-run) | Shares the caller job’s filesystem. A composite can natively invoke `actions/upload-artifact` or `actions/download-artifact` as nested `uses` steps; artifacts remain scoped to the caller workflow run. [Composite `uses`](https://docs.github.com/en/actions/reference/workflows-and-actions/metadata-syntax#runsstepsuses) [Artifact docs](https://docs.github.com/en/actions/tutorials/store-and-share-data) | Shares caller workspace; JavaScript metadata cannot contain nested workflow `uses` steps, so the caller normally uploads/downloads around it or the code calls an API itself. | Shares the mounted workspace with caller; the caller normally uploads/downloads around it. | Caller steps use the same workflow-run artifact scope. |
| **Job/workflow outputs** | Called steps map to called-job outputs, then `on.workflow_call.outputs`; caller consumes them through `needs.<call-job>.outputs`. [Reusable outputs](https://docs.github.com/en/actions/how-tos/reuse-automations/reuse-workflows#using-outputs-from-a-reusable-workflow) | Defines **action/step outputs** mapped from child steps. Caller must map those to `jobs.<id>.outputs` if downstream jobs need them. | Defines step outputs through action metadata/`GITHUB_OUTPUT`; caller maps to job outputs. | Same. | Writes `GITHUB_OUTPUT`; caller maps step output to job output. Outputs are limited to **1 MB per job** and **50 MB per workflow run**, approximated as UTF-16. [Metadata syntax](https://docs.github.com/en/actions/reference/workflows-and-actions/metadata-syntax#outputs-for-docker-container-and-javascript-actions) |
| **`continue-on-error`** | The reusable-workflow **call job cannot use it** because it is absent from the supported keyword list. Ordinary jobs/steps inside the called workflow may use job/step `continue-on-error`. [Reuse reference](https://docs.github.com/en/actions/reference/workflows-and-actions/reusing-workflow-configurations#supported-keywords-for-jobs-that-call-a-reusable-workflow) [Workflow syntax](https://docs.github.com/en/actions/reference/workflows-and-actions/workflow-syntax#jobsjob_idcontinue-on-error) | Caller may set step `continue-on-error`; child composite steps also support it. [Metadata syntax](https://docs.github.com/en/actions/reference/workflows-and-actions/metadata-syntax#runsstepscontinue-on-error) | Caller step may set it; no action-metadata equivalent around only `main`. | Caller step may set it. | Ordinary step supports it. |
| **Timeouts** | The call job cannot set `timeout-minutes` because it is not a supported call-job keyword. Ordinary jobs inside the callee can. Job timeout defaults to **360 minutes**; workflow-run lifetime is **35 days**, GitHub-hosted job execution is capped at **6 hours**, and self-hosted job execution at **5 days**. [Reuse reference](https://docs.github.com/en/actions/reference/workflows-and-actions/reusing-workflow-configurations#supported-keywords-for-jobs-that-call-a-reusable-workflow) [Workflow syntax](https://docs.github.com/en/actions/reference/workflows-and-actions/workflow-syntax#jobsjob_idtimeout-minutes) [Actions limits](https://docs.github.com/en/actions/reference/limits) | Caller can set a timeout on the composite invocation step. Step timeout maximum is exactly **360 minutes** on both GitHub-hosted and self-hosted runners. Composite metadata does not provide an internal `timeout-minutes` child-step field. [Workflow syntax](https://docs.github.com/en/actions/reference/workflows-and-actions/workflow-syntax#jobsjob_idstepstimeout-minutes) [Metadata syntax](https://docs.github.com/en/actions/reference/workflows-and-actions/metadata-syntax#runs-for-composite-actions) | Caller step maximum **360 minutes**. | Caller step maximum **360 minutes**. | Ordinary step maximum **360 minutes**. |
| **Nesting depth** | Exactly **10 workflow levels total**: top-level caller plus up to **9** called-workflow levels. Loops are forbidden. [Reuse workflows](https://docs.github.com/en/actions/how-tos/reuse-automations/reuse-workflows#nesting-reusable-workflows) | Composite actions can nest other actions through `uses`, including with `$/`; **no current numeric composite-action nesting limit was found** in GitHub’s metadata reference, Actions limits page, or action-composition changelog. [Metadata syntax](https://docs.github.com/en/actions/reference/workflows-and-actions/metadata-syntax#runsstepsuses) [Actions limits](https://docs.github.com/en/actions/reference/limits) [Composition changelog](https://github.blog/changelog/2021-08-25-github-actions-reduce-duplication-with-action-composition/) | No native nested `uses` in JavaScript action metadata. | No native nested `uses` in Docker action metadata. | Not applicable. |
| **Total reusable-unit calls** | Maximum **50 unique reusable workflows** from one top-level workflow file, including the entire nested tree. `A → B → C` counts as two called workflows. The limits were raised from 4/20 to 10/50 in November 2025. [Reuse reference](https://docs.github.com/en/actions/reference/workflows-and-actions/reusing-workflow-configurations#limitations-of-reusable-workflows) [GitHub changelog, 2025-11-06](https://github.blog/changelog/2025-11-06-new-releases-for-github-actions-november-2025/) | **No action-specific total-call number is documented** on the current Actions limits or metadata pages; invocations remain subject to overall workflow/job/step and matrix limits. [Actions limits](https://docs.github.com/en/actions/reference/limits) [Metadata syntax](https://docs.github.com/en/actions/reference/workflows-and-actions/metadata-syntax) | Same negative result. | Same negative result. | Not applicable. |
| **Runner portability** | Callee chooses its job runners, evaluated and billed in the caller’s context. [Reuse reference](https://docs.github.com/en/actions/reference/workflows-and-actions/reusing-workflow-configurations#how-reusable-workflows-use-runners) | Linux, macOS, Windows, subject to the shells/tools its steps require. | Linux, macOS, Windows if the packaged JavaScript does not depend on platform-specific binaries. | **Linux only**; self-hosted runner must be Linux with Docker installed. | Depends on the installed CLI binary and caller runner. [Action type matrix](https://docs.github.com/en/actions/concepts/workflows-and-actions/custom-actions#types-of-actions) |

## 2. Attestations and signing

### GitHub-native provenance and SBOM attestations

GitHub’s current documentation uses `actions/attest@v4` for both build provenance and SBOM attestations. `actions/attest-build-provenance@v4` is now a wrapper around `actions/attest`; `actions/attest-sbom` is deprecated in favor of `actions/attest` and remains a wrapper for compatibility. [GitHub Docs: artifact attestations](https://docs.github.com/en/actions/how-tos/secure-your-work/use-artifact-attestations/use-artifact-attestations) [GitHub-owned provenance action](https://github.com/actions/attest-build-provenance) [GitHub-owned SBOM action](https://github.com/actions/attest-sbom)

For either a binary provenance attestation or a binary SBOM attestation, the job requires:

```yaml
permissions:
  contents: read
  id-token: write
  attestations: write
```

GitHub documents those exact three permissions for both cases. For a container image pushed to GitHub Packages, add `packages: write`; `artifact-metadata: write` is separately required only if using the linked-artifacts storage-record feature. [GitHub Docs: provenance permissions](https://docs.github.com/en/actions/how-tos/secure-your-work/use-artifact-attestations/use-artifact-attestations#generating-build-provenance-for-binaries) [GitHub Docs: SBOM permissions](https://docs.github.com/en/actions/how-tos/secure-your-work/use-artifact-attestations/use-artifact-attestations#generating-an-sbom-attestation-for-binaries) [GitHub Docs: container permissions](https://docs.github.com/en/actions/how-tos/secure-your-work/use-artifact-attestations/use-artifact-attestations#generating-build-provenance-for-container-images)

The roles are distinct:

- `id-token: write` allows the job to request the OIDC JWT used for signing. It grants no write access to other resources. [OIDC reference](https://docs.github.com/en/actions/reference/security/oidc#required-permission)
- `attestations: write` authorizes creating the attestation in GitHub’s attestations service. [Workflow permission reference](https://docs.github.com/en/actions/reference/workflows-and-actions/workflow-syntax#permissions)
- `contents: read` permits reading repository contents/checking out the source; it is not the permission that mints the signature. [Workflow permission reference](https://docs.github.com/en/actions/reference/workflows-and-actions/workflow-syntax#permissions)

For a **cross-organization reusable workflow**, both sides matter:

1. The external caller’s call job must grant at least `contents: read`, `id-token: write`, and `attestations: write` (plus registry/package permissions as applicable).
2. The called workflow should declare the same least-privilege set on the attesting job.
3. The callee cannot elevate anything omitted or denied by the caller. GitHub explicitly requires an external reusable-workflow caller to set `id-token: write` at caller workflow/job level. [OIDC reusable-workflow rules](https://docs.github.com/en/actions/reference/security/oidc#reusable-workflows) [Reusable permission ceiling](https://docs.github.com/en/actions/reference/workflows-and-actions/reusing-workflow-configurations#access-and-permissions-for-nested-workflows)

For a **composite, JavaScript, or Docker action**, only the caller job can declare those permissions. A composite action can invoke `actions/attest` as a nested step and can request the OIDC token, but it cannot add `permissions` to the job. Putting a `permissions` field in `action.yml` is not part of the metadata schema. [Metadata syntax](https://docs.github.com/en/actions/reference/workflows-and-actions/metadata-syntax) [OIDC custom-action methods](https://docs.github.com/en/actions/reference/security/oidc#methods-for-requesting-the-oidc-token)

The attestation is associated with the repository from which the workflow was initiated. Since a reusable workflow’s `github` context is the caller’s context, an arbitrary-org consumer produces attestations for the consumer repository, not for the repository hosting the release workflow. [Reusable workflow `github` context](https://docs.github.com/en/actions/reference/workflows-and-actions/reusing-workflow-configurations#github-context) [Artifact-attestation contents](https://docs.github.com/en/actions/concepts/security/artifact-attestations#overview)

### Cosign keyless signing

Raw `cosign sign` keyless signing is different from uploading a GitHub-native attestation:

- It requires `id-token: write` so Cosign can retrieve the GitHub OIDC token.
- It requires whatever registry credential can push the signature; for GHCR that generally means `packages: write`.
- `contents: read` is needed only when the job must read/check out repository content.
- It **does not require `attestations: write` unless the workflow also calls GitHub’s attestations API/action**. That permission controls GitHub artifact attestations, not ordinary Sigstore/Cosign registry signatures. [GitHub OIDC permission semantics](https://docs.github.com/en/actions/reference/security/oidc#required-permission) [GitHub `attestations` permission semantics](https://docs.github.com/en/actions/reference/workflows-and-actions/workflow-syntax#permissions) [Sigstore CI example](https://docs.sigstore.dev/quickstart/quickstart-ci/#signing-and-verifying-a-container-image)

A composite action can run Cosign or a Cosign installer and receive the ambient OIDC request variables, but the caller job must grant `id-token: write`. The action cannot make that permission self-contained.

## 3. GitHub App credential path

### Consumer-owned configuration

For arbitrary organizations, prefer a GitHub App when `GITHUB_TOKEN` is insufficient, because `GITHUB_TOKEN` is scoped to the repository running the workflow. GitHub’s documented sequence is:

1. Register a GitHub App with the required permissions.
2. Store the App **client ID** as an Actions configuration variable.
3. Store the App **private key** as an Actions secret.
4. Install the App on the consumer account and grant it access to the required repositories.
5. Mint an installation token in the workflow, normally with `actions/create-github-app-token@v3`. [GitHub Docs: App authentication in Actions](https://docs.github.com/en/apps/creating-github-apps/authenticating-with-a-github-app/making-authenticated-api-requests-with-a-github-app-in-a-github-actions-workflow)

What is organization-specific:

- The App installation into that consumer organization/account.
- The repositories granted to that installation.
- The App permissions approved by that organization.
- The variable/secret names chosen by the consumer.
- If every organization creates its own App, its client ID/private key are also organization-specific. If one centrally owned App is installed into several organizations, the App credentials remain those of the same registration, while each installation and repository grant remains account-specific. [GitHub Docs: registration/install/token sequence](https://docs.github.com/en/apps/creating-github-apps/authenticating-with-a-github-app/making-authenticated-api-requests-with-a-github-app-in-a-github-actions-workflow#authenticating-with-a-github-app)

`actions/create-github-app-token` defaults to a token for the current repository. Its `owner` input selects the current owner’s full installation or another owner’s installation; `repositories` narrows the repository list. Requested permissions must already exist on the installation. [GitHub-owned action examples](https://github.com/actions/create-github-app-token#usage)

GitHub’s raw installation-token API may narrow a token to up to **500 repositories**. It cannot add repositories or permissions the installation lacks. An installation token expires after exactly **1 hour**. [GitHub Docs: installation token](https://docs.github.com/en/apps/creating-github-apps/authenticating-with-a-github-app/generating-an-installation-access-token-for-a-github-app)

Equivalent implementations are:

- `actions/create-github-app-token`;
- a script that creates the App JWT, resolves the installation ID, and posts to `/app/installations/{id}/access_tokens`;
- an Octokit SDK, which can generate and refresh installation tokens. GitHub documents all three paths. [GitHub Docs](https://docs.github.com/en/apps/creating-github-apps/authenticating-with-a-github-app/making-authenticated-api-requests-with-a-github-app-in-a-github-actions-workflow) [Installation-token docs](https://docs.github.com/en/apps/creating-github-apps/authenticating-with-a-github-app/generating-an-installation-access-token-for-a-github-app)

### Reusable workflow versus action

For a public reusable workflow called from an unrelated organization, `secrets: inherit` is unavailable unless both organizations belong to the same enterprise. The caller must map the private key explicitly:

```yaml
jobs:
  publish:
    permissions:
      contents: read
      id-token: write
      attestations: write
    uses: release-owner/release/.github/workflows/publish.yml@FULL_SHA
    with:
      app-client-id: ${{ vars.RELEASE_APP_CLIENT_ID }}
    secrets:
      app-private-key: ${{ secrets.RELEASE_APP_PRIVATE_KEY }}
```

The called workflow must declare `app-private-key` in `on.workflow_call.secrets`. [Reusable secrets](https://docs.github.com/en/actions/how-tos/reuse-automations/reuse-workflows#passing-inputs-and-secrets-to-a-reusable-workflow)

A composite action receives the client ID/private key as explicit action inputs because it has no `secrets` context:

```yaml
- uses: release-owner/release/path/to/action@FULL_SHA
  with:
    app-client-id: ${{ vars.RELEASE_APP_CLIENT_ID }}
    app-private-key: ${{ secrets.RELEASE_APP_PRIVATE_KEY }}
```

The composite **can mint the App token internally** by nesting `actions/create-github-app-token`; composite metadata explicitly supports `uses` child steps. The secret is explicit input data, not an ambient composite `secrets` context. [Composite `uses`](https://docs.github.com/en/actions/reference/workflows-and-actions/metadata-syntax#runsstepsuses) [Composite secret restriction](https://docs.github.com/en/actions/reference/workflows-and-actions/contexts#secrets-context)

A JavaScript action can implement the JWT/REST flow directly. A Docker action can do the same if its container contains the necessary tooling. Neither can declare its own job permissions, though App-token minting itself uses the App private key and does not require `id-token: write`.

## 4. Version pinning and upgrade UX

### Reusable workflows

An external caller references:

```yaml
uses: owner/repo/.github/workflows/publish.yml@REF
```

`REF` may be a full commit SHA, release tag, or branch. If a tag and branch have the same name, the tag wins. GitHub calls a commit SHA the safest option for stability and security. A same-repository `$/...` or `./...` reference has no `@ref` and resolves from the **same commit as the caller workflow**. Contexts/expressions cannot be used in the reusable-workflow `uses` value. [Reuse workflows](https://docs.github.com/en/actions/how-tos/reuse-automations/reuse-workflows#calling-a-reusable-workflow)

Re-run behavior matters for floating refs:

- “Re-run all jobs” resolves the reusable workflow from the currently specified non-SHA ref again.
- Re-running only failed jobs or one job reuses the exact callee commit SHA from the first attempt. [Reuse reference](https://docs.github.com/en/actions/reference/workflows-and-actions/reusing-workflow-configurations#behavior-of-reusable-workflows-when-re-running-jobs)

### Actions

Actions also accept a full SHA, tag, or branch. GitHub recommends full SHAs for third-party actions. A full SHA is immutable but does not automatically receive fixes. Tags can be moved or deleted; branches continuously move and can introduce breaking changes. [GitHub Docs: finding and customizing actions](https://docs.github.com/en/actions/how-tos/write-workflows/choose-what-workflows-do/find-and-customize-actions#using-release-management-for-your-custom-actions)

`@v1` is a **floating major tag** when the maintainer follows GitHub’s release-management guidance: the maintainer moves `v1` to the latest compatible `v1.x.y` release and creates `v2` for breaking changes. Consumers thereby accept future updates selected by the maintainer. GitHub explicitly says compatibility behavior remains at the action author’s discretion; it is not enforced by Actions. [Managing custom actions](https://docs.github.com/en/actions/how-tos/create-and-publish-actions/manage-custom-actions#using-tags-for-release-management) [Workflow `uses` guidance](https://docs.github.com/en/actions/reference/workflows-and-actions/workflow-syntax#jobsjob_idstepsuses)

The same Git-ref mechanics apply to a reusable workflow tagged `v1`, but GitHub’s documented moving-major release procedure is specifically written for actions. A full SHA remains the strongest external pin for both.

### Threading the CLI version

Both shapes can expose an explicit `cli-version` string:

- reusable workflow: `on.workflow_call.inputs.cli-version`;
- custom action: `inputs.cli-version`;
- plain steps: workflow input or environment variable.

The workflow should pass that value to the action, and the action should use it when installing the CLI. A layered consumer then has:

1. an immutable workflow/action source pin; and
2. an explicit CLI release pin.

Those pins serve different purposes. A workflow SHA is not necessarily a CLI semantic version, and `github.action_ref` may be `v1`, a branch, or a SHA. Do not silently equate them unless the release process guarantees one repository release atomically versions both surfaces.

For internal composition, use:

```yaml
- uses: $/path/to/release-action
```

This makes the reusable workflow and its sibling action come from exactly the same repository commit without a second hardcoded action ref. GitHub says this avoids quietly defeating SHA pinning and keeps internal references consistent when an external caller pins the workflow to a full SHA. [GitHub changelog, 2026-07-30](https://github.blog/changelog/2026-07-30-reference-same-repository-actions-with-self-repository-syntax/)

### Caller versus callee identity

A reusable workflow executes with the `github` context of its **caller**. Therefore `github.repository`, `github.ref`, and `github.sha` describe the consumer run, not the repository/ref that stores the reusable workflow. GitHub provides `github.job_workflow_ref` and `github.job_workflow_sha` to identify the called workflow. [Reuse reference](https://docs.github.com/en/actions/reference/workflows-and-actions/reusing-workflow-configurations#github-context) [OIDC claim reference](https://docs.github.com/en/actions/reference/security/oidc#oidc-token-claims)

For actions, `github.action_repository` and `github.action_ref` identify the action repository/ref. [Contexts reference](https://docs.github.com/en/actions/reference/workflows-and-actions/contexts#github-context)

## 5. Dogfood mode

### Reusable-workflow shape

The owning repository should call its workflow using a same-repository reference:

```yaml
jobs:
  publish:
    uses: $/.github/workflows/publish.yml
```

or the older:

```yaml
jobs:
  publish:
    uses: ./.github/workflows/publish.yml
```

Both resolve the called workflow from the same commit as the caller; `$/` is preferred on GitHub.com. Thus a branch run exercises the branch’s workflow definition rather than the last released workflow. [Reuse workflows](https://docs.github.com/en/actions/how-tos/reuse-automations/reuse-workflows#calling-a-reusable-workflow)

Inside dogfood jobs:

1. Check out the caller repository/current event commit.
2. Build the CLI from that source.
3. Invoke the resulting path instead of downloading a released CLI.
4. If later called-workflow jobs need that binary, upload/download it because each job has its own runner; workflow artifacts are GitHub’s documented mechanism for passing files between jobs. [Artifact docs](https://docs.github.com/en/actions/tutorials/store-and-share-data#passing-data-between-jobs-in-a-workflow)

External consumers instead call:

```yaml
uses: release-owner/release/.github/workflows/publish.yml@FULL_SHA
```

and pass an explicit `cli-version`. A dogfood switch must be constrained to the owning repository; otherwise an external caller could cause the reusable workflow to “build the CLI” from the consumer’s checked-out repository. The called workflow’s `github` context is the caller’s context, and `actions/checkout` defaults to `${{ github.repository }}` and the triggering ref/SHA. [Reusable `github` context](https://docs.github.com/en/actions/reference/workflows-and-actions/reusing-workflow-configurations#github-context) [GitHub-owned checkout defaults](https://github.com/actions/checkout#usage)

### Composite-action shape

The owning workflow can:

1. build `release-cli` in an earlier step;
2. invoke the branch’s action with `uses: $/path/to/release-action`; and
3. pass `cli-path: ./path/to/built/release-cli`.

The action and binary share the same caller job workspace. `$/` resolves the action from the exact running commit and requires no checkout merely to load `action.yml`. [GitHub Docs: same-repository action](https://docs.github.com/en/actions/how-tos/write-workflows/choose-what-workflows-do/find-and-customize-actions#adding-an-action-from-the-same-repository)

External callers pin `owner/repo/path@FULL_SHA` and omit `cli-path`, causing the action to install the explicitly pinned CLI release. This makes dogfood selection data-driven while preserving one action implementation.

A JavaScript wrapper can also accept/spawn a local path, but adds a JavaScript distribution. A Docker action is a poor dogfood bridge for a host-built binary because it is Linux-only and runs inside a container; the composite action directly uses the caller workspace and runner. [Custom-action platform matrix](https://docs.github.com/en/actions/concepts/workflows-and-actions/custom-actions#types-of-actions)

### Plain CLI steps

Plain steps make dogfooding mechanically simplest:

```yaml
- if: DOGFOOD_CONDITION
  run: go build -o "$RUNNER_TEMP/release-cli" ./cmd/release-cli

- if: DOGFOOD_CONDITION
  run: "$RUNNER_TEMP/release-cli" publish --profile go

- if: EXTERNAL_CONDITION
  run: |
    # install exact CLI version
    release-cli publish --profile go
```

They do not, however, transport the reusable permissions, environment, artifact, matrix, and job-output policy to consumers.

### What breaks when “the same workflow file” serves owner and consumers

There are three distinct issues:

1. **The caller syntax cannot be dynamically switched.** The owning repository wants `$/...` or `./...` at the current branch commit; an external repository requires `owner/repo/.github/workflows/file.yml@ref`. Reusable-workflow `jobs.<id>.uses` does not accept contexts or expressions, so one call job cannot select between those forms dynamically. The called workflow file may be shared, but the owner and consumer caller stubs need different static references. [Reuse workflow syntax](https://docs.github.com/en/actions/how-tos/reuse-automations/reuse-workflows#calling-a-reusable-workflow)

2. **Caller contexts do not identify the callee checkout.** A remote reusable workflow sees the external consumer’s repository/ref in `github.*`; an unqualified checkout gets consumer source. Use `github.job_workflow_ref`/`sha` when callee identity is needed, or use the new `$/` self-reference for sibling actions/workflows in the callee repository. [Reusable context](https://docs.github.com/en/actions/reference/workflows-and-actions/reusing-workflow-configurations#github-context) [Self-reference changelog](https://github.blog/changelog/2026-07-30-reference-same-repository-actions-with-self-repository-syntax/)

3. **A locally built binary is job-local.** A reusable workflow with multiple jobs cannot assume that the binary built in one job remains on another runner. It must upload/download it or rebuild it. A composite action invoked after a build in the same job does not have this problem. [Artifact docs](https://docs.github.com/en/actions/tutorials/store-and-share-data#passing-data-between-jobs-in-a-workflow)

## 6. Failure modes and ergonomics

### Secrets

- Composite actions cannot use the `secrets` context. Every App private key, registry password, or other secret must be an explicit input/env. [Contexts reference](https://docs.github.com/en/actions/reference/workflows-and-actions/contexts#secrets-context)
- JavaScript and Docker actions likewise receive ordinary secrets only when the caller includes them as inputs/env. [Secrets](https://docs.github.com/en/actions/concepts/security/secrets)
- A custom action can access `github.token` even when the caller did not explicitly pass `secrets.GITHUB_TOKEN`. Consequently the caller must minimize job permissions rather than assuming omission of a token input prevents access. [GITHUB_TOKEN tutorial](https://docs.github.com/en/actions/tutorials/authenticate-with-github_token#using-the-github_token-in-a-workflow)
- Reusable-workflow `secrets: inherit` does not make arbitrary cross-org delivery transparent; it is limited to the same organization or enterprise, and each nested edge must forward secrets again. [Reuse workflows](https://docs.github.com/en/actions/how-tos/reuse-automations/reuse-workflows#passing-secrets-to-nested-workflows)
- `required: true` in action metadata does **not automatically produce an error** when an input is missing; the action must validate it. [Metadata syntax](https://docs.github.com/en/actions/reference/workflows-and-actions/metadata-syntax#inputsinput_idrequired)

### `env` inheritance

Workflow-level `env` from the caller does not propagate into a called reusable workflow, and called-workflow `env` does not propagate back. Reusable workflow outputs or repository/organization/environment `vars` are the documented alternatives. `GITHUB_ENV` cannot pass values back to caller steps because a reusable workflow is a job call, not a step in the caller job. [Reuse configuration limitations](https://docs.github.com/en/actions/reference/workflows-and-actions/reusing-workflow-configurations#limitations-of-reusable-workflows)

Actions run inside the caller job and therefore participate in its process environment. A composite action can write `GITHUB_ENV` for subsequent steps in that same job. [Metadata syntax](https://docs.github.com/en/actions/reference/workflows-and-actions/metadata-syntax#runsstepsenv)

### `GITHUB_TOKEN` defaults

For ordinary actions and plain steps, GitHub calculates `GITHUB_TOKEN` permissions from enterprise/organization/repository defaults, then workflow-level `permissions`, then job-level `permissions`; all actions and run commands in the job receive the job-level result. If any permissions are enumerated, all unspecified permissions become `none`. [Workflow syntax](https://docs.github.com/en/actions/reference/workflows-and-actions/workflow-syntax#how-permissions-are-calculated-for-a-workflow-job)

For a reusable-workflow call job:

- if the caller does not specify `permissions`, the called workflow gets the caller repository’s default token permissions;
- the callee can only reduce those permissions;
- the called workflow automatically has `github.token` and `secrets.GITHUB_TOKEN` associated with the caller. [Reuse reference](https://docs.github.com/en/actions/reference/workflows-and-actions/reusing-workflow-configurations#supported-keywords-for-jobs-that-call-a-reusable-workflow) [Reusable `github` context](https://docs.github.com/en/actions/reference/workflows-and-actions/reusing-workflow-configurations#github-context)

Therefore a reusable workflow cannot guarantee attestation/OIDC permissions by declaring them only in the callee. The external caller must grant the ceiling explicitly.

### Environment gates

A caller cannot attach `environment` to a reusable-workflow call job. The environment must be declared on a job inside the called workflow. [Supported keywords](https://docs.github.com/en/actions/reference/workflows-and-actions/reusing-workflow-configurations#supported-keywords-for-jobs-that-call-a-reusable-workflow)

A nonexistent environment named by a workflow is automatically created without protection rules or secrets. Therefore merely hardcoding `environment: release` does not prove that a required-reviewer gate exists in every consumer repository; the consumer must configure that environment. [Managing environments](https://docs.github.com/en/actions/how-tos/deploy/configure-and-manage-deployments/manage-environments)

With a custom action, the consumer visibly declares the protected environment on its ordinary job, but this also leaves more orchestration policy in consumer YAML.

### Logging, grouping, and annotations

All four delivery shapes can use runner workflow commands:

- `::group::` / `::endgroup::` for expandable log sections;
- `::notice`, `::warning`, and `::error` for annotations;
- `GITHUB_OUTPUT`, `GITHUB_ENV`, `GITHUB_PATH`, and `GITHUB_STEP_SUMMARY`;
- JavaScript actions can use equivalent `@actions/core` methods. [Workflow commands](https://docs.github.com/en/actions/reference/workflows-and-actions/workflow-commands)

A composite can assign names and IDs to internal child steps, but callers still invoke the composite as one top-level step. A reusable workflow provides actual separate jobs and job dependencies, which is materially clearer for a multi-stage release. [Composite metadata](https://docs.github.com/en/actions/reference/workflows-and-actions/metadata-syntax#runsstepsname) [Reusable workflow calls](https://docs.github.com/en/actions/how-tos/reuse-automations/reuse-workflows#calling-a-reusable-workflow)

### Error surfacing

GitHub maps exit code `0` to success and any nonzero action exit to failure. A failed action causes dependent future actions to be skipped unless conditions or `continue-on-error` alter handling. JavaScript actions can call `core.setFailed`; Docker entrypoints must exit nonzero. [Action exit codes](https://docs.github.com/en/actions/how-tos/create-and-publish-actions/set-exit-codes)

A CLI used through composite or plain steps should both:

1. exit nonzero for failure; and
2. emit GitHub annotations only when running under Actions if richer error display is wanted.

The caller may use step `continue-on-error` around an action or CLI step. A reusable-workflow call job cannot use that keyword, so tolerated failures must be modeled within the called workflow’s own jobs/steps and exposed through outputs. [Supported call-job keywords](https://docs.github.com/en/actions/reference/workflows-and-actions/reusing-workflow-configurations#supported-keywords-for-jobs-that-call-a-reusable-workflow)

### Distribution/access policy

For arbitrary-org adoption:

- A reusable workflow in a public repository is callable when the consumer organization’s Actions policy allows public reusable workflows.
- Private called-workflow repositories require explicit access configuration and are not usable by public caller repositories.
- A custom action shared with everyone must be in a public repository.
- Caller repository/organization action allowlists can block either shape. [Reusable workflow access](https://docs.github.com/en/actions/reference/workflows-and-actions/reusing-workflow-configurations#access-to-reusable-workflows) [Custom action sharing](https://docs.github.com/en/actions/concepts/workflows-and-actions/custom-actions#about-custom-actions)

GitHub Actions does not follow redirects after an action/reusable-workflow owner or repository rename; old references fail. [Reuse reference](https://docs.github.com/en/actions/reference/workflows-and-actions/reusing-workflow-configurations#access-to-reusable-workflows)

## 7. Recommendation and deciding tradeoffs

### Recommended public contract

**Publish both:**

- **Primary:** profile-specific reusable workflow entrypoints such as a reusable prepublish/publish workflow, called with a `profile` and explicit `cli-version`.
- **Secondary:** one thin composite action that installs or accepts the CLI and invokes a requested profile/operation.
- **Underlying executable:** the CLI remains directly installable/runnable for local testing and for consumers that intentionally own their GitHub orchestration.

This is option **(c)**, not four equal alternatives.

### Why the reusable workflow is mandatory

The release system needs capabilities that actions fundamentally cannot own:

- job-scoped `permissions`, including OIDC/attestations;
- protected-environment jobs and required-reviewer gates;
- matrix fan-out;
- job/workflow concurrency;
- multi-job `needs`;
- artifact transfer across runners;
- called-workflow outputs;
- per-job timeouts and runner selection.

Those are workflow/job constructs, while GitHub defines actions as individual tasks used inside jobs. [Custom actions](https://docs.github.com/en/actions/concepts/workflows-and-actions/custom-actions) [Reusable workflow capabilities](https://docs.github.com/en/actions/reference/workflows-and-actions/reusing-workflow-configurations#supported-keywords-for-jobs-that-call-a-reusable-workflow)

### Why retain the composite action

The action provides a stable per-job bridge that the reusable workflow and custom consumer workflows can share:

- one CLI install/verification path;
- one `cli-version`/`cli-path` contract;
- action outputs mapped from the CLI;
- Action-native annotations and grouped logs;
- straightforward branch dogfooding with a caller-built executable;
- direct use in consumer-owned jobs with their own environment/permission choices;
- internal exact-ref composition through `$/`, avoiding a second internal source pin. [Self-reference changelog](https://github.blog/changelog/2026-07-30-reference-same-repository-actions-with-self-repository-syntax/)

Use a **composite**, not JavaScript or Docker, for this adapter unless later requirements demand Node APIs or container isolation. The CLI already contains the behavior; a JavaScript action would duplicate launcher/distribution code, and a Docker action would restrict consumers to Linux and add image pull/build latency. [Action types](https://docs.github.com/en/actions/concepts/workflows-and-actions/custom-actions#types-of-actions)

### Costs of layering

The costs are real and should remain visible:

- Two public schemas to maintain: `workflow_call` inputs/secrets/outputs and action inputs/outputs.
- A workflow caller still must grant cross-org OIDC/attestation permissions; neither the reusable workflow nor action can silently elevate them.
- Secrets must be explicitly mapped at the external workflow boundary and again into the composite action.
- A composite adds a top-level log layer around CLI output.
- `cli-version`, workflow/action source version, and any independently released binary can drift unless the release contract pins them explicitly.
- The action cannot provide environment gating, job concurrency, multi-job orchestration, or job-level timeout policy; consumers who use only the action must supply those themselves.

### Why not the other choices

- **(a) Reusable workflows only:** viable for turnkey consumers, but loses a supported step-level adapter, makes custom consumer orchestration duplicate install/output behavior, and makes same-job dogfood injection less convenient.
- **(b) Custom action only:** cannot encode the job-level security and orchestration required by a release pipeline.
- **(d) Plain installed-CLI steps:** maximizes transparency and local similarity but externalizes nearly all GitHub-specific correctness to copied consumer YAML; it is suitable as an escape hatch, not the system’s supported default.

## Unknowns / negative findings

1. **No current numeric nesting-depth or total-call limit for composite/JavaScript/Docker actions was found** in GitHub’s [metadata reference](https://docs.github.com/en/actions/reference/workflows-and-actions/metadata-syntax), [Actions limits](https://docs.github.com/en/actions/reference/limits), or official [action-composition changelog](https://github.blog/changelog/2021-08-25-github-actions-reduce-duplication-with-action-composition/). The exact documented **10 levels / 50 unique calls** apply to reusable workflows, not custom actions.

2. GitHub documents how moving major action tags such as `v1` should work, but compatibility is explicitly at the author’s discretion; Actions does not enforce semantic-version compatibility. [Workflow syntax](https://docs.github.com/en/actions/reference/workflows-and-actions/workflow-syntax#jobsjob_idstepsuses)

3. Raw Cosign’s registry credentials and signature-storage behavior are registry-specific. GitHub’s job-level conclusion is firm—`id-token: write` enables OIDC and `attestations: write` is not needed unless using GitHub’s attestation service—but the necessary push credential depends on the target registry. [GitHub OIDC reference](https://docs.github.com/en/actions/reference/security/oidc#required-permission) [Sigstore CI quickstart](https://docs.sigstore.dev/quickstart/quickstart-ci/#signing-and-verifying-a-container-image)
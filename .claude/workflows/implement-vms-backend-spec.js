export const meta = {
  name: 'implement-vms-backend-spec',
  description: 'Implement VMS backend tickets 02-18 in dependency order: one Opus implementer per ticket in its own worktree, serialized mergers onto vms/backend-api, then code review and cleanup',
  whenToUse: 'Resume or rerun the VMS backend spec implementation',
  phases: [
    { title: 'Implement', detail: 'one implementer per ready ticket, own worktree and branch', model: 'opus' },
    { title: 'Merge', detail: 'serialized merge onto vms/backend-api with full gate', model: 'opus' },
    { title: 'Review', detail: 'code-review of the integration branch and one fix pass', model: 'opus' },
    { title: 'Cleanup', detail: 'remove worktrees and scratch clusters', model: 'opus' },
  ],
}

const ROOT = 'D:\\oikn\\VMS'
const TRACKER = ROOT + '\\.scratch\\backend-api-2026-11-18'
const NOTES = TRACKER + '\\notes\\build-environment.md'
const PENDING = TRACKER + '\\notes\\pending-contract-updates'
const SPEC = TRACKER + '\\spec.md'
const BRANCH = args.integrationBranch
const TRAILER = 'Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>'

const IMPL = {
  type: 'object',
  properties: {
    status: { type: 'string', enum: ['complete', 'incomplete'] },
    branch: { type: 'string' },
    worktree: { type: 'string' },
    commits: { type: 'array', items: { type: 'string' } },
    summary: { type: 'string' },
    unmetCriteria: { type: 'array', items: { type: 'string' } },
    contractUpdates: { type: 'string' },
    gate: { type: 'string' },
  },
  required: ['status', 'branch', 'summary', 'contractUpdates', 'gate'],
}
const MERGE = {
  type: 'object',
  properties: {
    merged: { type: 'boolean' },
    mergeCommit: { type: 'string' },
    gate: { type: 'string' },
    contract: { type: 'string' },
    uncheckedCriteria: { type: 'array', items: { type: 'string' } },
    notes: { type: 'string' },
  },
  required: ['merged', 'gate', 'notes'],
}
const REVIEW = {
  type: 'object',
  properties: {
    findings: {
      type: 'array',
      items: {
        type: 'object',
        properties: {
          severity: { type: 'string', enum: ['high', 'medium', 'low'] },
          file: { type: 'string' },
          summary: { type: 'string' },
          fix: { type: 'string' },
        },
        required: ['severity', 'file', 'summary', 'fix'],
      },
    },
    verdict: { type: 'string' },
  },
  required: ['findings', 'verdict'],
}

const done = new Set(args.done)
const failed = new Set()
const started = new Set()
const results = {}
let mergeChain = Promise.resolve()

const ticketFile = (t) => `${TRACKER}\\issues\\${t.id}-${t.slug}.md`
const port = (id) => 56000 + parseInt(id, 10)

function implPrompt(t) {
  const special = t.id === '03'
    ? `\nThis ticket IS the contract pipeline. You may and must edit ${ROOT}\\docs\\contracts (outer repo; never commit there). Also apply every follow-up listed under "Contract tooling follow-ups" in the build notes, and apply then delete any files in ${PENDING} (contract updates queued by tickets merged before yours). Make the checker accept a target backend directory so worktree agents can run it against their own tree, and document the per-ticket contract-update procedure in the conventions doc section 8 step 4. Return contractUpdates as "applied by ticket 03".`
    : `\nDo NOT edit ${ROOT}\\docs\\contracts or any tracker file. Instead return, in contractUpdates, exactly what the contract pipeline needs for your ticket (operations to register or mark implemented, request/response schemas, runtime capture expectations, DTO/interface/route/migration inventory entries). A merger applies them.`
  return `You are the implementer for ticket ${t.id} (${t.title}) of the VMS backend spec.

Read, in order: the ticket ${ticketFile(t)} (its acceptance criteria are your definition of done); the shared build notes ${NOTES} (mandatory: toolchain, env vars, forbidden actions, git layout, throwaway PostgreSQL, helpers from earlier tickets); then the sections of the spec ${SPEC}, ${ROOT}\\docs\\BACKEND-GO-CONVENTIONS.md and ${ROOT}\\docs\\contracts\\PHYSICAL-SCHEMA-2026-11-18.md that the ticket points to. Use \`git -C ${ROOT}\\backend log ${BRANCH}\` and earlier tickets' code as prior art.

Setup: create your worktree as the notes describe: path ${ROOT}\\backend-wt-${t.id}, branch ticket/${t.id}-${t.slug}, based on ${BRANCH}. If it already exists from an earlier attempt, reuse it. Confirm it is based on the current ${BRANCH} tip; if it has none of your commits and is behind, reset it onto the tip. Work only inside your worktree, plus your throwaway PostgreSQL directory (port ${port(t.id)}). Never touch the main checkout ${ROOT}\\backend.

Build test-first: call the Skill tool with "tdd" and use the test seams in the spec's Testing Decisions. Commit on your branch in focused commits ending with the trailer "${TRAILER}".${special}

Before reporting: merge the current ${BRANCH} tip into your branch, resolving conflicts so both sides' intent survives. Then in your worktree run: gofmt -l on changed files, go vet ./..., go vet -tags integration ./..., go test -count=1 ./..., and the integration suites relevant to your ticket against your throwaway cluster. Stop the cluster and delete its directory afterwards. Leave the worktree in place.

Report status "complete" only if your work is committed and that gate is green. Otherwise report "incomplete", say why, and list unmet acceptance criteria.`
}

function mergePrompt(t, impl) {
  const pipelineReady = done.has('03') || t.id === '03'
  const contractStep = t.id === '03'
    ? `Ticket 03 already updated ${ROOT}\\docs\\contracts in place. Verify that ${PENDING} is empty, and apply then delete any file still there.`
    : pipelineReady
      ? `Apply any files in ${PENDING} first (then delete them). Then apply this ticket's contractUpdates to ${ROOT}\\docs\\contracts, following the procedure in the conventions doc section 8 step 4 and ticket 03's tooling. Regenerate served OpenAPI if the pipeline requires it, and commit generated backend files on ${BRANCH}.`
      : `The contract pipeline (ticket 03) is not merged yet. Do NOT edit ${ROOT}\\docs\\contracts. Write this ticket's contractUpdates verbatim to ${PENDING}\\${t.id}.md (create the directory if needed); ticket 03 applies them.`
  const contractGate = pipelineReady
    ? `- run the contract check (python ../docs/contracts/check.py --runtime --consumers, as make contract-check)`
    : ''
  return `You are the merger for ticket ${t.id} (${t.title}) of the VMS backend spec. Read the shared build notes ${NOTES} first.

Implementer result:
${JSON.stringify(impl, null, 2)}

1. In the main checkout ${ROOT}\\backend, confirm the branch is ${BRANCH} and the tree is clean. If it is not clean, stop and report merged=false. Run: git merge --no-ff ${impl.branch} with the message "Merge ticket ${t.id}: ${t.title}" and the trailer "${TRAILER}". Resolve any conflict so both tickets' intent survives; never discard already-merged work.
2. ${contractStep}
3. Gate on ${BRANCH} in the main checkout:
- gofmt -l on changed files, go build ./..., go vet ./..., go vet -tags integration ./..., go test -count=1 ./...
- integration suites: all of them, against a throwaway cluster on port 56099 with data dir pg-scratch-merge under .runtime; stop it and delete the directory afterwards
${contractGate}
4. If the gate fails, fix the integration problem with a minimal commit on ${BRANCH}. If you cannot fix it with reasonable effort, undo the merge with git revert -m 1 <merge commit> (never reset or force) and report merged=false with the reason.
5. On success, update the ticket file ${ticketFile(t)}: set the status line to "**Status:** done (merge <short hash> on ${BRANCH})", and tick only the acceptance criteria that the implementer report and your gate verify. List any you left unticked in uncheckedCriteria.`
}

function launch(t) {
  started.add(t.id)
  log(`Starting ticket ${t.id}: ${t.title}`)
  const p = agent(implPrompt(t), { label: `implement ${t.id}`, phase: 'Implement', model: 'opus', schema: IMPL })
    .then((impl) => {
      if (!impl || impl.status !== 'complete') {
        failed.add(t.id)
        results[t.id] = { impl, merged: false }
        log(`Ticket ${t.id} implementer did not complete; dependents stay blocked`)
        return
      }
      const m = mergeChain.then(() =>
        agent(mergePrompt(t, impl), { label: `merge ${t.id}`, phase: 'Merge', model: 'opus', schema: MERGE }))
      mergeChain = m.catch(() => null)
      return m.then((merge) => {
        results[t.id] = { impl, merge }
        if (merge && merge.merged) { done.add(t.id); log(`Ticket ${t.id} merged`) }
        else { failed.add(t.id); log(`Ticket ${t.id} merge failed`) }
      })
    })
    .catch(() => { failed.add(t.id) })
  return p.then(() => t.id)
}

const running = new Map()
while (true) {
  for (const t of args.tickets) {
    if (!started.has(t.id) && t.deps.every((d) => done.has(d))) running.set(t.id, launch(t))
  }
  if (running.size === 0) break
  const finished = await Promise.race([...running.values()])
  running.delete(finished)
}

const blocked = args.tickets.filter((t) => !started.has(t.id)).map((t) => t.id)
if (blocked.length) log(`Not started (blocked by failed tickets): ${blocked.join(', ')}`)

let review = null
let fixes = null
if (done.size > args.done.length) {
  phase('Review')
  review = await agent(`Review the VMS backend integration branch. In ${ROOT}\\backend (branch ${BRANCH}), call the Skill tool with "code-review" to review all changes since commit ${args.baseline} against the spec ${SPEC}, its tickets in ${TRACKER}\\issues, and ${ROOT}\\docs\\BACKEND-GO-CONVENTIONS.md. Read the build notes ${NOTES} first. Do not edit files. Return every verified finding with severity, file, summary and the concrete fix.`,
    { label: 'code review', phase: 'Review', model: 'opus', schema: REVIEW })
  if (review && review.findings.length) {
    const fixTicket = { id: '99', slug: 'review-fixes', title: 'Code review fixes', deps: [] }
    const fixImpl = await agent(`You are the implementer for the code-review fixes on the VMS backend integration branch. Read the build notes ${NOTES}. Create worktree ${ROOT}\\backend-wt-99 on branch ticket/99-review-fixes from ${BRANCH}, and work only there. Fix every finding below, test-first where behaviour changes, and commit with the trailer "${TRAILER}". Before reporting, merge the ${BRANCH} tip and run the full gate (gofmt, vet, vet with the integration tag, go test, integration suites against a throwaway cluster on port 56098). Do not edit docs/contracts; return any contract changes in contractUpdates.

Findings:
${JSON.stringify(review.findings, null, 2)}`,
      { label: 'implement review fixes', phase: 'Review', model: 'opus', schema: IMPL })
    if (fixImpl && fixImpl.status === 'complete') {
      done.add('03')
      fixes = { impl: fixImpl, merge: await agent(mergePrompt(fixTicket, fixImpl).replace(ticketFile(fixTicket), `${TRACKER}\\spec.md (append a "Code review fixes" note under Further Notes instead of a status line)`), { label: 'merge review fixes', phase: 'Review', model: 'opus', schema: MERGE }) }
    } else {
      fixes = { impl: fixImpl, merge: null }
    }
  }
}

phase('Cleanup')
const cleanup = await agent(`Clean up after the VMS backend build. Read the build notes ${NOTES}.
1. In ${ROOT}\\backend, list the worktrees (git worktree list). Remove every ${ROOT}\\backend-wt-* worktree whose branch is fully merged into ${BRANCH} (git merge-base --is-ancestor). Keep the branches. Keep any unmerged worktree, and report it.
2. Stop and delete any leftover throwaway PostgreSQL cluster under ${ROOT}\\backend\\.runtime\\pg-scratch-*: pg_ctl stop -D <dir> -m fast, then remove the directory. Never touch the stage-1 database or Docker.
3. Report the final ${BRANCH} log (git log --oneline ${args.baseline}..${BRANCH}) and git status of the main checkout.`,
  { label: 'cleanup', phase: 'Cleanup', model: 'opus' })

return {
  merged: [...done].filter((id) => !args.done.includes(id)).sort(),
  failed: [...failed].sort(),
  blocked,
  results,
  review,
  fixes,
  cleanup,
}
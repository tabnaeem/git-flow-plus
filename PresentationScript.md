# Git Flow Plus — Presentation Script

Companion narration document for **`presentation/GitFlowPlus-Team-Training.pptx`**
(22 slides). This is the same narration embedded as speaker notes on every
slide (visible in PowerPoint's Presenter View), collected here as a
standalone script — useful for rehearsing, for a co-presenter reading
along, or for anyone who needs the talk track without opening the deck.

**Estimated total runtime:** 35–45 minutes at a conversational pace,
including the live demo (Slide 18) and Q&A buffer. Timings below are
per-slide guidance, not hard limits — slow down on Slides 5, 12, and 16;
they carry the most new vocabulary.

**Before you start:** have a throwaway Git repository ready in a
terminal window for the live demo on Slide 18, with `git-flow-plus`
already built and on `PATH`. See the demo script embedded in that
slide's command box.

---

## Opening (before Slide 1)

Welcome the room, introduce yourself, and set expectations: this is a
training session, not a sales pitch — interruptions and "wait, what if…"
questions are encouraged throughout, especially once the live demo
starts. Mention that the full command reference and architecture docs
are in the repository for follow-up after the session.

---

## Slide 1 — Welcome to Git Flow Plus *(~1.5 min)*

Welcome everyone. Today we are introducing Git Flow Plus, a purpose-built
extension of Git Flow that turns release management from a manual,
tribal-knowledge process into a structured, auditable, tool-enforced
workflow. This deck is for everyone touching a release: developers
writing features, QA validating builds, DevOps shipping them, and the
business stakeholders who need visibility into what is going out and
when. By the end of this session everyone should understand the branch
model, the version numbering scheme, and exactly which command to run at
each stage of the release lifecycle.

*Transition:* "Let's start with why this exists at all."

## Slide 2 — Traditional Git Flow Problems *(~2 min)*

Plain Git Flow was designed in 2010 for a single maintainer cutting
infrequent releases. It says nothing about who approved a feature for
release, nothing about tracking QA builds separately from production,
and nothing about a durable audit trail. Most teams that adopt it end up
bolting on spreadsheets, Slack threads, and release-manager tribal
knowledge to compensate. Git Flow Plus keeps the branch topology teams
already know, but adds the missing enterprise layer on top.

## Slide 3 — Why Git Flow Plus *(~2 min)*

Four pillars: a Release Manifest that always answers "what is in this
release," an audit trail that answers "who put it there and when,"
mandatory tagging so a QA build is never confused with a production
build, and an explicit feature state machine so nobody has to guess
whether something is ready to ship. All four are just structured data and
Git primitives — nothing proprietary, nothing that locks you in.

## Slide 4 — Our SDLC *(~2 min)*

Five stages, five clear owners. Planning is the product/BA side, not
covered by tooling. Development happens on feature branches owned by
individual developers. Integration is the one stage owned exclusively by
the Release Manager — nobody else merges into staging. QA verification
produces a tagged, reproducible build. Release is the final, one-way
step onto main. Notice there is no stage where a developer talks
directly to production — every path runs through the Release Manager.

*Transition:* "That's the map. Now let's zoom into the branches
themselves."

## Slide 5 — Branch Strategy *(~3 min — slow down here)*

This is the picture to internalize before anything else. Three permanent
branches: main is production and only ever receives a merge from release
finish. Staging is the release line — feature branches start there, and
the entire release lifecycle happens on it. Develop is intentionally off
to the side — it exists purely so a developer can integrate and
unit-test against other in-flight feature branches before asking a
Release Manager to merge into staging. It is never merged into staging
or main by tooling. Feature branches, shown dashed, start from staging
and stay alive through the whole QA cycle — they are only deleted in
bulk when release finish runs.

*Pause for questions here — this slide is the foundation everything else
builds on.*

## Slide 6 — Developer Workflow *(~2.5 min)*

From a developer's seat, the workflow is deliberately narrow. You start
a feature — that always branches from staging, not develop, and it
immediately registers in the Feature Registry as "Created." You commit
normally. If you want to integrate against other people's in-flight
feature branches to unit test, you can merge through develop, but that
is optional and never required. When you are ready, you push and open a
pull request — that is the end of your responsibility. You never merge
your own feature branch; there is no feature finish command. A Release
Manager decides if and when the feature actually ships. If QA finds an
issue after your feature has already been merged into staging, you do
not open a new branch — you commit the fix to the same feature branch,
which is still alive, and ask the Release Manager to re-run the merge
command.

## Slide 7 — Release Manager Workflow *(~2.5 min)*

The Release Manager is the only role that can move a feature across the
staging boundary. release start opens a manifest for the release and
refuses a second concurrent release outright. From there, feature list
shows every feature currently sitting in Approved state, waiting to be
pulled in. Approving is purely a planning decision — it does not touch
the repository. Add is the one command that performs a real merge: it
merges the feature branch into staging, bumps the version's feature
counter, and writes an entry into featureHistory. Because it is safe to
re-run, if a developer pushes a QA fix commit to that same branch later,
the Release Manager just runs add again — it pulls in the new commit
without incrementing the counter a second time.

## Slide 8 — QA Workflow *(~2.5 min)*

Every QA cycle starts with release build, which cuts a mandatory
annotated Git tag — that tag is the single source of truth for exactly
what QA is testing. A post-qa-tag hook can push that build to a QA
environment automatically. If QA passes, the release moves to release
finish. If QA finds a defect, the fix is never a new branch from
scratch — the developer commits directly to the already-merged feature
branch (or a release-fix branch, covered next), and the Release Manager
re-runs release feature add. That resync pulls in the fix without
double-counting the feature in the version number, then release build is
run again to cut a fresh QA tag. This loop repeats until sign-off.

## Slide 9 — Release Fix Workflow *(~2 min)*

A release fix is Git Flow Plus's answer to "QA found a bug in this
release — now what." It looks almost identical to a feature branch:
releasefix start branches from staging, just like feature start. The
difference is intent and lifecycle — a release fix is not part of the
Feature Registry, it is scoped narrowly to one defect, and unlike a
feature, any developer can run releasefix finish themselves because
there is no separate approval gate — the defect was already found during
this release's own QA process. Finishing it merges into staging
immediately, but critically, that merge alone does not move the version
number. Only release build does that, by cutting the next QA tag.

## Slide 10 — DevOps Workflow *(~2 min)*

DevOps work — pipeline changes, infrastructure-as-code updates,
deployment script tweaks — needs to ride along with a release the same
way a feature or fix does, so it gets the same treatment: a branch from
staging, tracked in the manifest, merged back before the release closes.
The part that matters most for this audience is the hook system.
post-qa-tag and post-production-tag are plain shell hooks that fire
automatically at exactly the two moments that matter — a QA tag being
cut, and a production tag being cut. That is the integration point for
whatever CI/CD system you already run; Git Flow Plus does not care
whether that is GitHub Actions, Jenkins, or something else.

*Transition:* "We've covered every role's workflow. Now let's look at
the mechanics that tie them together — starting with what a build
actually does."

## Slide 11 — Release Build *(~2 min)*

release build is deliberately the only command in the whole tool that
touches the fifth version field, the QA Build counter. Every time QA
needs a fresh, reproducible artifact to test, the Release Manager runs
this command — it tags the current state of staging with a fully
self-describing message: which release, the full version string, which
QA build number this is, and the exact list of features, release fixes,
and DevOps changes included. That tag alone is enough to answer "what
exactly did QA test in build 2" months later, without cross-referencing a
spreadsheet.

## Slide 12 — Version Numbering *(~3 min — slow down here)*

Five fields, five independent counters, five different commands. Sprint
is set once, when the release opens. Feature increments once per new
feature merged via release feature add — resyncs do not increment it
again. ReleaseFix increments per releasefix finish. DevOps increments
per devops finish. QA increments per release build, and only release
build. In the example on screen, 5.3.4.1.2 tells a complete story at a
glance: sprint 5, the third feature just landed, four release fixes have
gone in, one DevOps change, and this is the second QA build cut for this
release. Nobody has to reconstruct that story from commit messages.

## Slide 13 — Git Tag Strategy *(~2 min)*

Every single build — QA or production — leaves a permanent, annotated
tag on the exact commit that was built. This timeline shows a release
with three QA builds before sign-off, and then the final production tag,
which lands on the same version as the last successful QA build —
proving the exact commit QA signed off on is the exact commit that
shipped. Because every tag is annotated rather than lightweight, the tag
message itself carries the full audit record we saw two slides ago:
features, fixes, DevOps changes, release manager, date.

## Slide 14 — Release Manifest *(~2 min)*

release.json is the single source of truth for a release while it is
active. It is nested by category — features, release fixes, DevOps — and
within features, split by included, pending, and deferred, so anyone can
see at a glance what shipped, what is still waiting for approval, and
what was explicitly held back for a future release. The featureHistory
array at the bottom is append-only — notice the second entry for LOGIN
has resync: true, recording that a QA follow-up commit was pulled in
without incrementing the feature counter a second time. This file is
what release manifest and release feature status read from.

## Slide 15 — QA Iteration Lifecycle *(~2 min)*

This is the loop a release actually lives in for most of its life. Build
cuts a QA tag, QA tests it, and if a defect turns up, the fix goes back
to the same branch — not a new one — and the Release Manager resyncs
with the same feature add command used originally. Because add is
idempotent with respect to the feature counter, this loop can run as
many times as needed without the version number inflating in a way that
misrepresents how many actual features shipped — only the QA build
counter climbs each pass. The loop only ends one way: QA finds nothing,
and the release proceeds to finish.

## Slide 16 — Complete Release Lifecycle *(~2.5 min)*

Pulling everything together: a release starts, features and fixes land
on staging in parallel via the Release Manager, QA cycles run
build-test-resync as many times as needed, and finish is the single,
one-way door to production. Notice the three panels below — what
branches are touched, what the finish command specifically does, and —
just as important — the things that structurally cannot happen in this
model: no direct push to main, no branch disappearing mid-QA, no version
field moving from the wrong command, no feature sneaking into a release
without a Release Manager's explicit add.

## Slide 17 — Command Cheat Sheet *(~1.5 min)*

This slide is the one to screenshot. Every command in Git Flow Plus, who
runs it, and what it does, in one table — organized so you can find
"what do I run next" in a few seconds without digging through
documentation mid-release. The full detail behind each of these lives in
CommandReference.md in the repository.

*Transition:* "Enough diagrams — let's actually run it."

## Slide 18 — Live Demonstration *(~10–15 min)*

Now we switch to the terminal. We will run this exact sequence against a
throwaway repository: initialize, start a feature, have the Release
Manager merge it in, simulate a QA-reported defect via a release fix, cut
a QA build, and finish the release. After every single command we will
stop and look at release status, the manifest, and git log so everyone
can see precisely what changed and why. This is the best place to ask
"what if" questions — try to break it, ask about edge cases, we have
time built in for that.

*Run, in order, pausing after each to inspect `release status`,
`release manifest`, and `git log --oneline --decorate --tags -5`:*

```bash
git flow init
git flow feature start LOGIN
git flow release feature add LOGIN
git flow releasefix start BUG-101 && git flow releasefix finish BUG-101
git flow release build   # narrate: this is where QA would test
git flow release finish
```

## Slide 19 — Roles and Responsibilities *(~2 min)*

Four roles, four clean lanes of responsibility, with almost no overlap.
Developers own their feature branch and nothing past it. Release
Managers own everything that happens on staging — they are the single
point of control for what enters a release. QA owns sign-off — nothing
reaches release finish without their approval. DevOps owns the pipeline
and the hook wiring that makes the other three roles' work show up
automatically in real environments. Business analysts and project
managers, not shown as a card here, consume the release manifest and
status commands as a reporting surface — they do not run any of the
mutating commands themselves.

## Slide 20 — Benefits *(~2 min)*

Six concrete payoffs, and every one of them maps directly back to a
specific feature we walked through today: governance comes from the
Release Manager gate, the audit trail from featureHistory, reproducible
builds from mandatory tagging, single source of truth from the manifest,
CI/CD integration from the hook system, and zero lock-in because — worth
repeating one more time — every single thing Git Flow Plus does is a
plain, ordinary Git operation underneath.

## Final Slide — Questions / Thank You *(remaining time)*

That is the full picture — from a feature branch on a developer's laptop
to a tagged production release, with a full audit trail at every step.
Documentation for everything we covered lives in the repository. Happy
to take questions now, or follow up individually afterward. Thank you.

---

## Anticipated Q&A

A few questions that tend to come up — worth having answers ready for:

- **"What happens if two Release Managers run `release start` at the
  same time?"** — The second one fails outright
  (`ErrReleaseAlreadyActive`); Git Flow Plus is deliberately
  single-release-at-a-time (see Roadmap.md for the multi-release
  candidate on the future list).
- **"Can I rename `staging`/`develop`/`main`?"** — Yes, via
  `.gitflowplus/config.json`'s `branches` block; every command resolves
  branch names through config, never hardcodes them (see [Diagrams.md
  §20](Diagrams.md#20-branch-resolver-architecture)).
- **"What if I merge a feature and then need to un-merge it?"** — There
  is deliberately no `release feature remove`; once `add` performs a
  real merge, Git Flow Plus doesn't attempt to safely reverse it. `defer`
  covers "hold this back," but only before the first `add`.
- **"Does this replace our CI/CD pipeline?"** — No — the `post-qa-tag`
  and `post-production-tag` hooks are the integration point; Git Flow
  Plus triggers your existing pipeline, it doesn't replace it.

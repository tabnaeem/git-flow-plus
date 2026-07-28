# Git Flow Plus — Diagrams

All diagrams use [Mermaid](https://mermaid.js.org/) syntax and render
natively on GitHub, in VS Code (with the Mermaid extension), in Obsidian,
and in any standard Markdown pipeline that supports Mermaid — no external
tools needed.

This file is being built **incrementally** across five phases, per the
current documentation initiative:

- [x] **Phase 1 — Architecture diagrams**
- [x] **Phase 2 — Workflow / lifecycle diagrams** (this delivery)
- [ ] Phase 3 — Full documentation set (Architecture.md, Workflow.md,
      DeveloperGuide.md, ReleaseManagerGuide.md, QAProcess.md,
      CommandReference.md, PresentationNotes.md)
- [x] Phase 4 — PowerPoint deck (`GitFlowPlus-Team-Training.pptx`) — the
      in-deck diagrams are native PowerPoint shapes rather than rendered
      Mermaid, for print/screen quality; every one of them is backed by a
      diagram in this file covering the same lifecycle
- [ ] Phase 5 — Live demo script

Every diagram below reflects the **actual current implementation** —
verified directly against the Go source (package imports, service
interfaces, branch/prefix configuration) as of this session, not the
originally-planned design. Where the real codebase includes more than the
minimum described in the request (e.g. `hotfix`/`support` branches, three
long-lived branches rather than two), the diagrams include it, since the
goal is that these match the codebase exactly.

---

## 1. Overall Git Flow Plus Architecture

**Purpose:** the single "explain it in one picture" view — who uses the
tool, what it's built from, and what it talks to. This is the diagram for
an audience that has never seen the tool before.

```mermaid
graph TB
    subgraph ACTORS["Actors"]
        DEV["👤 Developer"]
        RM["👤 Release Manager"]
    end

    subgraph CLI["Git Flow Plus CLI<br/>(git-flow-plus binary, invoked as 'git flow ...')"]
        ROOT["Cobra Command Tree"]
    end

    subgraph CORE["Release Management Engine"]
        GITFLOW["Branching Model<br/>(internal/gitflow)"]
        RELEASE["Release Management<br/>(internal/release)"]
        VERSION["Version Engine<br/>(internal/version)"]
        FEATURE["Feature Registry<br/>(internal/feature)"]
    end

    subgraph CROSS["Cross-Cutting Concerns"]
        CONFIG["Configuration<br/>(internal/config)<br/>.gitflowplus/config.json"]
        HOOKS["Lifecycle Hooks<br/>(internal/hooks)"]
        LOG["Structured Logging<br/>(internal/logging)"]
    end

    subgraph GITLAYER["Git Abstraction"]
        GITCLIENT["Git Client<br/>(internal/git)"]
    end

    subgraph EXTERNAL["External Systems"]
        REPO[("Git Repository")]
        CICD["CI/CD Pipeline<br/>(tag-triggered, or via hooks)"]
    end

    DEV -->|"feature start, commit, push, PR"| ROOT
    RM -->|"release start / build / finish,<br/>release feature add/approve/defer,<br/>releasefix, devops"| ROOT

    ROOT --> GITFLOW
    ROOT --> RELEASE
    ROOT --> CONFIG
    ROOT --> LOG

    RELEASE --> GITFLOW
    RELEASE --> VERSION
    RELEASE --> FEATURE
    RELEASE --> HOOKS
    RELEASE --> GITCLIENT
    GITFLOW --> GITCLIENT
    VERSION --> CONFIG
    FEATURE --> CONFIG
    GITFLOW --> CONFIG

    GITCLIENT --> REPO
    HOOKS -.->|"post-qa-tag,<br/>post-production-tag"| CICD
    REPO -.->|"git push --tags"| CICD

    classDef actor fill:#7c3aed,color:#fff,stroke:#5b21b6,stroke-width:1px;
    classDef engine fill:#1f6feb,color:#fff,stroke:#0d47a1,stroke-width:1px;
    classDef cross fill:#0891b2,color:#fff,stroke:#0e7490,stroke-width:1px;
    classDef ext fill:#57606a,color:#fff,stroke:#30363d,stroke-width:1px;

    class DEV,RM actor;
    class GITFLOW,RELEASE,VERSION,FEATURE engine;
    class CONFIG,HOOKS,LOG cross;
    class REPO,CICD ext;
```

**Key takeaway:** Git Flow Plus is a thin, predictable layer over real
`git` — nothing it does is something a `git` command line couldn't also
do. Every mutation flows through `internal/git`, so the repository stays
100% compatible with GitHub, GitLab, Azure DevOps, and any Git-aware tool.

---

## 2. System Components

**Purpose:** a component-level view for engineers onboarding to the
codebase — what each Go package is *responsible for*, grouped by role.
(For the exact import graph, see [§19 Package Dependency
Diagram](#19-package-dependency-diagram) below.)

```mermaid
graph TB
    subgraph ENTRY["Entry Point"]
        MAIN["cmd/git-flow-plus<br/><i>main.go — os.Exit(cli.Execute())</i>"]
    end

    subgraph PRESENTATION["Presentation Layer"]
        CLI["internal/cli<br/><i>Cobra commands: parse args,<br/>call a service, print the result.<br/>No Git or release logic here.</i>"]
    end

    subgraph DOMAIN["Domain / Core Layer"]
        RELEASE["internal/release<br/><i>Release Manifest, QA builds,<br/>Release Planning, tagging, archiving</i>"]
        GITFLOW["internal/gitflow<br/><i>Branching model: init, feature,<br/>hotfix, support, release merge mechanics</i>"]
        VERSION["internal/version<br/><i>Sprint.Feature.Fix.DevOps.QA<br/>version numbering</i>"]
        FEATURE["internal/feature<br/><i>Feature Registry & lifecycle state</i>"]
    end

    subgraph INFRA["Infrastructure Layer"]
        GIT["internal/git<br/><i>Real git binary wrapper</i>"]
        CONFIG["internal/config<br/><i>.gitflowplus/config.json<br/>(branches, prefixes, environment)</i>"]
        HOOKS["internal/hooks<br/><i>CI/CD-agnostic lifecycle scripts</i>"]
        LOGGING["internal/logging<br/><i>Leveled, colorized, JSON-capable logs</i>"]
    end

    subgraph TOOLS["External Tooling"]
        COBRA["Cobra<br/><i>CLI framework</i>"]
        VIPER["Viper<br/><i>config parsing</i>"]
        GITBIN["git<br/><i>the real binary</i>"]
    end

    MAIN --> CLI
    CLI --> RELEASE
    CLI --> GITFLOW
    CLI --> CONFIG
    CLI --> LOGGING
    CLI --> COBRA

    RELEASE --> GITFLOW
    RELEASE --> VERSION
    RELEASE --> FEATURE
    RELEASE --> HOOKS
    RELEASE --> GIT
    GITFLOW --> GIT
    VERSION --> CONFIG
    FEATURE --> CONFIG
    GITFLOW --> CONFIG

    CONFIG --> VIPER
    GIT --> GITBIN

    classDef entry fill:#7c3aed,color:#fff;
    classDef presentation fill:#1f6feb,color:#fff;
    classDef domain fill:#2ea043,color:#fff;
    classDef infra fill:#0891b2,color:#fff;
    classDef tools fill:#57606a,color:#fff;

    class MAIN entry;
    class CLI presentation;
    class RELEASE,GITFLOW,VERSION,FEATURE domain;
    class GIT,CONFIG,HOOKS,LOGGING infra;
    class COBRA,VIPER,GITBIN tools;
```

**Key takeaway:** business logic never lives in `internal/cli` — a
command handler is always "load config → call one service method → print
the result." Every package with git-touching logic is independently
unit-testable via injected interfaces (`git.Client`, `gitflow.Service`,
config/version/feature/release `Loader`s).

---

## 19. Package Dependency Diagram

**Purpose:** the literal, verified Go import graph — every arrow below is
a real `import` statement, not an approximation. Used for architecture
review and confirming there are no import cycles (there are none; the
graph is a DAG).

```mermaid
graph LR
    subgraph GO["Git Flow Plus — Go Packages"]
        CMD["cmd/git-flow-plus"]
        CLI["internal/cli"]
        RELEASE["internal/release"]
        GITFLOW["internal/gitflow"]
        VERSION["internal/version"]
        FEATURE["internal/feature"]
        GIT["internal/git"]
        CONFIG["internal/config"]
        HOOKS["internal/hooks"]
        LOGGING["internal/logging"]
    end

    subgraph EXT["External Modules"]
        COBRA["spf13/cobra"]
        VIPER["spf13/viper"]
        TERM["golang.org/x/term"]
        SLOG["log/slog (stdlib)"]
        OSEXEC["os/exec (stdlib)"]
    end

    CMD --> CLI

    CLI --> RELEASE
    CLI --> GITFLOW
    CLI --> CONFIG
    CLI --> GIT
    CLI --> HOOKS
    CLI --> LOGGING
    CLI --> VERSION
    CLI --> FEATURE
    CLI --> COBRA
    CLI --> TERM

    RELEASE --> GITFLOW
    RELEASE --> GIT
    RELEASE --> VERSION
    RELEASE --> FEATURE
    RELEASE --> HOOKS
    RELEASE --> CONFIG

    GITFLOW --> GIT
    GITFLOW --> CONFIG

    FEATURE --> CONFIG
    VERSION --> CONFIG

    CONFIG --> VIPER
    GIT --> OSEXEC
    HOOKS --> OSEXEC
    LOGGING --> SLOG

    classDef pkg fill:#1f6feb,color:#fff;
    classDef leaf fill:#2ea043,color:#fff;
    classDef ext fill:#57606a,color:#fff;

    class CMD,CLI,RELEASE,GITFLOW pkg;
    class VERSION,FEATURE,GIT,CONFIG,HOOKS,LOGGING leaf;
    class COBRA,VIPER,TERM,SLOG,OSEXEC ext;
```

**Key takeaway:** `internal/gitflow` and `internal/git` know nothing about
release management, versions, or manifests — `internal/release` composes
on top of them, never the reverse. This is what lets `internal/gitflow`
stay a reusable, standalone "Git Flow branching engine" in principle,
independent of the release-management features built on top of it.

---

## 17. Domain-Driven Design Architecture

**Purpose:** the same system, viewed through bounded contexts rather than
Go packages — useful for reasoning about where a new business rule
belongs, independent of file layout.

```mermaid
graph TB
    subgraph RMC["Release Management Context<br/><i>Aggregate Root: Manifest</i>"]
        MANIFEST["Manifest<br/>(Release, Version, Features,<br/>ReleaseFixes, DevOps)"]
        BUILDHIST["BuildRecord<br/>(QA build history)"]
        FEATUREHIST["FeatureEvent<br/>(feature-add audit trail)"]
    end

    subgraph FMC["Feature Management Context<br/><i>Aggregate Root: Feature Registry</i>"]
        FEATUREENT["Feature<br/>(ID, Branch, MergeCommit, Release)"]
        STATE["FeatureState<br/>(Created → ... → Released → Archived)"]
    end

    subgraph VC["Versioning Context<br/><i>Value Object</i>"]
        VERSIONVO["Version<br/>(Sprint.Feature.Fix.DevOps.QA)"]
    end

    subgraph BC["Branching Context<br/><i>Domain Service</i>"]
        GFSVC["gitflow.Service<br/>(branch/merge/tag mechanics)"]
    end

    subgraph GIC["Git Infrastructure Context<br/><i>Repository Pattern</i>"]
        GITCLIENTDDD["git.Client<br/>(the one seam to the real git binary)"]
    end

    subgraph CC["Configuration Context<br/><i>Shared Kernel</i>"]
        CFGDDD["Config<br/>(Branches, Prefixes, Environment, Logging)"]
    end

    subgraph IC["Integration Context<br/><i>Anti-Corruption Layer</i>"]
        HOOKSDDD["hooks.Runner<br/>(post-qa-tag, post-production-tag)"]
    end

    subgraph PC["Presentation Context"]
        CLIDDD["CLI Commands<br/>(translate operator intent → domain calls)"]
    end

    RMC -->|"reads/writes"| FMC
    RMC -->|"reads"| VC
    RMC -->|"drives merges via"| BC
    RMC -->|"triggers"| IC
    BC -->|"uses"| GIC
    RMC -->|"uses"| GIC
    RMC -.->|"shared kernel"| CC
    FMC -.->|"shared kernel"| CC
    BC -.->|"shared kernel"| CC
    PC -->|"orchestrates"| RMC
    PC -->|"orchestrates"| BC

    classDef aggregate fill:#7c3aed,color:#fff;
    classDef entity fill:#1f6feb,color:#fff;
    classDef vo fill:#2ea043,color:#fff;
    classDef service fill:#0891b2,color:#fff;
    classDef kernel fill:#57606a,color:#fff;

    class MANIFEST aggregate;
    class BUILDHIST,FEATUREHIST,FEATUREENT,STATE entity;
    class VERSIONVO vo;
    class GFSVC,GITCLIENTDDD,HOOKSDDD service;
    class CFGDDD kernel;
```

**Key takeaway:** `Manifest` is the aggregate root for a release cycle —
everything about "what's in this release" is reached through it. The
Feature Registry is a *separate* aggregate on purpose: a feature's
lifecycle outlives any single release cycle (it can sit `Pending` across
several planning rounds), while `Manifest` is reset every cycle.

---

## 18. Clean Architecture Layers

**Purpose:** the same system again, viewed through Clean
Architecture/Hexagonal rings — what's a stable domain rule vs. a
swappable implementation detail.

```mermaid
graph TB
    subgraph L1["① Entities — pure domain data & invariants"]
        E1["version.Version<br/>+ ApplyBuild / ApplyFeature"]
        E2["feature.Feature / Registry<br/>+ Find / Upsert / Approved"]
        E3["release.Manifest / BuildRecord<br/>/ FeatureEvent"]
    end

    subgraph L2["② Use Cases — application-specific business rules"]
        U1["release.Service<br/>StartRelease, Build, FinishRelease,<br/>AddFeatureToRelease, ApproveFeature, ..."]
        U2["gitflow.Service<br/>FeatureStart, FeatureMerge,<br/>ReleaseFinish, HotfixFinish, ..."]
    end

    subgraph L3["③ Interface Adapters"]
        A1["internal/cli commands<br/>(Cobra RunE handlers)"]
        A2["config.Loader / version.Loader /<br/>feature.Loader / release.Loader<br/>(persistence adapters)"]
    end

    subgraph L4["④ Frameworks & Drivers — swappable details"]
        F1["Cobra (CLI framework)"]
        F2["Viper (JSON config parsing)"]
        F3["os/exec + the real git binary"]
        F4["OS filesystem"]
        F5["log/slog"]
    end

    L4 --> L3 --> L2 --> L1

    A1 -.->|"depends inward on"| U1
    A1 -.->|"depends inward on"| U2
    A2 -.->|"depends inward on"| E1
    A2 -.->|"depends inward on"| E2
    A2 -.->|"depends inward on"| E3
    U1 -.->|"depends inward on"| E1
    U1 -.->|"depends inward on"| E2
    U1 -.->|"depends inward on"| E3

    classDef entities fill:#7c3aed,color:#fff;
    classDef usecases fill:#1f6feb,color:#fff;
    classDef adapters fill:#0891b2,color:#fff;
    classDef frameworks fill:#57606a,color:#fff;

    class E1,E2,E3 entities;
    class U1,U2 usecases;
    class A1,A2 adapters;
    class F1,F2,F3,F4,F5 frameworks;
```

**Key takeaway:** the dependency rule holds throughout the codebase —
`internal/release`/`internal/gitflow` (Use Cases) never import
`internal/cli` (Adapters), and none of the domain code imports Cobra,
Viper, or `os/exec` directly. Everything that touches the OS or a
third-party framework is reached through an injected interface, which is
exactly what makes the fake-based unit tests possible (see
`DeveloperGuide.md`, once published).

---

## 8. Branch Relationship Diagram

**Purpose:** which branches exist, which are permanent vs. ephemeral, and
which branch spawns/merges into which. This is the diagram that makes the
"`develop` is not part of the release lifecycle" rule visually obvious.

```mermaid
graph TD
    MAIN["main<br/><b>Production</b> (permanent)"]
    STAGING["staging<br/><b>Release Branch</b> (permanent)<br/><i>the release lifecycle begins here</i>"]
    DEVELOP["develop<br/><i>Integration / unit testing only</i><br/><b>NOT part of the release lifecycle</b>"]
    FEATURE["feature/*<br/>(ephemeral — alive until release finish)"]
    RELEASEFIX["release-fix/*<br/>(ephemeral — alive until release finish)"]
    DEVOPS["release-devops/*<br/>(ephemeral — alive until release finish)"]
    HOTFIX["hotfix/*<br/>(ephemeral — deleted at hotfix finish)"]
    SUPPORT["support/*<br/>(permanent — never merged or deleted)"]

    MAIN -->|"git flow init"| STAGING
    MAIN -->|"git flow init"| DEVELOP
    MAIN -->|"support start"| SUPPORT
    MAIN -->|"hotfix start"| HOTFIX

    STAGING -->|"feature start"| FEATURE
    STAGING -->|"releasefix start"| RELEASEFIX
    STAGING -->|"devops start"| DEVOPS

    FEATURE -->|"release feature add<br/>(real merge, branch survives)"| STAGING
    RELEASEFIX -->|"releasefix finish<br/>(branch survives)"| STAGING
    DEVOPS -->|"devops finish<br/>(branch survives)"| STAGING

    HOTFIX -->|"hotfix finish"| MAIN
    HOTFIX -->|"hotfix finish"| STAGING
    HOTFIX -->|"hotfix finish"| DEVELOP

    STAGING -->|"release finish<br/>(tags v&lt;release&gt;)"| MAIN

    classDef permanent fill:#1f6feb,color:#fff,stroke:#0d47a1,stroke-width:2px;
    classDef excluded fill:#8b949e,color:#fff,stroke:#57606a,stroke-width:2px,stroke-dasharray: 5 3;
    classDef ephemeral fill:#2ea043,color:#fff,stroke:#1a7431,stroke-width:1px;
    classDef eternal fill:#7c3aed,color:#fff,stroke:#5b21b6,stroke-width:1px;

    class MAIN,STAGING permanent;
    class DEVELOP excluded;
    class FEATURE,RELEASEFIX,DEVOPS,HOTFIX ephemeral;
    class SUPPORT eternal;
```

A concrete topology, showing one full cycle end to end (illustrative
branch names use hyphens instead of slashes purely for Mermaid `gitGraph`
compatibility — the real branch names use `/`, e.g. `feature/LOGIN`):

```mermaid
gitGraph
   commit id: "initial commit"
   branch staging
   commit id: "staging created"
   checkout main
   branch develop
   commit id: "develop created (integration only)"
   checkout staging
   branch feature-LOGIN
   commit id: "LOGIN: work in progress"
   commit id: "LOGIN: push, open PR"
   checkout staging
   merge feature-LOGIN id: "release feature add LOGIN (Feature++)"
   branch releasefix-BUG101
   commit id: "fix BUG-101"
   checkout staging
   merge releasefix-BUG101 id: "releasefix finish (pending)"
   commit id: "release build" tag: "v5.3.1.0.2"
   checkout main
   merge staging id: "release finish"
   commit id: "production" tag: "v5.2"
```

**Key takeaway:** every branch that's part of the release
(`feature/*`, `release-fix/*`, `release-devops/*`) survives its merge —
none of them are deleted until `release finish` runs. `develop` never
receives a merge from `staging` or `main` through the release path at
all; the only thing that ever touches it (besides `git flow init`
creating it) is `hotfix finish`, which is a separate, production-incident
flow, not part of Release Planning.

---

## 20. Branch Resolver Architecture

**Purpose:** *how* a raw command like `git flow feature start LOGIN`
turns into an actual branch name and an actual `git` invocation — this is
the mechanism behind "branch names are configurable through
`.gitflowplus/config.json`."

```mermaid
graph LR
    JSON["<b>.gitflowplus/config.json</b><br/>{ branches: { main, staging, develop },<br/>prefixes: { feature, hotfix, support,<br/>releaseFix, devops, versionTag } }"]
    LOADER["config.Loader<br/>(Viper-backed)"]
    CFGOBJ["*config.Config<br/>(typed Go struct;<br/>falls back to config.Default()<br/>if the file doesn't exist yet)"]
    SVC["gitflow.Service /<br/>release.Service<br/>(cfg injected at construction)"]
    RESOLVE["Branch name resolution<br/>e.g. cfg.Prefixes.Feature + 'LOGIN'<br/>→ 'feature/LOGIN'"]
    GITCLIENTRESOLVE["git.Client<br/>(operates on the resolved,<br/>concrete branch name string —<br/>knows nothing about prefixes/config)"]
    REPORESOLVE[("Git Repository")]

    JSON --> LOADER
    LOADER --> CFGOBJ
    CFGOBJ --> SVC
    SVC --> RESOLVE
    RESOLVE --> GITCLIENTRESOLVE
    GITCLIENTRESOLVE --> REPORESOLVE

    classDef config fill:#0891b2,color:#fff;
    classDef domain fill:#1f6feb,color:#fff;
    classDef infra fill:#57606a,color:#fff;

    class JSON,LOADER,CFGOBJ config;
    class SVC,RESOLVE domain;
    class GITCLIENTRESOLVE,REPORESOLVE infra;
```

The same resolution, as a concrete call sequence for
`git flow feature start LOGIN`:

```mermaid
sequenceDiagram
    participant User as Developer
    participant CLI as Cobra Command<br/>(feature start)
    participant App as cli.App
    participant Loader as config.Loader
    participant GF as gitflow.Service
    participant Git as git.Client
    participant Repo as Git Repository

    User->>CLI: git flow feature start LOGIN
    CLI->>App: LoadConfig()
    App->>Loader: Load(repoRoot)
    Loader-->>App: *config.Config{Branches, Prefixes}
    App-->>CLI: cfg
    CLI->>GF: NewService(gitClient, cfg, logger)
    CLI->>GF: FeatureStart(ctx, "LOGIN")
    Note over GF: branch := cfg.Prefixes.Feature + "LOGIN"<br/>= "feature/LOGIN"<br/>base := cfg.Branches.Staging = "staging"
    GF->>Git: BranchExists(ctx, "feature/LOGIN")
    Git-->>GF: false
    GF->>Git: CreateBranch(ctx, "feature/LOGIN", "staging")
    Git->>Repo: git checkout -b feature/LOGIN staging
    Repo-->>Git: OK
    Git-->>GF: nil
    GF-->>CLI: BranchResult{Branch: "feature/LOGIN", Base: "staging"}
    CLI-->>User: Switched to a new branch "feature/LOGIN", based on "staging"
```

**Key takeaway:** `internal/git` never sees the word "feature" or
"staging" as concepts — it only ever receives fully-resolved string
branch names. All prefix/branch-name knowledge lives in `config.Config`
and is resolved exactly once, inside `internal/gitflow`/`internal/release`,
before it ever reaches the Git layer. Rename `staging` to `release` (or
`feature/` to `feat/`) in `config.json`, and every command adapts with no
code change.

---

## 3. Feature Lifecycle (State Machine)

**Purpose:** the explicit state machine backing `internal/feature` —
`State.AtLeast()` uses the ordinal values shown on each transition, so
e.g. a feature already `IncludedInRelease` still satisfies an `Approved`
check.

```mermaid
stateDiagram-v2
    [*] --> Created: feature start<br/>(auto-registration)
    Created --> InDevelopment: developer commits
    InDevelopment --> AwaitingReview: pull request opened<br/>(not automatic)
    AwaitingReview --> Approved: release feature approve
    Created --> Approved: release feature approve<br/>(review step is optional)
    Approved --> IncludedInRelease: release feature add<br/>(real merge — resync re-runs<br/>stay in this state)
    IncludedInRelease --> Released: release finish
    Released --> Archived: reserved for a future<br/>cleanup pass (not yet implemented)

    note right of Approved
        release feature defer
        only valid before the
        first `add` — once merged,
        there is no "un-merge"
    end note
```

**Key takeaway:** ordinals `0`–`6` (`Created` … `Archived`) exist purely so
`AtLeast()` can do a single integer comparison instead of an explicit
allow-list — the state machine itself has no other numeric meaning.

---

## 4. Release Lifecycle

**Purpose:** the release-manifest-centric view — what `release.json` looks
like at each stage, independent of which specific features/fixes are in
it.

```mermaid
graph LR
    NONE(["No active release"])
    STARTED["Started<br/><i>release start SPRINT</i><br/>Sprint field set,<br/>Feature/Fix/DevOps/QA = 0"]
    PLANNING["Planning<br/><i>release feature approve/defer</i><br/>candidates move Created→Approved"]
    MERGING["Merging<br/><i>release feature add,<br/>releasefix finish, devops finish</i><br/>real merges into staging"]
    BUILDING["QA Build Cut<br/><i>release build</i><br/>QA field++, tag pushed"]
    QALOOP{"QA sign-off?"}
    FINISHED["Finished<br/><i>release finish</i><br/>production tag,<br/>staging → main,<br/>branches deleted,<br/>manifest archived"]

    NONE --> STARTED --> PLANNING --> MERGING --> BUILDING --> QALOOP
    QALOOP -->|"defect found"| MERGING
    QALOOP -->|"signed off"| FINISHED
    FINISHED --> NONE

    classDef state fill:#1f6feb,color:#fff;
    classDef decision fill:#d29922,color:#000;
    classDef terminal fill:#57606a,color:#fff;
    class STARTED,PLANNING,MERGING,BUILDING state;
    class QALOOP decision;
    class NONE,FINISHED terminal;
```

**Key takeaway:** `release start` refuses a second concurrent release
(`ErrReleaseAlreadyActive`), so this whole lifecycle is a single-threaded
loop — the `QALOOP → MERGING` edge is the resync cycle from
[§15 QA Iteration Lifecycle](#15-qa-iteration-lifecycle-in-the-slide-deck),
and it can repeat any number of times before `FINISHED`.

---

## 5. QA Build Lifecycle

**Purpose:** exactly what changes on each `release build` call —
the only command that touches the version's 5th field.

```mermaid
sequenceDiagram
    participant RM as Release Manager
    participant CLI as CLI (release build)
    participant Svc as release.Service
    participant Ver as version.Version
    participant Git as git.Client
    participant Hooks as hooks.Runner

    RM->>CLI: git flow release build
    CLI->>Svc: Build(ctx)
    Svc->>Ver: ApplyBuild()<br/>(QA field += 1)
    Svc->>Git: TagCommit(ctx, "v5.3.4.1.2", richMessage)
    Note over Svc: richMessage = Release, Version,<br/>QA Build, Features, ReleaseFixes,<br/>DevOps, Branch, Commit, RM, Date
    Git-->>Svc: OK
    Svc->>Svc: append BuildRecord to manifest<br/>Manifest.Builds
    Svc->>Hooks: Run("post-qa-tag", tagCtx)
    Hooks-->>Svc: (best-effort, errors logged not fatal)
    Svc-->>CLI: BuildResult{Version, Tag}
    CLI-->>RM: "QA build v5.3.4.1.2 tagged"
```

**Key takeaway:** `Build` never merges anything — it only tags and
increments the QA counter. All merging happens earlier, via
`release feature add` / `releasefix finish` / `devops finish`.

---

## 6. Release Fix Workflow

**Purpose:** the release-fix branch's full path, contrasted with a
feature branch — it starts and ends on `staging` directly, with no
Release Manager approval gate (any developer can finish their own fix).

```mermaid
sequenceDiagram
    participant QA as QA Engineer
    participant Dev as Developer
    participant CLI as CLI
    participant GF as gitflow.Service
    participant Git as git.Client

    QA->>Dev: files defect BUG-101<br/>against active QA build
    Dev->>CLI: git flow releasefix start BUG-101
    CLI->>GF: ReleaseFixStart(ctx, "BUG-101")
    GF->>Git: CreateBranch("release-fix/BUG-101", "staging")
    Dev->>Dev: commit the fix
    Dev->>CLI: git flow releasefix finish BUG-101
    CLI->>GF: ReleaseFixFinish(ctx, "BUG-101")
    GF->>Git: Merge("release-fix/BUG-101" → "staging")
    Note over GF: branch is NOT deleted here —<br/>stays alive until release finish
    GF-->>CLI: OK
    CLI-->>Dev: "release-fix/BUG-101 merged into staging"
    Note over Dev: version is unchanged<br/>Release Manager must run<br/>release build to cut a new QA tag
```

**Key takeaway:** unlike a feature, a release fix needs no
`release feature approve`/`add` step — merging is self-service, because
the defect was already found during this release's own QA cycle. The
version only moves when `release build` runs next.

---

## 7. DevOps Workflow

**Purpose:** `devops start`/`finish` mirrors the release-fix shape
exactly, but the branch is scoped to infrastructure/pipeline changes and
tracked in a separate `devops` bucket in the manifest.

```mermaid
sequenceDiagram
    participant Ops as DevOps Engineer
    participant CLI as CLI
    participant GF as gitflow.Service
    participant Git as git.Client
    participant Manifest as release.json

    Ops->>CLI: git flow devops start CI-PIPELINE-UPDATE
    CLI->>GF: DevOpsStart(ctx, "CI-PIPELINE-UPDATE")
    GF->>Git: CreateBranch("release-devops/CI-PIPELINE-UPDATE", "staging")
    Ops->>Ops: update pipeline config, commit
    Ops->>CLI: git flow devops finish CI-PIPELINE-UPDATE
    CLI->>GF: DevOpsFinish(ctx, "CI-PIPELINE-UPDATE")
    GF->>Git: Merge("release-devops/CI-PIPELINE-UPDATE" → "staging")
    Note over GF: branch survives until release finish
    GF-->>Manifest: devops.included += "CI-PIPELINE-UPDATE"
    CLI-->>Ops: "release-devops/CI-PIPELINE-UPDATE merged"
```

**Key takeaway:** DevOps changes never bump the version by themselves —
same as a release fix, only `release build` (QA field) and
`release feature add` (Feature field) move version numbers; DevOps merges
only move the manifest's `devops.included` list.

---

## 9. Version Increment Flow

**Purpose:** which of the five `Sprint.Feature.ReleaseFix.DevOps.QA`
fields each command touches — the single most-asked question in
onboarding, made visual.

```mermaid
graph TD
    V["Version<br/>Sprint.Feature.ReleaseFix.DevOps.QA"]

    S["release start SPRINT<br/><b>sets</b> Sprint;<br/>resets Feature/ReleaseFix/DevOps/QA to 0"]
    F["release feature add<br/><b>Feature += 1</b><br/>(skipped on resync)"]
    RF["releasefix finish<br/><b>ReleaseFix += 1</b>"]
    D["devops finish<br/><b>DevOps += 1</b>"]
    Q["release build<br/><b>QA += 1</b><br/>(only command that moves this field)"]

    S -.-> V
    F -.-> V
    RF -.-> V
    D -.-> V
    Q -.-> V

    classDef cmd fill:#1f6feb,color:#fff;
    classDef ver fill:#7c3aed,color:#fff;
    class S,F,RF,D,Q cmd;
    class V ver;
```

**Key takeaway:** every field has exactly one command that can move it —
there is no path in the codebase where two fields change from a single
action, which is what makes the version string an unambiguous audit
summary on its own.

---

## 10. Release Manager Responsibilities

**Purpose:** everything gated behind the Release Manager role, grouped by
release phase.

```mermaid
graph TB
    subgraph OPEN["Opening a Release"]
        RS["release start SPRINT"]
    end
    subgraph PLAN["Planning"]
        RFL["release feature list"]
        RFA["release feature approve"]
        RFD["release feature defer"]
    end
    subgraph INTEGRATE["Integration"]
        RFAdd["release feature add<br/>(real merge + resync)"]
    end
    subgraph VERIFY["QA Verification"]
        RB["release build<br/>(cut QA tag)"]
        RSt["release status / manifest"]
    end
    subgraph CLOSE["Closing"]
        RF["release finish<br/>(prod tag, merge to main,<br/>delete branches, archive manifest)"]
    end

    OPEN --> PLAN --> INTEGRATE --> VERIFY --> CLOSE
    VERIFY -.->|"defect: back to"| INTEGRATE

    classDef rm fill:#7c3aed,color:#fff;
    class RS,RFL,RFA,RFD,RFAdd,RB,RSt,RF rm;
```

**Key takeaway:** every command in this diagram is Release-Manager-only —
a developer or QA engineer has read access (`status`/`manifest`/`list`)
but no write path into any of these boxes.

---

## 11. Developer Responsibilities

**Purpose:** the mirror image of §10 — everything a developer can do, and
the explicit boundary (there is no `feature finish`) where their
responsibility ends.

```mermaid
graph TB
    subgraph START["Starting Work"]
        FS["feature start NAME<br/>(auto-registers as Created)"]
    end
    subgraph DEV["Developing"]
        C["commit (as many as needed)"]
        DEVEL["optionally merge via `develop`<br/>to unit-test against other<br/>in-flight feature branches"]
    end
    subgraph SHIP["Shipping"]
        P["push origin feature/NAME"]
        PR["open a pull request"]
    end
    subgraph BOUNDARY["Hard boundary — developer cannot cross"]
        X["✗ no feature finish<br/>✗ cannot merge into staging<br/>✗ cannot approve their own feature"]
    end
    subgraph FOLLOWUP["QA Follow-up (same branch)"]
        FIX["commit a fix to the<br/>still-alive feature branch"]
    end

    START --> DEV --> SHIP --> BOUNDARY
    BOUNDARY -.->|"Release Manager decides"| FOLLOWUP
    FOLLOWUP -.->|"branch already merged once;<br/>RM re-runs `add` (resync)"| BOUNDARY

    classDef dev fill:#2ea043,color:#fff;
    classDef boundary fill:#8b949e,color:#fff,stroke-dasharray: 5 3;
    class FS,C,DEVEL,P,PR,FIX dev;
    class X,BOUNDARY boundary;
```

**Key takeaway:** the only two developer actions that exist *after* a
feature has been merged are "commit a fix" and "wait" — every state
transition past that point belongs to the Release Manager.

---

## 12. Build History Lifecycle

**Purpose:** `Manifest.Builds []BuildRecord` — the append-only list every
`release build` call writes to, and what happens to it at `release
finish`.

```mermaid
graph LR
    B1["BuildRecord #1<br/>QA=1, tag v5.3.4.1.1<br/>timestamp, commit SHA"]
    B2["BuildRecord #2<br/>QA=2, tag v5.3.4.1.2<br/>(after a resync)"]
    B3["BuildRecord #3<br/>QA=3, tag v5.3.4.1.3<br/>(signed off)"]
    ARCHIVE["Archived Manifest<br/><i>release finish copies the<br/>full Builds slice into the<br/>archived release record —<br/>nothing is dropped</i>"]

    B1 --> B2 --> B3 --> ARCHIVE

    classDef build fill:#d29922,color:#000;
    classDef archive fill:#57606a,color:#fff;
    class B1,B2,B3 build;
    class ARCHIVE archive;
```

**Key takeaway:** `Builds` is strictly additive for the life of a release
— nothing is ever overwritten or removed, so "how many QA cycles did
release 5.3 take" is always answerable from the archived manifest alone.

---

## 13. Manifest Lifecycle

**Purpose:** `release.json`'s own life, from creation to archival —
distinct from the release lifecycle in §4, which is about *state*; this
is about the *file*.

```mermaid
graph TD
    CREATE["release start<br/>writes fresh release.json<br/>{features:{}, releaseFixes:{}, devops:{},<br/>featureHistory:[], builds:[]}"]
    MUTATE["Every release-scoped command<br/>(feature add/approve/defer,<br/>releasefix/devops finish,<br/>release build)<br/>reads, mutates, and rewrites<br/>the same release.json"]
    ARCHIVE["release finish<br/>copies release.json into<br/>.gitflowplus/releases/5.3.json<br/>(permanent archive)"]
    RESET["Working release.json<br/>is cleared — repo returns to<br/>'no active release' state"]

    CREATE --> MUTATE --> ARCHIVE --> RESET

    classDef step fill:#1f6feb,color:#fff;
    classDef archive fill:#7c3aed,color:#fff;
    class CREATE,MUTATE step;
    class ARCHIVE,RESET archive;
```

**Key takeaway:** there is exactly one *live* `release.json` at a time
(enforced by `ErrReleaseAlreadyActive`), but an unlimited, permanent
history of *archived* ones — past releases stay queryable forever, they
just aren't the one being mutated.

---

## 14. Git Tag Lifecycle

**Purpose:** the tag timeline across a release, and the distinction
between a QA tag and the single production tag — the source diagram for
[Slide 13 of the training deck](#git-tag-strategy).

```mermaid
gitGraph
   commit id: "release start 5.3"
   branch staging
   commit id: "feature/LOGIN merged"
   commit id: "release build" tag: "v5.3.1.0.1"
   commit id: "defect fix (resync)"
   commit id: "release build" tag: "v5.3.1.0.2"
   commit id: "QA signs off"
   checkout main
   merge staging id: "release finish" tag: "v5.3.1.0.2"
```

**Key takeaway:** the production tag always lands on the exact same
version string as the last QA-signed-off build — that equality is the
proof that what QA tested is what shipped, not a separately-built
artifact.

---

## 15. Complete SDLC Flow

**Purpose:** the single end-to-end picture — every actor, every branch,
every command, one release from open to close. This is the diagram
[Slide 16 of the training deck](#complete-release-lifecycle) is built
from.

```mermaid
graph TB
    RM1["Release Manager:<br/>release start 5.3"]
    DEV1["Developers:<br/>feature start (parallel, many)"]
    PR["Developers:<br/>commit, push, open PRs"]
    RM2["Release Manager:<br/>review, approve, add<br/>(real merges into staging)"]
    RM3["Release Manager:<br/>release build<br/>(QA tag cut)"]
    QA1["QA Engineer:<br/>test the tagged build"]
    DECIDE{"Defects found?"}
    FIXPATH["Developer/DevOps:<br/>commit fix to same branch<br/>(feature / release-fix / devops)"]
    RESYNC["Release Manager:<br/>re-run add / finish<br/>(resync, no double-count)"]
    SIGNOFF["QA Engineer:<br/>sign off"]
    RM4["Release Manager:<br/>release finish<br/>(prod tag, staging→main,<br/>branch cleanup, archive)"]
    PROD(["Production"])

    RM1 --> DEV1 --> PR --> RM2 --> RM3 --> QA1 --> DECIDE
    DECIDE -->|"yes"| FIXPATH --> RESYNC --> RM3
    DECIDE -->|"no"| SIGNOFF --> RM4 --> PROD

    classDef rm fill:#7c3aed,color:#fff;
    classDef dev fill:#2ea043,color:#fff;
    classDef qa fill:#d29922,color:#000;
    classDef decision fill:#8b949e,color:#fff;
    classDef prod fill:#1f6feb,color:#fff;
    class RM1,RM2,RM3,RM4,RESYNC rm;
    class DEV1,PR,FIXPATH dev;
    class QA1,SIGNOFF qa;
    class DECIDE decision;
    class PROD prod;
```

**Key takeaway:** there is exactly one path to `Production`, and it
always passes through `release finish` — no actor, including the Release
Manager, has a shortcut around it.

---

## 16. Command Interaction Diagram

**Purpose:** which commands read vs. write each piece of persistent
state (`config.json`, `features.json`, `release.json`, Git itself) — the
diagram to check before adding a new command, to see what it should and
shouldn't touch.

```mermaid
graph LR
    subgraph CMDS["Commands"]
        INIT["init"]
        FSTART["feature start"]
        RFA2["release feature add"]
        RFXF["releasefix finish"]
        DOF["devops finish"]
        RBUILD["release build"]
        RFINISH["release finish"]
        STATUS["release status /<br/>manifest / version"]
    end

    subgraph STATE["Persistent State"]
        CFGJSON["config.json"]
        FEATJSON["features.json"]
        RELJSON["release.json"]
        GITSTATE[("Git branches, commits, tags")]
    end

    INIT -->|"write"| CFGJSON
    INIT -->|"write"| GITSTATE
    FSTART -->|"read"| CFGJSON
    FSTART -->|"write"| FEATJSON
    FSTART -->|"write"| GITSTATE
    RFA2 -->|"read"| FEATJSON
    RFA2 -->|"write"| FEATJSON
    RFA2 -->|"write"| RELJSON
    RFA2 -->|"write"| GITSTATE
    RFXF -->|"write"| RELJSON
    RFXF -->|"write"| GITSTATE
    DOF -->|"write"| RELJSON
    DOF -->|"write"| GITSTATE
    RBUILD -->|"write"| RELJSON
    RBUILD -->|"write"| GITSTATE
    RFINISH -->|"write"| FEATJSON
    RFINISH -->|"write"| RELJSON
    RFINISH -->|"write"| GITSTATE
    STATUS -->|"read only"| RELJSON

    classDef cmd fill:#1f6feb,color:#fff;
    classDef state fill:#57606a,color:#fff;
    class INIT,FSTART,RFA2,RFXF,DOF,RBUILD,RFINISH,STATUS cmd;
    class CFGJSON,FEATJSON,RELJSON,GITSTATE state;
```

**Key takeaway:** `release status`/`manifest`/`version` are the only
read-only commands in the whole tool — every other command listed here
writes to at least `GITSTATE`, which is why Git Flow Plus has no
"dry-run" mode: every mutating command's effect is a real, inspectable
Git operation, not a simulated one.

---

## What's next

All 20 originally-scoped diagrams are now delivered across Phases 1 and
2. Remaining work in the documentation initiative:

- [ ] **Phase 3** — the full documentation set (Architecture.md rewrite,
      new Workflow.md/ReleaseManagerGuide.md/QAProcess.md/
      PresentationNotes.md, CommandReference.md rewrite)
- [ ] **Phase 5** — the live demo script (a fuller, standalone document
      beyond the in-deck Slide 18 demo script and speaker notes)
- [ ] README.md rewrite with embedded architecture/workflow diagrams and
      a FAQ section

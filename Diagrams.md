# Git Flow Plus — Diagrams

All diagrams use [Mermaid](https://mermaid.js.org/) syntax and render
natively on GitHub, in VS Code (with the Mermaid extension), in Obsidian,
and in any standard Markdown pipeline that supports Mermaid — no external
tools needed.

This file is being built **incrementally** across five phases, per the
current documentation initiative:

- [x] **Phase 1 — Architecture diagrams** (this delivery)
- [ ] Phase 2 — Workflow / lifecycle diagrams
- [ ] Phase 3 — Full documentation set (Architecture.md, Workflow.md,
      DeveloperGuide.md, ReleaseManagerGuide.md, QAProcess.md,
      CommandReference.md, PresentationNotes.md)
- [ ] Phase 4 — PowerPoint deck (`GitFlowPlus-Demo.pptx`)
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
    RESOLVE["Branch name resolution<br/>e.g. cfg.Prefixes.Feature + \"LOGIN\"<br/>→ \"feature/LOGIN\""]
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

## What's next

**Phase 2 (next delivery)** covers the remaining 13 diagrams from the
original list — all lifecycle/workflow-oriented: Feature Lifecycle,
Release Lifecycle, QA Build Lifecycle, Release Fix Workflow, DevOps
Workflow, Version Increment Flow, Release Manager Responsibilities,
Developer Responsibilities, Build History Lifecycle, Manifest Lifecycle,
Git Tag Lifecycle, Complete SDLC Flow, and Command Interaction Diagram.

# bonker

A concurrent numeric password cracker written in Go with a live-updating terminal dashboard powered by [PTerm](https://github.com/pterm/pterm). Built to demonstrate goroutines, `context` cancellation, `sync.WaitGroup`, `sync/atomic` lock-free telemetry, and non-blocking `select` orchestration.

## Features

* **Partitioned Search Space:** Dynamically splits the target keyspace (`0` to `10^N - 1`) across `-w` concurrent worker goroutines.

* **Instant Broadcast Cancellation:** Uses `context.WithCancel` so the moment any worker cracks the target PIN, all remaining workers abort immediately via `<-ctx.Done()`.

* **Lock-Free Live Telemetry:** Tracks each worker's status (`Waiting`, `Working...`, `CRACKED!`, `Terminated`) and current iteration using `sync/atomic` (`atomic.Int32` and `atomic.Int64`) without mutex contention.

* **Live Terminal UI:** Renders a centered, alternating-row boxed table at 20 FPS (`50ms` ticker) via `pterm.AreaPrinter`, cleanly shutting down the UI goroutine before printing final results.

## Installation

### Prerequisites

* Go 1.22+ (uses integer range loops `for range Partitions`)

### Build from Source

```
git clone <your-repo-url>
cd pswd-breaker
go mod tidy
go build -o bonker .

```

## Usage

Run `bonker` by specifying the target numeric password (`-p`) and the number of concurrent workers (`-w`):

```
./bonker -p target-password-to-crack -w number-of-worker-to-run

```

### CLI Flags

| Flag | Type  | Default  | Description                            |     |
| ---- | ----- | -------- | -------------------------------------- | --- |
| `-p` | `int` | `999999` | Target numeric password to brute-force |     |
| `-w` | `int` | `4`      | Number of worker goroutines to spawn   |     |

## Demo

![[Pasted image 20261125224640.png|700]]

## Architecture & Concurrency Design


## Architecture & Concurrency Design

```text
               ┌──► [Worker-0] ──┐
               ├──► [Worker-1] ──┤
[main] ─(ctx)──┼──► ...        ──┼─(result)─► [main]
   │           └──► [Worker-9] ──┘              │
   │                    ▲                       │
   │                    └── cancel() on match ──┘
   │
   └─(50ms ticker)─► [UI Renderer] ─► [pterm Area]
```

### Key Primitives Used

1. **`sync/atomic` (`routineData`):** Each worker owns a dedicated slot in a pre-allocated `[]routineData` slice. Workers update their `status` and `current` fields atomically so the UI renderer can read live progress every 50ms without locking the hot search loop.

2. **`context.WithCancel`:** Checked on every iteration via a non-blocking `select` (`case <-ctx.Done(): ... default:`). When a worker finds the password, it calls `cancel()`, immediately halting all sibling workers and transitioning their status to `Terminated`.

3. **`sync.WaitGroup` + Graceful UI Teardown (`stopUI` / `uiDone`):** `main` blocks on `wg.Wait()` until every worker has recorded its final status, then closes `stopUI` and waits on `<-uiDone` so the final `CRACKED!` frame is guaranteed to render before `area.Stop()` restores the terminal.
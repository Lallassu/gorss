# RFC: High-Performance UI Navigation & Asynchronous Storage Engine

- **Author**: ngxccc
- **Status**: Active Development (Milestones 1 & 2 Completed)
- **Target Component**: `internal/controller.go`, `internal/window.go`, `internal/db.go`
- **Goal**: Sub-millisecond UI latency and zero-allocation keypress navigation in `gorss`.

---

## 1. Context & Motivation

`gorss` provides a functional TUI feed reader built on `tview` and `tcell`. However, as users accumulate large feed archives (hundreds or thousands of articles), continuous keyboard navigation (`w`/`s`, `j`/`k`, arrow keys) degrades in responsiveness.

### Empirical Benchmark Progression (AMD Ryzen 5 PRO 6650U, Go 1.25)

| Scenario / Archive Size      | Baseline (Full Rebuild) | Milestone 1 (In-Place Mutation) | Milestone 2 (Async Write Queue) | Total Speedup Factor    |
| :--------------------------- | :---------------------- | :------------------------------ | :------------------------------ | :---------------------- |
| **100 Articles - Latency**   | `540.7 μs/op`           | `94.0 μs/op`                    | **`23.3 μs/op`**                | **$23.2\times$ faster** |
| **100 Articles - Memory**    | `440.6 KB/op`           | `15.9 KB/op`                    | **`15.2 KB/op`**                | **$-96.5\%$ RAM**       |
| **100 Articles - Allocs**    | `5,460 allocs/op`       | `166 allocs/op`                 | **`152 allocs/op`**             | **$-97.2\%$ allocs**    |
| **500 Articles - Latency**   | `2,258.5 μs/op`         | `110.5 μs/op`                   | **`31.6 μs/op`**                | **$71.5\times$ faster** |
| **500 Articles - Memory**    | `1,838.3 KB/op`         | `16.0 KB/op`                    | **`15.4 KB/op`**                | **$-99.2\%$ RAM**       |
| **500 Articles - Allocs**    | `24,009 allocs/op`      | `170 allocs/op`                 | **`156 allocs/op`**             | **$-99.35\%$ allocs**   |
| **1,000 Articles - Latency** | `4,074.3 μs/op`         | `134.7 μs/op`                   | **`49.1 μs/op`**                | **$83.1\times$ faster** |
| **1,000 Articles - Memory**  | `3,538.7 KB/op`         | `16.0 KB/op`                    | **`15.4 KB/op`**                | **$-99.56\%$ RAM**      |
| **1,000 Articles - Allocs**  | `47,197 allocs/op`      | `172 allocs/op`                 | **`158 allocs/op`**             | **$-99.66\%$ allocs**   |

---

## 2. Architecture Overview

```
+-------------------------------------------------------------------------+
|                              Main UI Thread                             |
|                                                                         |
|  [Key Event (j/k)]                                                      |
|         │                                                               |
|         ▼                                                               |
|  [In-Place Row Mutation] ──► Direct TableCell.SetText() (O(1), Done)    |
|         │                                                               |
|         ├───────────────────────────────┐                               |
+─────────┼───────────────────────────────┼───────────────────────────────+
          │ (Non-blocking Channel Send)   │ (Cache Lookup / Debounce)
          ▼                               ▼
+──────────────────────────────+  +───────────────────────────────────────+
|   Background DB Worker       |  |       Preview Cache Engine            |
|                              |  |                                       |
| [Bounded Channel (chan int)] |  | [LRU Cache (Capacity: 100 entries)]   |
|         │                    |  |   - Hit: Instant render               |
|   (200ms Debounce Batch)     |  |   - Miss: Lazy Background HTML2Text   |
|         ▼                    |  +───────────────────────────────────────+
| [Batch SQLite UPDATE]        |
|  "WHERE id IN (?, ?, ...)"   |
+──────────────────────────────+
```

---

## 3. Implemented Components

### 3.1. In-Place Table Updates (`internal/window.go` & `internal/controller.go`) - COMPLETED

- `MarkArticleRowAsReadInPlace(row int, markedWeb bool)` mutates existing cells via `SetAttributes(cell.Attributes &^ tcell.AttrBold)`.

### 3.2. Asynchronous Write-Behind Queue (`internal/db.go`) - COMPLETED

- `markReadChan chan int` (buffered capacity 200).
- `startWriteWorker()` aggregates IDs in a map and flushes via `UPDATE articles SET read = true WHERE id IN (...)` every 200ms or 50 items.
- `DB.Close()` flushes in-flight queue items and closes the SQLite connection cleanly.

### 3.3. Preview Cache & Lazy Rendering (`internal/window.go`) - UPCOMING (Milestone 3)

- In-memory LRU/map cache for `html2text` plain-text conversion.

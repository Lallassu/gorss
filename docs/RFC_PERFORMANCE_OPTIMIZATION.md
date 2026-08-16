# RFC: High-Performance UI Navigation & Asynchronous Storage Engine

- **Author**: ngxccc
- **Status**: Active Development (Milestone 1 Completed)
- **Target Component**: `internal/controller.go`, `internal/window.go`, `internal/db.go`
- **Goal**: Sub-millisecond UI latency and zero-allocation keypress navigation in `gorss`.

---

## 1. Context & Motivation

`gorss` provides a functional TUI feed reader built on `tview` and `tcell`. However, as users accumulate large feed archives (hundreds or thousands of articles), continuous keyboard navigation (`w`/`s`, `j`/`k`, arrow keys) degrades in responsiveness.

### Empirical Benchmark Results (AMD Ryzen 5 PRO 6650U, Go 1.25)

| Metric / Scenario                | Baseline (Full Rebuild) | Milestone 1 (In-Place Mutation) | Improvement Factor              |
| :------------------------------- | :---------------------- | :------------------------------ | :------------------------------ |
| **100 Articles - Latency**       | `540.7 μs/op`           | **`94.0 μs/op`**                | **$5.7\times$ faster**          |
| **100 Articles - Heap Memory**   | `440.6 KB/op`           | **`15.9 KB/op`**                | **$-96.4\%$ memory allocated**  |
| **100 Articles - Allocations**   | `5,460 allocs/op`       | **`166 allocs/op`**             | **$-96.9\%$ allocations**       |
| **500 Articles - Latency**       | `2,258.5 μs/op`         | **`110.5 μs/op`**               | **$20.4\times$ faster**         |
| **500 Articles - Heap Memory**   | `1,838.3 KB/op`         | **`16.0 KB/op`**                | **$-99.1\%$ memory allocated**  |
| **500 Articles - Allocations**   | `24,009 allocs/op`      | **`170 allocs/op`**             | **$-99.3\%$ allocations**       |
| **1,000 Articles - Latency**     | `4,074.3 μs/op`         | **`134.7 μs/op`**               | **$30.2\times$ faster**         |
| **1,000 Articles - Heap Memory** | `3,538.7 KB/op`         | **`16.0 KB/op`**                | **$-99.55\%$ memory allocated** |
| **1,000 Articles - Allocations** | `47,197 allocs/op`      | **`172 allocs/op`**             | **$-99.63\%$ allocations**      |

---

## 2. Root Cause Analysis (Code References)

1. **O(N) Table Allocation Storm (`internal/controller.go:498` & `internal/window.go:660`)**:
   `SelectArticle` previously invoked `c.ShowArticles()`, which called `w.win.ClearArticles()` and re-iterated over all $N$ articles in the feed. For every single row, `AddToArticles` allocated 4 brand new `tview.TableCell` objects on the Go heap.
   - _Resolution (Milestone 1)_: Implemented `MarkArticleRowAsReadInPlace(row, markedWeb)` to mutate existing `TableCell` pointers via `SetAttributes()` with the `&^` bit-clear operator.
2. **Synchronous SQLite Disk I/O (`internal/controller.go:490` & `internal/db.go:132`)**:
   `c.db.MarkRead(a)` executes `UPDATE articles SET read = 1 WHERE id = ?` synchronously on the UI event loop thread.
3. **Synchronous HTML2Text Regex Passes (`internal/window.go:776`)**:
   `AddPreview` executes 20+ regex and string replacement passes on the raw HTML content of the article on the UI thread for each step.

---

## 3. Architecture Overview

```text
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

## 4. Milestone 2 Implementation: Asynchronous Write-Behind Queue

- Introduce `markReadChan chan int` on `DB` struct.
- Background worker goroutine debounces writes with a 200ms window:

  ```sql
  UPDATE articles SET read = 1 WHERE id IN (?, ?, ?, ...);
  ```

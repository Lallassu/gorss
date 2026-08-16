# Pull Request: Eliminate O(N) Table Allocation Storm and Defer Synchronous Disk I/O on Navigation

## 1. Summary & Problem Statement

When navigating articles rapidly using keybindings (`w`/`s`, `j`/`k`, or arrow keys) in feeds containing hundreds or thousands of articles, `gorss` experiences progressive UI latency degradation, stuttering, and visible frame drops.

Profiling and benchmarking on an AMD Ryzen 5 system demonstrates that each single article selection step previously triggered an **$O(N)$ UI table tear-down and full rebuild**, allocating tens of thousands of temporary heap objects per second and blocking the UI thread with synchronous SQLite disk transactions.

---

## 2. Benchmark Measurements (Milestone Progression)

Measured using `go test -bench=BenchmarkSelectArticle -benchmem` on **AMD Ryzen 5 PRO 6650U (Go 1.25)**:

| Archive Size       | Metric          | Baseline (Original) | Milestone 1 (In-Place Mutation) | Milestone 2 (Async Write Queue) | Total Improvement       |
| :----------------- | :-------------- | :------------------ | :------------------------------ | :------------------------------ | :---------------------- |
| **100 Articles**   | **Latency**     | `540.7 μs/op`       | `94.0 μs/op`                    | **`23.3 μs/op`**                | **$23.2\times$ faster** |
|                    | **Heap Memory** | `440.6 KB/op`       | `15.9 KB/op`                    | **`15.2 KB/op`**                | **$-96.5\%$ memory**    |
|                    | **Allocations** | `5,460 allocs/op`   | `166 allocs/op`                 | **`152 allocs/op`**             | **$-97.2\%$ allocs**    |
| **500 Articles**   | **Latency**     | `2,258.5 μs/op`     | `110.5 μs/op`                   | **`31.6 μs/op`**                | **$71.5\times$ faster** |
|                    | **Heap Memory** | `1,838.3 KB/op`     | `16.0 KB/op`                    | **`15.4 KB/op`**                | **$-99.2\%$ memory**    |
|                    | **Allocations** | `24,009 allocs/op`  | `170 allocs/op`                 | **`156 allocs/op`**             | **$-99.35\%$ allocs**   |
| **1,000 Articles** | **Latency**     | `4,074.3 μs/op`     | `134.7 μs/op`                   | **`49.1 μs/op`**                | **$83.1\times$ faster** |
|                    | **Heap Memory** | `3,538.7 KB/op`     | `16.0 KB/op`                    | **`15.4 KB/op`**                | **$-99.56\%$ memory**   |
|                    | **Allocations** | `47,197 allocs/op`  | `172 allocs/op`                 | **`158 allocs/op`**             | **$-99.66\%$ allocs**   |

### Key Takeaway

Per-keypress latency for a 1,000-article feed dropped from **$4.07\text{ ms}$** down to **$0.049\text{ ms}$ ($49\text{ μs}$)**. Holding navigation keys no longer causes GC thrashing or SQLite disk lockups.

---

## 3. Root Cause Analysis

1. **Table Allocation Storm (`internal/controller.go` & `internal/window.go`)**:
   - In `SelectArticle`, every cursor movement invoked `c.ShowArticles(c.activeFeed)` and `c.ShowFeeds()`.
   - `ShowArticles` called `w.win.ClearArticles()`, discarding all existing table cells, and iterated over all $N$ articles in the active feed, allocating new `tview.TableCell` objects for every column (Title, Feed, Date, Markers) on the heap for each row on every keystroke.
2. **Synchronous SQLite Write on Main UI Loop (`internal/db.go`)**:
   - `c.db.MarkRead(a)` executed a synchronous SQLite `UPDATE` query on the main UI event loop thread for each step, blocking keyboard input on disk I/O.
3. **Synchronous HTML2Text Regex Conversion (`internal/window.go`)**:
   - `AddPreview` executed regex passes on raw HTML payloads synchronously on the UI thread for every cursor step.

---

## 4. Proposed Solution & Architecture

1. **In-Place Table Updates (O(1) Cell Mutation)**:
   - Added `MarkArticleRowAsReadInPlace(row int, markedWeb bool)` in `internal/window.go`.
   - Directly mutates existing `TableCell` pointers in-place using `SetText()` and `SetAttributes(cell.Attributes &^ tcell.AttrBold)`.
   - Full table rebuilds (`ShowArticles`) are strictly reserved for feed switching, search query updates, and RSS network refreshes.
2. **Asynchronous Write-Behind Queue (`internal/db.go`)**:
   - Dispatches article IDs to a non-blocking buffered channel (`markReadChan chan int`).
   - A background worker debounces writes with a 200ms window / 50-item threshold and executes batched SQLite transactions:

     ```sql
     UPDATE articles SET read = true WHERE id IN (?, ?, ?, ...);
     ```

   - Added `DB.Close()` to ensure graceful flushing of in-flight updates on shutdown.
3. **Preview Caching**:
   - Caches parsed plain text to avoid redundant regex passes on revisited articles.

---

## 5. Backward Compatibility & Invariants

- **Database**: Zero schema migrations or table structure changes.
- **Configuration**: `gorss.conf`, OPML imports, theme files, and keybindings remain 100% compatible.
- **Behavior**: Unread markers, preview formatting, link opening, and selection behavior remain identical.

---

## 6. How to Verify

Run the benchmark suite:

```bash
cd internal
go test -bench=BenchmarkSelectArticle -benchmem -run=^$
```

# Requirements: Benchmark Dashboard ()

## 1. Objective
Build a clean, single-page React dashboard running on port  that lets a user create API targets, run load tests, inspect run history, and compare any two runs side-by-side.

## 2. Tech Stack
- React (scaffolded with Vite)
- Plain CSS (keep it clean, white background, high contrast—no heavy UI libraries needed)

## 3. Required UI Sections

### A. Sidebar / Left Panel: Targets
- **Create Target Form:** Inputs for , ,  (dropdown), and an optional textarea for JSON  (only shown or enabled for /).
- **Target List:** Displays all saved targets. Clicking a target selects it and loads its history. Include a delete button on each item.

### B. Main Panel (Top): Run Controller
- Displays the currently selected target's name, method, and URL.
- Numeric inputs for  (default ) and  (default ).
- **Run Benchmark button:**
  - While the request is in flight, disable the button and show a clear loading state (e.g., ).
  - When finished, automatically prepend the new run to the history table without needing a page refresh.

### C. Main Panel (Middle): Run History Table
- Displays all runs for the selected target with columns:
  - Checkbox (for selecting runs to compare)
  - Timestamp
  - Config (e.g., )
  - Avg Latency ()
  - P95 Latency ()
  - Throughput ()
  - Error Rate ()
  - Status Codes summary (e.g., )

### D. Main Panel (Bottom): Side-by-Side Comparison
- Allow the user to check **at most 2** runs in the history table.
- Once 2 runs are checked, call  and render a comparison card showing the deltas:
  - Color-code improvements in **green** (lower latency, lower error rate, higher RPS).
  - Color-code regressions in **red** (higher latency, higher error rate, lower RPS).

## 4. Implementation Hints
- Keep your API base URL () in one small helper file (e.g., ) so your components stay clean.
- Core React state you will need in :
  -  (array)
  -  (string or null)
  -  (array)
  -  (boolean)
  -  (array of up to 2 IDs for comparison)
  -  (object or null)
- Use one  on mount to fetch , and a second  watching  to fetch  (and clear  when switching targets).
- Use  watching —when , fetch the comparison automatically.

## 5. Definition of Done
- You can add a target, run two benchmarks back-to-back, see both appear in the table, check both boxes, and immediately see the green/red delta comparison.
- Errors from the backend (like an unreachable URL or offline Python engine) display cleanly in the UI instead of failing silently in the console.

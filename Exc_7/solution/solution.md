# Answers
---
### **1. Why the example fails**
In the first cell, the lake starts empty, so *run_once()* reads from the beginning (*read_new(None)*) and loads all 2,000,000 records with 0 duplicates.

However, on the second run, the job has no memory of where it stopped, so it reads from offset 0 again and copies the exact same 2,000,000 records a second time.
The lake ends up with 4,000,000 rows (2,000,000 duplicates, 0 missing) and the checker fails.
---

---
### **2. Choose architecture**
I choose **Architecture B (The lake is the truth)**
* **Why not A?** With a state file, writing the data to the lake and saving the offset in a separate file are two different steps. 
If the job crashes after writing the files but before updating the state file, the next run copies the same records again and creates duplicates. If it crashes while writing, half-written files stay visible in the lake and also get duplicated on retry. 
Saving the state first does not help either, because a crash before writing loses those records forever.

* **Why B?** There is only one source of truth, which is the lake itself. The job checks lake_position() to see where the last run stopped, writes the new Parquet files into a STAGING folder that nobody reads, and moves the finished run into `LAKE` with a single rename (publish(run)). A crash can only leave unfinished files in STAGING, which get cleaned up automatically on the next run.

* **Why not C?** Letting duplicates pile up in the lake wastes storage, slows down every query, and puts the burden on every reader to remember to use the deduplication view.
---

---
### **3. Implementation and explanation of results**
```python
def run_once():
    # 1. where did the last run stop?
    start = lake_position()
    # 2. read what is new
    new = read_new(start)
    # 3. write it to STAGING
    run = write_files(new, STAGING)
    # 4. publish: one rename moves it into the lake
    publish(run)
```

* Every run starts right where the lake ends so nothing is ever copied twice.
* If there are no new records, write_files writes nothing and publish(None) does nothing.
* A run only becomes visible to readers when the rename happens, so nothing is ever half-written in the lake.
* In the two crash tests, the unfinished files stay hidden in STAGING, the lake stays completely clean with 0 duplicates and 0 missing, and the next run cleans up STAGING and finishes the batch properly.
---

---
### **4. Output of check_all(run_once)**
| # | Situation | In lake | Duplicates | Missing | Not loaded | Run | Check |
|---|-----------|--------:|----------:|--------:|----------:|----:|:-----:|
| 1 | First run | 2,600,000 | 0 | 0 | 0 | 103.8s | PASS |
| 2 | New records arrive, run again | 2,700,000 | 0 | 0 | 0 | 15.9s | PASS |
| 3 | Nothing new, run again | 2,700,000 | 0 | 0 | 0 | 1.0s | PASS |
| 4 | Run stops WHILE writing: right after | 2,700,000 | 0 | 0 | 100,000 | — | PASS |
| 4.1 | ... then the next run | 2,800,000 | 0 | 0 | 0 | 9.4s | PASS |
| 5 | Run stops AFTER writing: right after | 2,800,000 | 0 | 0 | 100,000 | — | PASS |
| 5.1 | ... then the next run | 2,900,000 | 0 | 0 | 0 | 11.9s | PASS |
Exactly once after every run: YES

---

## Part 2 — *Line Guardian*: predictive maintenance (graded)

**Scenario.** A manufacturer runs CNC machine tools (Computer Numerical
Control: computer-controlled milling machines and lathes) and
injection-moulding machines.
When a spindle bearing fails, the line stands still for a shift. The data
science team has trained a model that recognises bearing wear in the
vibration signal tens of seconds before it becomes critical. The
maintenance team wants an alert (an anomaly event) within seconds, so that
a technician can stop the machine before the bearing breaks.

The lab plant has 60 machines that send a reading 10 times per second
(600 readings/s). The group rolls the system out to 2,000 machines at 50 Hz
(100,000 readings/s). Today the gateways write every reading into
PostgreSQL, and a detector polls the table. **Your job:** find out whether
and why that is not good enough, decide on a better architecture, and, as
an optional challenge, build it.

![Today: 60 machines (10 readings/s each) write into the PostgreSQL table readings; the anomaly model polls the table and alerts the maintenance team within seconds](images/part2-today.png)

Everything is in the folder `guardian/`:

| File | What | Change it? |
|---|---|---|
| `plant.py` | the machines and their gateways; injects bearing faults | only the class marked `TODO` (Stage C) |
| `model.py` | the data science team's model | no |
| `services.py` | the connection to PostgreSQL, the plant's system of record (machine registry) | no |
| `baseline_detector.py` | today's detector: polls PostgreSQL | no |
| `detector_service.py` | a new detector service (skeleton) | **yours** |
| `scoreboard.py` | scores a run against the ground truth | no |

Read the description at the top of `model.py` first: the model is
**stateful per machine**, and it has an inference cost per reading.

**Requirements**

| | Requirement |
|---|---|
| R1 | **Scale:** 600 readings/s in the lab; the design must grow to 100,000 readings/s. |
| R2 | **Complete and ordered:** the model gets *every* reading of a machine, in `seq` order (the model counts `missing` and `out-of-order` readings). |
| R3 | **Fresh:** anomalies at most 5 s (p95) after the reading, at full rate. |
| R4 | **Independent consumers:** the raw readings also feed a training-data archive and a live dashboard. Adding a consumer must not slow down, change or break the others, and must not add load to PostgreSQL. |

Do not change the files marked "no", and do not read `ground_truth.jsonl`
in your code (in real life, it is only written after the repair).

> Every terminal of Part 2 needs the virtual environment of Part 0. In
> `guardian/`: `source ../venv/bin/activate` (Windows:
> `..\venv\Scripts\activate`).

### Stage A — Today's system

What the two commands of step 1 start: the plant's four line gateways
write the readings of their machines into the table `readings`, and the
detector polls that table for new rows.

![Stage A setup: in terminal 2, plant.py simulates 20 machines (10 Hz each, first fault at 40 s); four line gateways write their machines' readings into the PostgreSQL table readings, each batch with INSERT ... COMMIT; the id is a BIGSERIAL (1, 2, 3, ...). In terminal 1, baseline_detector.py polls the table in a loop: SELECT ... WHERE id > last_id, scores every row with the model (3 ms each), and remembers last_id, the highest id read so far. Afterwards, in terminal 3, you count the rows with SELECT count(*) and compare with what the detector read](images/part2-stage-a-setup.png)

1. In two terminals in `guardian/`, start the detector first, then a small
   plant:

   ```bash
   python3 baseline_detector.py
   python3 plant.py --sink postgres --machines 20
   ```

   After ~90 s (the first fault starts at 40 s), stop the plant, wait until
   the detector's backlog is 0, and stop the detector (`Ctrl+C`). Compare the
   number of readings the detector read with the table:

   ```bash
   docker exec -it postgres psql -U postgres -d factory \
     -c "SELECT count(*) FROM readings WHERE run = (SELECT max(run) FROM readings);"
   ```
---
   Number of readings the detector read: 
   - read   21,008  backlog       0 rows  delay   0.0 s  missing  3,387  anomalies 491

   Number of reading from the table:
   - count: 24400

---

   The detector keeps up, yet `missing` grows. Are the missing readings in
   the table? Why did the detector never see them? Read `PostgresSink` in
   `plant.py`, then think through this timeline: gateway 1 inserts rows that
   get the ids 100–114 and commits 100 ms later; gateway 2 inserts rows
   115–129 and commits at once. A row gets its `id` when it is inserted,
   but others see it only when it is committed. Which rows does the
   detector see when it polls in between, and which `id` does it remember?
---
**Answer**
Yes, the missing readings are in the table: 24.400 rows in the table - 21.008 read, that means 3.392 rows never read, is similar to the 3387 missing counted by the model.

A row gets its **id** at INSERT, but becomes visible only at COMMIT and each of the 4 gateways commits after a random delay. So the id order ≠ commit order.
Detector uses 'WHERE id > last_id' as a mark.

Gateway 1 inserts 100-114, gateway 2 inserts 115-129 and commits at once. If the detector polls now, it sees only 115-129 (100-114 are invisible to it) and remembers last_id = 129. When gateway 1 commits 100ms later, rows from 100-114 have lower id than 129, so the next 'WHERE id > 129' never returns them. They stay in the table forever, but the model never receives them, they become missing.
---

2. Start the detector again, then the full plant
   (`python3 plant.py --sink postgres`), and let both run for ~60 s. Watch
   the detector's lines **while the plant runs**: what happens to `delay`,
   `backlog` and `missing`, and why? Stop the detector right after the
   plant: afterwards it only works through its backlog.

---
After 60 seconds
**Detector**: stopped: read 17,059 readings, last id 58,650, 46 readings missing, 0 anomalies
**Plant**: run 1793879256: sent 36,540 readings in 61 s

- **Backlog**: increments constantly without stopping. Model needs 3ms per reading, one detector handles around 280 rows/s (17059 / 61 = 280 rows/s) but the plant produces 600 rows/s
- **Delay**: is incrementing around 2.4s every 5s. The detector falls further behind in real time. The anomaly alert would come far too late.
- **Missing**: starts at 0 and then goes to 46 and stays in there all the time. At the beginning, the detector was at the head, where uncommited rows are. Once it fell behind, it starts reading old rows, after 61s it had only processed the first 28s of data, before the fault started.

- **Conclusion**: with this design, the detector is fast and loses rows or it is overloaded and late. One detector polling one table cannot fix both. So it violates the requirements.
---
### Stage B — Decide

You may discuss the options with a neighbour, but write your own decision
record.

Compare these two options against R1–R4 (add your own idea if you have
one):

1. keep PostgreSQL, but poll it better (e.g. several pollers, each for a
   part of the machines);
2. the gateways write every reading into Kafka (or another log-based
   platform) instead of into PostgreSQL.

For each option: which requirements does it meet, which are at risk? Then
decide, and estimate for your choice: how many detector instances does
the model (3 ms per reading) need for 600 readings/s, and for 100,000? A
small office that logs one temperature reading per minute is perfectly
served by option 1: where, roughly, is the tipping point?

Write it down as a short decision record (half a page: options, decision,
reasons, numbers). PostgreSQL stays the system of record for the machine
registry in any case.

---
**Option 1:** Keep PostgreSQL but poll it better 
- R1: Scale (600 to 100.000 readings): **at high risk**, 100.000 inserts/s into one only table plus the pollers reads on the same DB.

- R2: Complete and ordered: **at risk**, id order ≠ commit order. Fixable but fixing it requires extra complex code, and waiting for late commits adds delay.

- R3: Freshness: **possible**, poll interval plus safety margin add latency, real limit is the number of detectors.

- R4: Independent consumers: **Not met**, every consumer polls PostgreSQL = extra load, and each must track which rows they already read.

**Option 2:** Kafka 
- R1: Scale (600 to 100.000 readings): **met**, handles high data rates easily and scales adding partitions.

- R2: Complete and ordered: **met**, same key --> same partition --> same order. Offsets are only visible once written.

- R3: Freshness: **met**, consumer receives events in ms, freshness depends only on enough detector instances.

- R4: Independent consumers: **met**, each consumer has its own group, reads without touching others.

**Decision:** 
Option 2. Kafka is the transport and buffer for the readings, while PostgreSQL stays the system of record for the machine registry only.

**Numbers (takes 3ms per reading)**
If it takes around 3ms per reading, one detector can handle around 333 readings/s in theory.
- **Lab setup (600 readings/s):** 600*0.003 = 1.8 instances running at full capacity. This means 2 is the minimum and 4 gives us a safe buffer.
- Full rollout (100.000 readings/s): 100.000 * 0.003 = 300 instances running at full capacity. 300 is the minimum and 450 gives us a safe buffer.

Kafka allows at most one active detector per partition within a group, we always need at least as many partitions as detector instances.
We should create the topic with at least 8 partitions for the lab and 500 for production.


**Tipping point**
A small office logging one temperature reading per minute works completely fine with PostgreSQL. Tipping point where a regular database stops being enough isn't just about speed, but a mix of things happening at once:
- Multiple gateways writing data at the same time (causing order and missing row issues).
- Needing alerts within just a few seconds.
- Having a model that takes longer to process data than how fast it arrives.
- Needing separate apps to read the same data without slowing down the database.

This Tipping point happens around a few hundred to thousand readings per second, or as soon as you add a second app that shouldn't overload the database. 
Our lab is already past the point where a single database poller can keep up.

If I had chosen option 1: I would have needed one reader per machine, plus a separate path for the archive and dashboard so the wouldn't overload PostgreSQL.
---

### Stage C — Challenge: Build the core (Optional)

The group already operates Kafka (it is in `docker-compose.yml`), so you
build option 2. If you decided otherwise in Stage B, that is fine: say in
your decision record what would be different. Run every long-running
process in its own terminal in `guardian/`.

![Stage C: plant.py (KafkaSink) writes to your topic; N instances of detector_service.py read it and publish to the topic anomalies; scoreboard.py checks R2 and R3. Under each part, the decision it needs: which key and settings, how many partitions, how many instances, which group and where to start reading, which key for the anomalies, and whether you can explain a FAIL](images/part2-stage-c.png)

1. **Gateways.** Create the topic for the readings with `kafka-topics.sh`
   (Part 1, step 1), implement `KafkaSink` in `plant.py`, and start the plant:
   `python3 plant.py --sink kafka`.
2. **Detector.** Complete the `TODO`s in `detector_service.py` and start as
   many instances as you estimated. The detector publishes its anomaly
   events to the topic **`anomalies`**: the maintenance team and the
   scoreboard read them from there.
3. After ~3 minutes (two faults), check:

   ```bash
   python3 scoreboard.py
   ```

   If something fails, find out why with `kafka-topics.sh
   --describe`, `kafka-consumer-groups.sh --describe` (Part 1, step 4) and the
   Kafka UI, and fix it.


**Topics**
```bash
docker exec -it kafka kafka-topics.sh --bootstrap-server localhost:9092 --create --topic readings --partitions 8 --replication-factor 1
```

```bash
docker exec -it kafka kafka-topics.sh --bootstrap-server localhost:9092 --create --topic anomalies --partitions 3 --replication-factor 1
```

```bash
docker exec -it kafka kafka-topics.sh --bootstrap-server localhost:9092 --describe --topic readings
```

After 3 minutes we look at scoreboard.py and we have:
- [PASS] R2 complete and ordered: 0 readings missing, 0 out of order: None of the instances missed data.
- [PASS] R3 fresh: p95 latency < 5 s: the latency was 0.08s is way lower than the limit that is 5s
- [PASS] detection: all faults found, at most 2 false alarms: detected 3 real faults, in 7.6s when it started.

det-1 --> 15 machines (27.858 readings)
det-2 --> 14 machines (26.002 readings)
det-3 --> 14 machines (26.002 readings)
det-4 --> 17 machines (31.568 readings)


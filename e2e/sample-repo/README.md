# widget-service

A tiny service that prices widget orders. It is intentionally small and is used
as a fixture repo for the orchestrator's end-to-end suite: an agent is pointed at
this directory and asked questions it can only answer by reading the code.

## Layout

- `pricing.py` — the base price and the total-price calculation.
- `inventory.py` — in-memory stock levels.

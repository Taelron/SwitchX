-- Sleeps long enough that two concurrent runners actually contend
-- for the advisory lock. The second runner waits, then proceeds.
SELECT pg_sleep(2);

CREATE TABLE IF NOT EXISTS switchx_test_slow_marker (
    id integer PRIMARY KEY
);

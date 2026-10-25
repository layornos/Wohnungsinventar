-- Spec 08: price reminder and give-away reminder as two more kinds of reminder. SQLite can't change a CHECK, so
-- reminder_schedule and reminder are rebuilt (foreign keys are off during migrations).

-- When the asking price last changed; the price reminder counts from there or from offered_since, whichever is later.
ALTER TABLE item ADD COLUMN price_changed_at TEXT;
UPDATE item SET price_changed_at = offered_since WHERE asking_price_cents IS NOT NULL;

CREATE TABLE reminder_schedule_new (
    item_id INTEGER NOT NULL REFERENCES item (id) ON DELETE CASCADE,
    kind    TEXT NOT NULL CHECK (kind IN ('keep_review', 'sell_later', 'price_reminder', 'give_away_reminder')),
    due_on  TEXT NOT NULL,
    PRIMARY KEY (item_id, kind)
);
INSERT INTO reminder_schedule_new (item_id, kind, due_on) SELECT item_id, kind, due_on FROM reminder_schedule;
DROP TABLE reminder_schedule;
ALTER TABLE reminder_schedule_new RENAME TO reminder_schedule;
CREATE INDEX reminder_schedule_due ON reminder_schedule (due_on);

CREATE TABLE reminder_new (
    id           INTEGER PRIMARY KEY,
    item_id      INTEGER NOT NULL REFERENCES item (id) ON DELETE CASCADE,
    kind         TEXT NOT NULL CHECK (kind IN ('keep_review', 'sell_later', 'price_reminder', 'give_away_reminder')),
    due_on       TEXT NOT NULL,
    triggered_at TEXT NOT NULL,
    done_at      TEXT,
    done_by      INTEGER REFERENCES person (id) ON DELETE SET NULL,
    reaction     TEXT
);
INSERT INTO reminder_new SELECT id, item_id, kind, due_on, triggered_at, done_at, done_by, reaction FROM reminder;
DROP TABLE reminder;
ALTER TABLE reminder_new RENAME TO reminder;
CREATE INDEX reminder_item ON reminder (item_id);
CREATE INDEX reminder_open ON reminder (done_at);

-- Migration 0011 set disposition_since but planned no reminders for the items that already were kept or to be sold
-- later; they are planned here from that date. Dates are taken in UTC, which is at most a day off from the
-- household's time zone. February 29 becomes February 28 as in the code. Items in the archive get none.
INSERT OR IGNORE INTO reminder_schedule (item_id, kind, due_on)
    SELECT id, 'keep_review',
        date(disposition_since, CASE WHEN strftime('%m-%d', disposition_since) = '02-29' THEN '-1 day' ELSE '+0 days' END, '+3 years')
    FROM item
    WHERE disposition = 'keep' AND handed_over_on IS NULL AND disposition_since IS NOT NULL;
INSERT OR IGNORE INTO reminder_schedule (item_id, kind, due_on)
    SELECT id, 'sell_later',
        date(disposition_since, CASE WHEN strftime('%m-%d', disposition_since) = '02-29' THEN '-1 day' ELSE '+0 days' END, '+1 year')
    FROM item
    WHERE disposition = 'sell_later' AND handed_over_on IS NULL AND disposition_since IS NOT NULL;

-- The new kinds for the items already on offer.
INSERT OR IGNORE INTO reminder_schedule (item_id, kind, due_on)
    SELECT id, 'price_reminder', date(MAX(offered_since, price_changed_at), '+30 days') FROM item
    WHERE disposition = 'sell' AND asking_price_cents IS NOT NULL AND handed_over_on IS NULL AND offered_since IS NOT NULL;
INSERT OR IGNORE INTO reminder_schedule (item_id, kind, due_on)
    SELECT id, 'give_away_reminder', date(offered_since, '+30 days') FROM item
    WHERE disposition = 'give_away' AND handed_over_on IS NULL AND offered_since IS NOT NULL;

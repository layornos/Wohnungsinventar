-- Spec 07 (reminders). Dates (due_on, month) are days in the household's time zone.

-- When the disposition last changed; the periods of keep review and sell later count from there.
ALTER TABLE item ADD COLUMN disposition_since TEXT;
UPDATE item SET disposition_since = updated_at;

-- Whether a user gets the monthly reminder mail.
ALTER TABLE app_user ADD COLUMN reminder_mail INTEGER NOT NULL DEFAULT 1 CHECK (reminder_mail IN (0, 1));

-- The next planned due date of a reminder for an item, at most one per item and kind.
CREATE TABLE reminder_schedule (
    item_id INTEGER NOT NULL REFERENCES item (id) ON DELETE CASCADE,
    kind    TEXT NOT NULL CHECK (kind IN ('keep_review', 'sell_later')),
    due_on  TEXT NOT NULL,
    PRIMARY KEY (item_id, kind)
);

CREATE INDEX reminder_schedule_due ON reminder_schedule (due_on);

-- A reminder that became due. It applies to the whole household: whoever reacts settles it for everybody.
CREATE TABLE reminder (
    id           INTEGER PRIMARY KEY,
    item_id      INTEGER NOT NULL REFERENCES item (id) ON DELETE CASCADE,
    kind         TEXT NOT NULL CHECK (kind IN ('keep_review', 'sell_later')),
    due_on       TEXT NOT NULL,
    triggered_at TEXT NOT NULL,
    done_at      TEXT,
    -- Who reacted; NULL when the reminder was settled automatically, e.g. by a change of the disposition.
    done_by      INTEGER REFERENCES person (id) ON DELETE SET NULL,
    reaction     TEXT
);

CREATE INDEX reminder_item ON reminder (item_id);
CREATE INDEX reminder_open ON reminder (done_at);

-- The monthly mail to a user (month as YYYY-MM). A month without open reminders is recorded as 'empty', so that
-- reminders becoming due later in that month wait for the next month.
CREATE TABLE monthly_mail (
    person_id  INTEGER NOT NULL REFERENCES person (id) ON DELETE CASCADE,
    month      TEXT NOT NULL,
    state      TEXT NOT NULL CHECK (state IN ('pending', 'sent', 'failed', 'empty')),
    attempts   INTEGER NOT NULL DEFAULT 0,
    last_error TEXT,
    sent_at    TEXT,
    PRIMARY KEY (person_id, month)
);

-- The reminders a monthly mail contains.
CREATE TABLE monthly_mail_reminder (
    person_id   INTEGER NOT NULL,
    month       TEXT NOT NULL,
    reminder_id INTEGER NOT NULL REFERENCES reminder (id) ON DELETE CASCADE,
    PRIMARY KEY (person_id, month, reminder_id),
    FOREIGN KEY (person_id, month) REFERENCES monthly_mail (person_id, month) ON DELETE CASCADE
);

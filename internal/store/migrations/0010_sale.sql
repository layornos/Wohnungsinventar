-- Spec 06 (sale): asking price, when an item went on offer, and the archive of sold and given away items.
-- Amounts are whole cents. handed_over_on is a date (YYYY-MM-DD) in the household's time zone; while it is set,
-- the item is in the archive and sale_proceeds_cents is set too.
ALTER TABLE item ADD COLUMN asking_price_cents INTEGER CHECK (asking_price_cents >= 0);
ALTER TABLE item ADD COLUMN offered_since TEXT;
ALTER TABLE item ADD COLUMN handed_over_on TEXT;
ALTER TABLE item ADD COLUMN sale_proceeds_cents INTEGER CHECK (sale_proceeds_cents >= 0);

-- Items already for sale or to give away count as offered since their last change.
UPDATE item SET offered_since = updated_at WHERE disposition IN ('sell', 'give_away');

CREATE INDEX item_handed_over_on ON item (handed_over_on);

-- Settings of the one household as key and value, e.g. the listing template; missing keys use the defaults in code.
CREATE TABLE household_setting (
    key   TEXT PRIMARY KEY,
    value TEXT NOT NULL
);

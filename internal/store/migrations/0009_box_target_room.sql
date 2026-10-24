-- Optional target room of a box (in the new apartment); items in the box without a target room
-- of their own go there.
ALTER TABLE box ADD COLUMN target_room_id INTEGER REFERENCES room (id);

CREATE INDEX box_target_room ON box (target_room_id);

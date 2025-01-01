CREATE TABLE IF NOT EXISTS pages_closures (
    ancestor_id uuid NOT NULL,
    descendant_id uuid NOT NULL,
    PRIMARY KEY (ancestor_id, descendant_id),
    FOREIGN KEY (ancestor_id) REFERENCES pages(id),
    FOREIGN KEY (descendant_id) REFERENCES pages(id),
    CONSTRAINT ancestor_descendant_no_cycle_constraint CHECK (LEAST(ancestor_id, descendant_id) <> GREATEST(ancestor_id, descendant_id))
    is_parent boolean NOT NULL DEFAULT false
);

CREATE INDEX IF NOT EXISTS idx_pages_closures_ancestor_id ON pages_closures (ancestor_id);
CREATE INDEX IF NOT EXISTS idx_pages_closures_descendant_id ON pages_closures (descendant_id);

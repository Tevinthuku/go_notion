ALTER TABLE pages_closures 
    DROP CONSTRAINT pages_closures_ancestor_id_fkey,
    DROP CONSTRAINT pages_closures_descendant_id_fkey;

ALTER TABLE pages_closures
    ADD CONSTRAINT pages_closures_ancestor_id_fkey
        FOREIGN KEY (ancestor_id) REFERENCES pages(id) ON DELETE CASCADE,
    ADD CONSTRAINT pages_closures_descendant_id_fkey
        FOREIGN KEY (descendant_id) REFERENCES pages(id) ON DELETE CASCADE;

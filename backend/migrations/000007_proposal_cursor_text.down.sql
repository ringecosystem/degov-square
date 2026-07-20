DO $$
BEGIN
    IF EXISTS (
        SELECT 1
        FROM dgv_dao
        WHERE length(last_tracked_proposal_id) > 255
    ) THEN
        RAISE EXCEPTION 'cannot narrow last_tracked_proposal_id to varchar(255): values longer than 255 characters exist';
    END IF;
END;
$$;

ALTER TABLE dgv_dao ALTER COLUMN last_tracked_proposal_id TYPE varchar(255);

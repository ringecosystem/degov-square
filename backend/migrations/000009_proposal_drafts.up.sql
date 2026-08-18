CREATE TABLE dgv_proposal_draft (
    id varchar(50) PRIMARY KEY,
    client_request_id varchar(100) NOT NULL,
    dao_code varchar(255) NOT NULL,
    chain_id bigint NOT NULL,
    user_id varchar(50) NOT NULL,
    user_address varchar(255) NOT NULL,
    title varchar(200) NOT NULL,
    payload jsonb NOT NULL,
    payload_version integer NOT NULL,
    revision integer NOT NULL DEFAULT 1,
    ctime timestamptz NOT NULL DEFAULT now(),
    utime timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT chk_dgv_proposal_draft_payload_version
        CHECK (payload_version = 1),
    CONSTRAINT chk_dgv_proposal_draft_revision
        CHECK (revision > 0),
    CONSTRAINT uq_dgv_proposal_draft_request
        UNIQUE (user_id, dao_code, client_request_id)
);

CREATE INDEX idx_dgv_proposal_draft_owner_updated
    ON dgv_proposal_draft (user_id, dao_code, utime DESC, id DESC);

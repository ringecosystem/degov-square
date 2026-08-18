CREATE TABLE dgv_proposal_comment (
    id varchar(50) PRIMARY KEY,
    dao_code varchar(255) NOT NULL,
    chain_id bigint NOT NULL,
    proposal_id varchar(78) NOT NULL,
    user_id varchar(50) NOT NULL,
    user_address varchar(255) NOT NULL,
    reply_to_id varchar(50),
    body text NOT NULL,
    state varchar(20) NOT NULL DEFAULT 'ACTIVE',
    ctime timestamptz NOT NULL DEFAULT now(),
    utime timestamptz,
    CONSTRAINT chk_dgv_proposal_comment_state
        CHECK (state IN ('ACTIVE', 'DELETED')),
    CONSTRAINT fk_dgv_proposal_comment_reply
        FOREIGN KEY (reply_to_id) REFERENCES dgv_proposal_comment(id)
);

CREATE INDEX idx_dgv_proposal_comment_thread
    ON dgv_proposal_comment (dao_code, proposal_id, ctime, id);

CREATE INDEX idx_dgv_proposal_comment_reply
    ON dgv_proposal_comment (reply_to_id);

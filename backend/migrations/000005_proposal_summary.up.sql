-- Migration: Add proposal_summary table for AI-generated proposal summaries
-- This table stores AI-generated summaries of proposals

create table
  if not exists dgv_proposal_summary (
    id varchar(50) not null,
    dao_code varchar(255),
    chain_id int not null,
    proposal_id varchar(255) not null,
    indexer varchar(255),
    description text not null,
    summary text not null,
    ctime timestamp default now (),
    utime timestamp,
    primary key (id)
  );

-- Index for efficient lookups by proposal_id and chain_id
create unique index uq_dgv_proposal_summary_proposal_chain on dgv_proposal_summary (proposal_id, chain_id);

comment on table dgv_proposal_summary is 'AI-generated proposal summaries';
comment on column dgv_proposal_summary.dao_code is 'DAO code';
comment on column dgv_proposal_summary.chain_id is 'Chain ID';
comment on column dgv_proposal_summary.proposal_id is 'Proposal ID';
comment on column dgv_proposal_summary.indexer is 'Indexer endpoint used to fetch proposal data';
comment on column dgv_proposal_summary.description is 'Original proposal description';
comment on column dgv_proposal_summary.summary is 'AI-generated summary';

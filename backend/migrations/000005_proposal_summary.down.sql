-- Rollback: Drop proposal_summary table

drop index if exists uq_dgv_proposal_summary_proposal_chain;
drop table if exists dgv_proposal_summary;

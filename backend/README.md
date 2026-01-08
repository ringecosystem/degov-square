# DeGov Apps Backend

## Background Tasks

### Proposal Fulfill Task

The `proposal-fulfill` task is responsible for AI-based voting on governance proposals. It analyzes on-chain voting data and casts votes on behalf of the DeGov agent.

#### Flow Diagram

```
ListUnfulfilledProposals (query: state=Active, fulfilled=0, fulfill_errored=0)
       │
       ▼
filterReadyProposals
       │
       ▼
┌──────────────────────┐
│ Has delegators?      │
│ (other than self)    │
└────────┬─────────────┘
      Yes│              No
         ▼               ▼
  ┌──────────────────┐  MarkFulfillNoDelegators
  │ Voting ended?    │  (fulfill_errored=1)
  └────────┬─────────┘
        Yes│         No
           ▼          ▼
  MarkFulfillExpired   ┌─────────────────┐
  (fulfill_errored=1)  │ Past midpoint?  │
                       └────────┬────────┘
                            Yes │         No
                                ▼          ▼
                          Add to ready    Skip
                          list & vote     (wait for next cycle)
```

#### Logic Explanation

1. **No delegators** → Mark as skipped (`fulfill_errored=1`), will not vote without delegation
2. **Voting ended** → Mark as expired (`fulfill_errored=1`), will not be queried again
3. **Voting ongoing + Past midpoint** → Add to ready list, execute AI analysis and cast vote
4. **Voting ongoing + Before midpoint** → Skip, wait for next task execution cycle

#### Delegation Check

The agent only votes when it has delegators other than itself. This ensures:
- The agent represents actual community delegation
- No self-voting without real voting power from others
- Queries the indexer for `delegates` where `toDelegate = agentAddress` and `fromDelegate != agentAddress`

#### Why Vote at Midpoint?

The task waits until the voting period midpoint before casting a vote. This allows:
- Sufficient time for community members to vote and provide reasons
- More voting data for AI analysis to make informed decisions
- Avoiding premature voting that could influence other voters

#### Configuration

```env
# Enable/disable the task
TASK_PROPOSAL_FULFILL_ENABLED=false
TASK_PROPOSAL_FULFILL_INTERVAL=30s

# OpenRouter AI Configuration
OPENROUTER_API_KEY=sk-or-v1-...
OPENROUTER_MODEL=google/gemini-2.0-flash-001

# DeGov Agent Private Key (for on-chain voting)
DEGOV_AGENT_PRIVATE_KEY=0x...

# Gas buffer percentage for voting transactions (default: 20)
# Adjusts estimated gas by adding this percentage as a buffer
DEGOV_AGENT_GAS_BUFFER_PERCENT=20
```

#### Database Fields

The following fields in `dgv_proposal_tracking` table are used for fulfill tracking:

| Field | Type | Description |
|-------|------|-------------|
| `fulfilled` | int | 0: not fulfilled, 1: fulfilled |
| `fulfilled_explain` | text | AI decision explanation JSON |
| `fulfilled_at` | timestamp | Time when fulfilled |
| `times_fulfill` | int | Number of fulfill attempts |
| `fulfill_errored` | int | 0: no error, 1: errored/expired |

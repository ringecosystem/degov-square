{{template "layout.md" .}}

{{define "content"}}
{{$proposalDb := .Proposal.ProposalDb}}
{{$proposalIndexer := .Proposal.ProposalIndexer}}
{{$dao := .Dao}}
{{$voteIndexer := .Vote.VoteIndexer}}
{{$payload := .PayloadData}}

Hello {{if .EnsName}}{{.EnsName}}{{else}}{{.UserAddress}}{{end}},

A new vote has been cast on the proposal "**{{$proposalDb.Title}}**" in {{$dao.Name}}.

- **Proposal:** [{{$proposalDb.Title}}]({{$proposalDb.ProposalLink}})
- **Voter:** {{if $payload.VoterEnsName}}{{$payload.VoterEnsName}}{{else}}{{$voteIndexer.Voter}}{{end}}
- **Vote Direction:** {{if eq $voteIndexer.Support 1}}✅ For{{else if eq $voteIndexer.Support 0}}❌ Against{{else}}⚪️ Abstain{{end}}
- **Voting Power:** {{(formatBigIntWithDecimals $voteIndexer.Weight $payload.DecimalsInt) | formatLargeNumber}}

---

### Quick Links

- [View This Vote]({{index .DaoConfig.Chain.Explorers 0}}/tx/{{$voteIndexer.TransactionHash}})
- [View Proposal Details]({{$proposalDb.ProposalLink}})

---

Thank you for staying engaged with {{$dao.Name}} onchain governance!

Best regards,
The {{.DegovSiteConfig.Name}} Team
{{end}}

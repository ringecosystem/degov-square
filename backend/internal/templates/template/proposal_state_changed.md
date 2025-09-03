{{template "layout.md" .}}

{{define "content"}}
{{$proposalDb := .Proposal.ProposalDb}}
{{$proposalIndexer := .Proposal.ProposalIndexer}}
{{$dao := .Dao}}
{{$payload := .PayloadData}}

Hello {{if .EnsName}}{{.EnsName}}{{else}}{{.UserAddress}}{{end}},

The status of the proposal "**{{$proposalDb.Title}}**" you're following in {{$dao.Name}} has been updated.

- **Proposal:** [{{$proposalDb.Title}}]({{$proposalDb.ProposalLink}})
- **Latest Status:** **{{$payload.new_state}}**

---

### Quick Links

- [View Proposal Details]({{$proposalDb.ProposalLink}})
{{if and $proposalIndexer.TransactionHash $daoConfig.Chain.Explorers}}
- [Transaction Details]({{index $daoConfig.Chain.Explorers 0}}/tx/{{$proposalIndexer.TransactionHash}})
{{end}}

---

Thank you for staying engaged with {{$dao.Name}} onchain governance!

Best regards,
The {{.Degov.AI Team}}
{{end}}

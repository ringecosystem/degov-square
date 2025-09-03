{{template "layout.md" .}}

{{define "content"}}
{{$proposalDb := .Proposal.ProposalDb}}
{{$proposalIndexer := .Proposal.ProposalIndexer}}
{{$dao := .Dao}}
{{$vote := .Vote}}
{{$payload := .PayloadData}}

Hello {{if .EnsName}}{{.EnsName}}{{else}}{{.UserAddress}}{{end}},

This is a friendly reminder that voting for the proposal "**{{$proposalDb.Title}}**" in {{$dao.Name}} is ending soon.

**Proposal:** [{{$proposalDb.Title}}]({{$proposalDb.ProposalLink}})
**Voting Ends:** {{$proposalIndexer.VoteEndTimestamp | formatDate}} {{if $payload.TimeRemainingSeconds}}({{$payload.TimeRemainingSeconds | formatDurationShort}} remaining){{end}}
**Your Voting Power:** {{(formatBigIntWithDecimals $vote.VoteIndexer.Weight $payload.DecimalsInt) | formatLargeNumber}}

---

📊 Voting Progress ({{(formatBigIntWithDecimals $vote.TotalVotePower $payload.DecimalsInt) | formatLargeNumber}} / {{(formatBigIntWithDecimals $proposalIndexer.Quorum $payload.DecimalsInt) | formatLargeNumber}})
✅ **For:** {{(formatBigIntWithDecimals $proposalIndexer.MetricsVotesWeightForSum $payload.DecimalsInt) | formatLargeNumber}} ({{$vote.PercentFor | formatPercent}})
❌ **Against:** {{(formatBigIntWithDecimals $proposalIndexer.MetricsVotesWeightAgainstSum $payload.DecimalsInt) | formatLargeNumber}} ({{$vote.PercentAgainst | formatPercent}})
⚪️ **Abstain:** {{(formatBigIntWithDecimals $proposalIndexer.MetricsVotesWeightAbstainSum $payload.DecimalsInt) | formatLargeNumber}} ({{$vote.PercentAbstain | formatPercent}})

{{if ge $vote.PercentQuorum 100.0}}
**{{$vote.PercentQuorum | formatPercent}}** ✅ (Threshold exceeded!)
{{else}}
**{{$vote.PercentQuorum | formatPercent}}** ⚠️ (Needs more votes!)
{{end}}

---

Every vote counts in decentralized governance. Make your voice heard!

[**Cast Your Vote Now**]({{$proposalDb.ProposalLink}})

Best regards,
The {{.DegovSiteConfig.Name}} Team
{{end}}

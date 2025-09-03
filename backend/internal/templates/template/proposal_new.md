{{define "content"}}
Hello {{if .EnsName}}{{.EnsName}}{{else}}{{.UserAddress}}{{end}},

A new proposal has been created in {{.Dao.Name}} that you're subscribed to.

---

### **Proposal Details**

- **Title:** [{{.Proposal.ProposalDb.Title}}]({{.Proposal.ProposalDb.ProposalLink}})
- **Proposer:** {{.Proposal.ProposalIndexer.Proposer}}{{if .Proposal.ProposerEnsName}}({{.Proposal.ProposerEnsName}}){{end}}
- **DAO:** {{.Dao.Name}}
- **Chain:** {{.Dao.ChainName}}
- **Created:** {{.Proposal.ProposalIndexer.BlockTimestamp | formatDate}}
- **Voting Starts:** {{.Proposal.ProposalIndexer.VoteStartTimestamp | formatDate}}
- **Voting Ends:** {{.Proposal.ProposalIndexer.VoteEndTimestamp | formatDate}}

---

### **Take Action**

Please review the proposal and cast your vote.

- [**View & Vote on This Proposal**]({{.Proposal.ProposalDb.ProposalLink}})
{{if .DaoConfig.OffChainDiscussionURL}}
- [**Join Discussion**]({{.DaoConfig.OffChainDiscussionURL}})
{{end}}
{{if .Proposal.TweetLink}}
- [**View Tweet**]({{.Proposal.TweetLink}})
{{end}}

⏰ **Important:** Please cast your vote before the voting period ends on **{{.Proposal.ProposalIndexer.VoteEndTimestamp | formatDate}}**.

{{if .DegovSiteConfig.EmailProposalIncludeDescription}}
  {{if .Proposal.ProposalDescriptionMarkdown}}
---

### **Proposal Description**

{{.Proposal.ProposalDescriptionMarkdown | formatAsMdQuote}}
  {{end}}
{{end}}

---

Your participation helps shape the future of {{.Dao.Name}}!

Best regards,
The {{.DegovSiteConfig.Name}} Team


{{end}}

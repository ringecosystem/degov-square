import type { Metadata } from 'next';

import { LegalPage } from '@/components/legal-page';

export const metadata: Metadata = {
  title: 'Privacy Policy | DeGov Square',
  description: 'Privacy policy for DeGov Square.'
};

const sections = [
  {
    title: 'Information we process',
    body: [
      'DeGov Square primarily displays public DAO governance information, including DAO metadata, proposal records, voting records, contributor activity, wallet addresses, transaction hashes, and other information that is already public on blockchains or DAO registries.',
      'If you connect a wallet, sign in, subscribe to notifications, or use account-based features, we may process the wallet address, authentication status, notification preferences, email address or other contact details you choose to provide, and related service records needed to operate those features.'
    ]
  },
  {
    title: 'How we use information',
    body: [
      'We use information to provide DeGov Square, authenticate users, show DAO and proposal data, support notification features, protect the service, troubleshoot issues, improve reliability, and respond to support or governance inquiries.',
      'Proposal summaries and other AI-assisted features are provided to help users understand public governance content. They are informational and may be generated from public proposal text, voting activity, and related governance data.'
    ]
  },
  {
    title: 'ChatGPT and MCP integrations',
    body: [
      'If you use DeGov Square through ChatGPT or another MCP client, the client may send tool requests to DeGov Square and receive responses that include public DAO, proposal, contributor, voting, and registry data.',
      'The third-party client you use, including ChatGPT, may process your prompts, tool calls, and responses under its own terms and privacy policy. DeGov Square does not control those third-party clients.'
    ]
  },
  {
    title: 'Public blockchain data',
    body: [
      'Blockchain data is public by design. Wallet addresses, votes, proposal actions, delegation events, and transaction records may remain visible on public networks and third-party explorers even if they are no longer shown in DeGov Square.',
      'We do not control public blockchains, DAO registries, wallet providers, or third-party websites linked from DeGov Square.'
    ]
  },
  {
    title: 'Cookies and analytics',
    body: [
      'DeGov Square may use cookies, local storage, and similar technologies to keep the application working, remember preferences, support wallet and mini app behavior, and understand aggregate usage.',
      'We may use analytics tools, including Google Analytics, to measure traffic and improve the service. These providers may process technical information such as device, browser, page, referral, and usage data according to their own policies.'
    ]
  },
  {
    title: 'Service providers',
    body: [
      'We may use service providers for hosting, analytics, authentication, notifications, data indexing, AI processing, and infrastructure operations. These providers process information only as needed to provide their services to DeGov Square.',
      'When you use external wallets, blockchain networks, DAO websites, explorers, or communication platforms, your use is governed by those third parties and their policies.'
    ]
  },
  {
    title: 'Data retention',
    body: [
      'We keep service records only for as long as needed for the purposes described in this policy, unless a longer period is required for security, troubleshooting, legal, or governance reasons.',
      'Public blockchain records and public DAO data may remain available independently of DeGov Square and cannot be deleted by us.'
    ]
  },
  {
    title: 'Your choices',
    body: [
      'You may choose not to connect a wallet or provide optional profile or notification information. You can also manage wallet connections, browser storage, and cookie settings through your browser, wallet, or device.',
      'If you want to request access, correction, deletion, or other privacy assistance for non-public information you provided to DeGov Square, contact us through the Ring Ecosystem GitHub discussions or the public contact channels listed on helixbox.ai.'
    ]
  },
  {
    title: 'Changes',
    body: [
      'We may update this Privacy Policy as DeGov Square evolves. The updated version will be posted on this page with a new last updated date.'
    ]
  },
  {
    title: 'Contact',
    body: [
      'For privacy questions, support, or commercial hosting inquiries, use Ring Ecosystem discussions at https://github.com/orgs/ringecosystem/discussions or the public contact channels listed at https://helixbox.ai.'
    ]
  }
];

export default function PrivacyPage() {
  return (
    <LegalPage
      title="Privacy Policy"
      updatedAt="June 15, 2026"
      intro={[
        'This Privacy Policy explains how DeGov Square handles information when you use the website, MCP tools, and related services.',
        'DeGov Square is a hub for DAOs that use the DeGov governance toolkit. DeGov.AI is a RingDAO project supported by Helixbox Labs for research, development, and marketing services.'
      ]}
      sections={sections}
    />
  );
}

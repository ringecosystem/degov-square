import type { Metadata } from 'next';
import Link from 'next/link';

import { LegalPage } from '@/components/legal-page';

export const metadata: Metadata = {
  title: 'Terms of Service | DeGov Square',
  description: 'Terms of service for DeGov Square.'
};

const sections = [
  {
    title: 'Using DeGov Square',
    body: [
      'DeGov Square provides access to DAO governance information, including public DAO metadata, proposal data, proposal summaries, contributor activity, voting records, and related registry or indexer data.',
      'You are responsible for how you use the information shown by DeGov Square. You must comply with applicable laws, wallet provider terms, blockchain network rules, and DAO governance rules.'
    ]
  },
  {
    title: 'No financial, legal, or governance advice',
    body: [
      'DeGov Square is provided for informational purposes only. Nothing in DeGov Square is financial, investment, legal, tax, security, or governance advice.',
      'Proposal summaries, AI-assisted content, analytics, and governance data may be incomplete, delayed, or inaccurate. You should review original proposal text, on-chain records, DAO documentation, and other primary sources before making governance or financial decisions.'
    ]
  },
  {
    title: 'Wallets and transactions',
    body: [
      'Some DeGov experiences may let users connect wallets or navigate to DAO interfaces, wallets, explorers, or third-party services. You are solely responsible for any wallet action, signature, transaction, delegation, vote, or token transfer you initiate.',
      'The DeGov Square MCP tools are designed for read-only access to governance data and do not execute blockchain transactions.'
    ]
  },
  {
    title: 'Public data and third-party services',
    body: [
      'DeGov Square uses public blockchain data, DAO registry data, indexers, analytics, AI services, wallet infrastructure, and third-party websites. We do not control public blockchains or third-party services and are not responsible for their availability, accuracy, security, or policies.',
      'Links to external websites are provided for convenience and do not mean that RingDAO or Helixbox Labs endorses or controls those websites.'
    ]
  },
  {
    title: 'Software license',
    body: [
      'The DeGov Square source code is distributed under the license included in the repository. As of the last updated date of these Terms, the repository uses the Functional Source License, Version 1.1, Apache 2.0 Future License.',
      'These Terms govern use of the DeGov Square website, MCP tools, and related hosted services. They do not replace the software license. If there is a conflict about use, copying, modification, distribution, self-hosting, or commercial hosting of the software, the repository license controls for the software.'
    ]
  },
  {
    title: 'Commercial hosting',
    body: [
      'The DeGov Square software license restricts unauthorized commercial Software-as-a-Service and Platform-as-a-Service hosting. Commercial hosting requires approval through RingDAO governance or another approval path described in the repository license.',
      'Using the public DeGov Square service does not grant permission to offer DeGov Square or modified DeGov Square software as a commercial hosted service.'
    ]
  },
  {
    title: 'Acceptable use',
    body: [
      'You may not misuse DeGov Square, interfere with the service, attempt unauthorized access, bypass rate limits or security controls, scrape or overload the service in a way that harms availability, or use the service to violate laws or third-party rights.',
      'We may limit, suspend, or block access when needed to protect users, infrastructure, DAO communities, or the service.'
    ]
  },
  {
    title: 'Availability and changes',
    body: [
      'DeGov Square is provided on an as-available basis. Features, data sources, MCP tools, supported DAOs, supported networks, and integrations may change, pause, or be removed.',
      'We may update these Terms as DeGov Square evolves. The updated version will be posted on this page with a new last updated date.'
    ]
  },
  {
    title: 'Disclaimers',
    body: [
      'DeGov Square is provided "as is" and "as available" without warranties of any kind, whether express or implied, including warranties of merchantability, fitness for a particular purpose, non-infringement, accuracy, availability, or security.',
      'To the maximum extent permitted by law, RingDAO, Helixbox Labs, contributors, and service providers will not be liable for indirect, incidental, special, consequential, punitive, or similar damages, or for loss of profits, data, goodwill, tokens, or other assets arising from use of DeGov Square.'
    ]
  },
  {
    title: 'Contact',
    body: [
      <>
        For support, commercial hosting inquiries, or governance participation, use{' '}
        <Link
          href="https://github.com/orgs/ringecosystem/discussions"
          target="_blank"
          rel="noopener noreferrer"
          className="text-foreground underline underline-offset-4 hover:opacity-80"
        >
          Ring Ecosystem GitHub discussions
        </Link>{' '}
        or the public contact channels listed at{' '}
        <Link
          href="https://helixbox.ai"
          target="_blank"
          rel="noopener noreferrer"
          className="text-foreground underline underline-offset-4 hover:opacity-80"
        >
          helixbox.ai
        </Link>
        .
      </>
    ]
  }
];

export default function TermsPage() {
  return (
    <LegalPage
      title="Terms of Service"
      updatedAt="June 15, 2026"
      intro={[
        'These Terms of Service explain the rules for using DeGov Square, including the website, MCP tools, and related hosted services.',
        'DeGov Square is a hub for DAOs that use the DeGov governance toolkit. DeGov.AI is a RingDAO project supported by Helixbox Labs for research, development, and marketing services.'
      ]}
      sections={sections}
    />
  );
}

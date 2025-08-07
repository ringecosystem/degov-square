import type { Chip } from '@/lib/graphql/types';
import { cn } from '@/lib/utils';

import ActiveTag from './active-tag';
import AgentEnabled from './agent-enabled';

interface TagGroupProps {
  chips?: Chip[];
  className?: string;
}

const TagGroup = ({ chips, className }: TagGroupProps) => {
  console.log(chips);
  const activeTag = chips?.find((chip) => chip.chipCode === 'ACTIVE');
  const agentEnabledTag = chips?.find((chip) => chip.chipCode === 'AGENT');

  return activeTag || agentEnabledTag ? (
    <div className={cn('flex items-center gap-[10px]', className)}>
      {activeTag && <ActiveTag />}
      {agentEnabledTag && <AgentEnabled />}
    </div>
  ) : null;
};

export default TagGroup;

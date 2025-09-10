import Link from 'next/link';

import { NotFoundIcon } from '@/components/icons/not-found-icon';

export default function NotFound() {
  return (
    <div className="flex h-full flex-col items-center justify-center space-y-6 text-center">
      <NotFoundIcon className="h-16 w-auto text-muted-foreground" />
      
      <div className="space-y-2">
        <h2 className="text-xl font-semibold text-foreground">Page Not Found</h2>
        <p className="text-muted-foreground">
          The page you are looking for doesn't exist or has been moved.
        </p>
      </div>
      
      <Link 
        href="/notification" 
        className="inline-flex items-center rounded-md bg-primary px-4 py-2 text-sm font-medium text-primary-foreground hover:bg-primary/90 transition-colors"
      >
        Back to Notifications
      </Link>
    </div>
  );
}
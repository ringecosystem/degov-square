'use client';

import { useRouter } from 'next/navigation';
import { Button } from '@/components/ui/button';

export default function AddExistingSuccess() {
  const router = useRouter();

  return (
    <div className="container flex flex-col gap-[20px] py-6">
      <div className="bg-card mx-auto flex w-[800px] flex-col items-center justify-center gap-[20px] rounded-[14px] p-[20px] text-center">
        <header>
          <h2 className="text-[24px] font-bold">DAO Added Successfully!</h2>
        </header>

        <div className="py-10">
          <p className="mb-6 text-lg">
            Your DAO has been successfully added to DeGov. You can now manage your DAO from the
            dashboard.
          </p>

          <Button onClick={() => router.push('/dashboard')} className="mt-4 rounded-full px-8">
            Go to Dashboard
          </Button>
        </div>
      </div>
    </div>
  );
}

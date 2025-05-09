import { Separator } from '@/components/ui/separator';

export default function AddExisting() {
  return (
    <div className="container flex flex-col gap-[20px]">
      <div className="bg-card mx-auto flex w-[800px] flex-col gap-[20px] rounded-[14px] p-[20px]">
        <header>
          <h2 className="text-[24px] font-bold">Add existing DAO</h2>
        </header>
        <Separator className="my-0" />
        <h3>Provide the most basic information for displaying the DAO</h3>
      </div>
    </div>
  );
}

import Link from 'next/link';

type LegalSection = {
  title: string;
  body: string[];
};

type LegalPageProps = {
  title: string;
  updatedAt: string;
  intro: string[];
  sections: LegalSection[];
};

export function LegalPage({ title, updatedAt, intro, sections }: LegalPageProps) {
  return (
    <div className="container max-w-[920px]">
      <div className="flex flex-col gap-[30px]">
        <div className="flex flex-col gap-[12px]">
          <Link href="/" className="text-muted-foreground text-[14px] hover:opacity-80">
            Back to DeGov Square
          </Link>
          <h1 className="text-[32px] leading-[1.15] font-semibold md:text-[44px]">{title}</h1>
          <p className="text-muted-foreground text-[14px]">Last updated: {updatedAt}</p>
        </div>

        <div className="flex flex-col gap-[16px]">
          {intro.map((paragraph) => (
            <p key={paragraph} className="text-muted-foreground text-[16px] leading-[1.7]">
              {paragraph}
            </p>
          ))}
        </div>

        <div className="flex flex-col gap-[28px]">
          {sections.map((section) => (
            <section key={section.title} className="flex flex-col gap-[12px]">
              <h2 className="text-[22px] leading-[1.25] font-semibold">{section.title}</h2>
              {section.body.map((paragraph) => (
                <p key={paragraph} className="text-muted-foreground text-[16px] leading-[1.7]">
                  {paragraph}
                </p>
              ))}
            </section>
          ))}
        </div>
      </div>
    </div>
  );
}

import Image from "next/image";
import type { BlogAuthor, BlogPost, NewsItem } from "@/entities/article";
import { formatDate } from "@/shared/lib";
import { AppLink, cardClass, Section } from "@/shared/ui";

type NewsSectionProps = {
  /** The latest news and one more, shown first. */
  items: readonly [NewsItem, NewsItem];
  post: BlogPost;
  author: BlogAuthor;
};

function NewsCard({ item }: { item: NewsItem }) {
  return (
    <AppLink href={`/news/${item.slug}`} className={cardClass("surface", "rounded-lg p-2.5")}>
      <div className="relative h-[180px] overflow-hidden rounded-sm">
        <Image
          src={item.photo.src}
          alt={item.photo.alt}
          fill
          sizes="(min-width: 1024px) 400px, 50vw"
          className="object-cover"
        />
      </div>
      <span className="flex flex-col gap-1.5 px-3 pt-4 pb-2.5">
        <span className="text-sm text-text-muted">{formatDate(item.date)}</span>
        <span className="text-[21px] leading-tight font-bold">{item.shortTitle ?? item.title}</span>
      </span>
    </AppLink>
  );
}

export function NewsSection({ items, post, author }: NewsSectionProps) {
  const [latest, second] = items;
  const postTitle = `${post.title}: ${post.subtitle.toLocaleLowerCase("ru")}`;

  return (
    <Section
      title="Новости"
      action={
        <AppLink href="/news" className="tap-area hidden font-semibold md:inline">
          Все новости ↗
        </AppLink>
      }
    >
      {/* Mobile: two cards in a rail (the latest news and the chef's blog) */}
      <div className="-mx-3 flex gap-2 overflow-x-auto px-3 pb-1 md:hidden">
        <AppLink href={`/news/${latest.slug}`} className={cardClass("surface", "w-60 shrink-0 rounded-md p-2")}>
          <div className="relative h-[130px] overflow-hidden rounded-sm">
            <Image src={latest.photo.src} alt="" fill sizes="240px" className="object-cover" />
          </div>
          <span className="px-1.5 pt-2.5 pb-1 font-bold">{latest.title}</span>
        </AppLink>
        <AppLink href={`/news/${post.slug}`} className={cardClass("surface", "w-60 shrink-0 rounded-md p-2")}>
          <div className="relative h-[130px] overflow-hidden rounded-sm">
            <Image src={post.photo.src} alt="" fill sizes="240px" className="object-cover" />
          </div>
          <span className="px-1.5 pt-2.5 pb-1 font-bold">Блог шефа: {post.title.toLocaleLowerCase("ru")}</span>
        </AppLink>
      </div>

      {/* From 768 px: two news cards and the chef card */}
      <div className="hidden gap-3 md:grid md:grid-cols-2 lg:grid-cols-3">
        <NewsCard item={latest} />
        <NewsCard item={second} />
        <AppLink
          href={`/news/${post.slug}`}
          className={cardClass("surface", "gap-3 rounded-lg p-6 md:col-span-2 lg:col-span-1")}
        >
          <Image
            src={author.photo.src}
            alt={author.photo.alt}
            width={72}
            height={72}
            className="size-[72px] rounded-full object-cover object-top"
          />
          <span className="text-sm text-text-muted">Блог шефа · {author.name}</span>
          <span className="flex-1 text-[22px] leading-tight font-bold">{postTitle}</span>
          <span className="font-semibold">Читать ↗</span>
        </AppLink>
      </div>
    </Section>
  );
}

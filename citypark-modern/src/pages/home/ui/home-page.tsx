import { getBlogAuthor, getBlogPosts, getNews } from "@/entities/article";
import { getMenu } from "@/entities/dish";
import { getFreeRoomsCount, getRooms } from "@/entities/room";
import { PLACEHOLDER_PHOTO } from "@/shared/lib";
import { ContactsBlock } from "./contacts-block";
import { DeliverySection } from "./delivery-section";
import { Hero } from "./hero";
import { NewsSection } from "./news-section";
import { RestaurantSection } from "./restaurant-section";
import { RoomsSection } from "./rooms-section";

const HOME_CATEGORIES = 4;
const HOME_DISHES = 3;

export async function HomePage() {
  const [rooms, freeRooms, menu, news, posts, author] = await Promise.all([
    getRooms(),
    getFreeRoomsCount().catch((error: unknown) => {
      console.error(error);
      return null;
    }),
    getMenu().catch((error: unknown) => {
      console.error(error);
      return null;
    }),
    getNews(),
    getBlogPosts(),
    getBlogAuthor(),
  ]);
  const available = menu?.dishes.filter((dish) => dish.available) ?? [];
  const withPhoto = available.filter((dish) => dish.photo.src !== PLACEHOLDER_PHOTO);
  const dishes = [...withPhoto, ...available.filter((dish) => !withPhoto.includes(dish))].slice(0, HOME_DISHES);
  const latest = news[0];
  const second = news.find((item) => item.slug === "new-menu-2026") ?? news[1];

  return (
    <main>
      <Hero />
      <RoomsSection rooms={rooms} free={freeRooms} />
      <RestaurantSection />
      {menu && <DeliverySection categories={menu.categories.slice(0, HOME_CATEGORIES)} dishes={dishes} />}
      <NewsSection items={[latest, second]} post={posts[0]} author={author} />
      <div className="mx-auto w-full max-w-page px-3 pt-14 pb-0 md:px-6 md:pt-28 md:pb-6">
        <ContactsBlock />
      </div>
    </main>
  );
}

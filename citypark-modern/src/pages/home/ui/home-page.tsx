import { getBlogAuthor, getBlogPosts, getNews } from "@/entities/article";
import { getMenu } from "@/entities/dish";
import { getRooms } from "@/entities/room";
import { ContactsBlock } from "./contacts-block";
import { DeliverySection } from "./delivery-section";
import { Hero } from "./hero";
import { NewsSection } from "./news-section";
import { RestaurantSection } from "./restaurant-section";
import { RoomsSection } from "./rooms-section";

// Dishes shown on the home page, in the order of the design.
const HOME_DISH_IDS = ["beet-salad", "vegetable-pan", "tofu-poke"];

export async function HomePage() {
  const [rooms, menu, news, posts, author] = await Promise.all([
    getRooms(),
    getMenu(),
    getNews(),
    getBlogPosts(),
    getBlogAuthor(),
  ]);
  const dishes = HOME_DISH_IDS.map((id) => menu.dishes.find((dish) => dish.id === id)).filter(
    (dish) => dish !== undefined,
  );
  const latest = news[0];
  const second = news.find((item) => item.slug === "new-menu-2026") ?? news[1];

  return (
    <main>
      <Hero />
      <RoomsSection rooms={rooms} />
      <RestaurantSection />
      <DeliverySection
        categories={menu.categories.filter((c) => ["lenten", "steaks", "rolls", "pizza"].includes(c.id))}
        dishes={dishes}
      />
      <NewsSection items={[latest, second]} post={posts[0]} author={author} />
      <div className="mx-auto w-full max-w-page px-3 pt-14 pb-0 md:px-6 md:pt-28 md:pb-6">
        <ContactsBlock />
      </div>
    </main>
  );
}

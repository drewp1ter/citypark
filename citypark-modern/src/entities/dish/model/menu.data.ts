import { typo } from "@/shared/lib";
import type { Dish, MenuCategory } from "./types";

// Copied from design/reference/*/menu.html and home.html. Only the "Постное" category has dishes in the design.
export const categories: readonly MenuCategory[] = [
  { id: "lenten", name: "Постное", title: "Постное меню" },
  { id: "specials", name: "Спецпредложения" },
  { id: "burgers", name: "Бургеры" },
  { id: "steaks", name: "Стейки" },
  { id: "breakfasts", name: "Завтраки до 12:00" },
  { id: "kurze", name: "Курзе" },
  { id: "lunch-sets", name: "Обеденные сеты" },
  { id: "rolls", name: "Роллы" },
  { id: "poke", name: "Поке" },
  { id: "pizza", name: "Пицца" },
  { id: "mangal", name: "Мангал" },
];

/** The menu page shows "Ещё 16 ▾" after the categories above; their names are not known yet. */
export const hiddenCategoriesCount = 16;

const photo = (file: string, alt: string) => ({ src: `/images/${file}.jpg`, alt });

export const dishes: readonly Dish[] = [
  {
    id: "beet-salad",
    name: "Салат свекольный",
    categoryId: "lenten",
    composition: "Свекла, чеснок, грецкий орех",
    photo: photo("dish-beet-salad", "Салат свекольный"),
    variants: [{ id: "default", label: typo("200 г"), weight: typo("200 г"), price: 250 }],
    available: true,
  },
  {
    id: "warm-bean-salad",
    name: "Тёплый салат с фасолью",
    categoryId: "lenten",
    composition: "Фасоль, шампиньоны, чеснок",
    photo: photo("dish-warm-bean-salad", "Тёплый салат с фасолью"),
    variants: [{ id: "default", label: typo("400 г"), weight: typo("400 г"), price: 550 }],
    available: true,
  },
  {
    id: "vegetable-salad",
    name: "Салат овощной",
    categoryId: "lenten",
    composition: "Огурцы, томаты, перец",
    photo: photo("dish-vegetable-salad", "Салат овощной"),
    variants: [{ id: "default", label: typo("360 г"), weight: typo("360 г"), price: 450 }],
    available: true,
  },
  {
    id: "pumpkin-soup",
    name: "Крем-суп из тыквы",
    categoryId: "lenten",
    composition: "Тыква, кокосовое молоко, тофу",
    photo: photo("dish-pumpkin-soup", "Крем-суп из тыквы"),
    variants: [{ id: "default", label: typo("300 г"), weight: typo("300 г"), price: 300 }],
    available: true,
  },
  {
    id: "bean-borscht",
    name: "Борщ с фасолью",
    categoryId: "lenten",
    composition: "Фасоль, капуста, свекла",
    photo: photo("dish-borscht", "Борщ с фасолью"),
    variants: [{ id: "default", label: typo("250/50/20 г"), weight: typo("250/50/20 г"), price: 250 }],
    available: true,
  },
  {
    id: "mushroom-risotto",
    name: "Ризотто с грибами",
    categoryId: "lenten",
    composition: "Рис, шампиньоны, черри",
    photo: photo("dish-mushroom-risotto", "Ризотто с грибами"),
    variants: [{ id: "default", label: typo("300 г"), weight: typo("300 г"), price: 400 }],
    available: true,
  },
  {
    id: "kurze-potato",
    name: "Курзе с картофелем",
    categoryId: "lenten",
    composition: "Постное тесто, картофель, лук",
    photo: photo("dish-kurze", "Курзе"),
    variants: [{ id: "default", label: typo("250 г"), weight: typo("250 г"), price: 350 }],
    available: true,
  },
  {
    id: "thai-rice",
    name: "Рис в тайском стиле",
    categoryId: "lenten",
    composition: "Рис, соевый соус, овощи",
    photo: photo("dish-thai-rice", "Рис в тайском стиле"),
    variants: [{ id: "default", label: typo("200 г"), weight: typo("200 г"), price: 250 }],
    available: true,
  },
  {
    id: "cabbage-pie",
    name: "Пирог с капустой",
    categoryId: "lenten",
    photo: photo("dish-cabbage-pie", "Пирог с капустой"),
    variants: [
      { id: "500g", label: typo("500 г"), weight: typo("500 г"), price: 350 },
      // The design shows the switch but not the 1 кг price. It will come dynamically (POS / backend);
      // until then the variant is a placeholder: visible, but it cannot be added to the cart.
      { id: "1kg", label: typo("1 кг"), weight: typo("1 кг"), price: null },
    ],
    available: true,
  },
  {
    id: "tofu-poke",
    name: "Поке с тофу",
    categoryId: "lenten",
    photo: photo("dish-tofu-poke", "Поке с тофу"),
    variants: [{ id: "default", label: typo("350 г"), weight: typo("350 г"), price: 350 }],
    available: true,
  },
  {
    id: "vegetable-pan",
    name: "Овощная сковородка",
    categoryId: "lenten",
    photo: photo("dish-vegetable-pan", "Овощная сковородка"),
    variants: [{ id: "default", label: typo("500 г"), weight: typo("500 г"), price: 550 }],
    available: true,
  },
];

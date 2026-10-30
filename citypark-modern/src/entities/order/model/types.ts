export type Order = {
 id: number; createdAt: string; status: string; comment: string; deliveryTime: string;
 total: number; deliveryPrice: number;
 customer: { id: number; name: string; phone: string };
 address: { id: number; city: string; address: string } | null;
 items: { productId: number; name: string; categoryName: string; amount: number; offeredPrice: number }[];
}

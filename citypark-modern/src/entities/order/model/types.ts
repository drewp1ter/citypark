export type Order = {
 paymentRetry?: { allowed: boolean; reason: '' | 'not_payable' | 'expired' | 'closed' };
 id: number; createdAt: string; status: string; comment: string; deliveryTime: string;
 subtotal: number; discount: number; promoCode: string | null;
 total: number; deliveryPrice: number;
 customer: { id: number; name: string; phone: string };
 address: { id: number; city: string; address: string } | null;
 items: { productId: number; name: string; categoryName: string; amount: number; offeredPrice: number; discount: number }[];
}

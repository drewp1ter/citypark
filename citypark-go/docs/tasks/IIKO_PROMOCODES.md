# Promo discounts in iiko: RMS contract planning

Status: **planned RMS mapping**, revised 2026-10-31. Task 84 remains unchecked until its revised
source/fixture checks are completed. Owner accepts different iiko per-line allocations when eligible positions,
total discount and food payment match exactly. The previous exact-per-line blocker is superseded.
No application code, configuration, real orders or payments were changed.

## Official evidence

Retrieved 2026-10-31:

- [iikoCloud API documentation](https://api-ru.iiko.services/docs).
- [Official OpenAPI JSON](https://api-ru.iiko.services/api-docs/docs), referenced directly by that page's
  `Redoc.init`. Retrieved successfully using HTTPS with curl; the browsing tool could not open this host.
  SHA-256 of retrieved bytes: `022ffac44d581f981a76408c426bd30c3393ff0c5333c7b3d40629dc26bbffb7`.
  Relevant schemas are under `components.schemas`, prefix
  `iikoTransport.PublicApi.Contracts.Deliveries.Request.CreateOrder.`.
- [Official iikoFront discount explanation](https://iiko.github.io/front.api.doc/v6/ru/Discounts.html).
  This describes the Front SDK, **not** an alternative HTTP integration. It explains why selective application
  alone does not prove a fixed result: other discount restrictions still apply, automatic discounts can exist,
  and calculations may change until fixed. Do not infer HTTP capabilities from SDK methods.

The repository defaults to `https://api-ru.iiko.services/api` and sends `POST /api/1/deliveries/create`.
This identifies the integration family; it does not establish the deployed restaurant's Front version or settings.

| Schema / endpoint | Established by official OpenAPI | Not established |
|---|---|---|
| `ProductOrderItem` | `productId`, unit `price`, optional `positionId`; supplied price can differ from menu price; position UUID must be globally unique | Monetary precision/rounding for a non-terminating discounted unit price; whether separate price positions are preserved or merged on the deployed Front |
| `OrderItem` | `type`, `amount`; quantity range 0–999.999 | A line-total override field |
| `DiscountsInfo` | `discounts`, `card`, `fixedLoyaltyDiscounts` | That fixing loyalty discounts disables every other automatic RMS discount |
| `RmsDiscount` | `type: RMS`, `discountTypeId`, `sum`, `selectivePositions` (position UUIDs); supply sum only for an assign-amount discount | Exact allocation/rounding across positions; repeated instances of the same discount type with different sums/positions |
| `IikoCardDiscount` | `type: iikoCard`, `programId`, `programName`, `discountItems`; each item has `positionId`, `sum`, `amount`; available from 7.4.4 | Permission to use this as an arbitrary website discount without a real configured card program; deployed program/version support |
| `Payment` | `paymentTypeKind`, `paymentTypeId`, `sum`, `isProcessedExternally`; external processing means payment was made outside iiko | Acceptance of underpaying unchanged food lines as a discount |
| `/api/1/discounts` | Organization-specific IDs, mode (`Percent`, `FlexibleSum`, `FixedSum`), selective-application capability and other restrictions | A usable discount ID or its deployed configuration without an organization response |
| `/api/1/payment_types` | Payment kind, processing type (`External`, `Internal`, `Both`), available terminal groups | That the configured payment type supports these discounted orders |

## Exact arithmetic fixtures

All amounts below are **integer kopecks**, quantities are unchanged. Product IDs are local fixture IDs, not
invented iiko identifiers. These are exact saved-pricing examples; task 84 must still add complete RMS create-request fixtures.
Campaign targets product 1 plus its category and product 2's category: product 1 qualifies once through the union;
product 3 is ineligible. Percentage is 20, cap is 100 kopecks.

```json
{
  "percentage": 20,
  "maximumDiscountKopecks": 100,
  "items": [
    {"productId": 1, "amount": 3, "offeredPriceKopecks": 101, "discountKopecks": 43},
    {"productId": 2, "amount": 2, "offeredPriceKopecks": 202, "discountKopecks": 57},
    {"productId": 3, "amount": 1, "offeredPriceKopecks": 500, "discountKopecks": 0}
  ],
  "subtotalKopecks": 1207,
  "eligibleSubtotalKopecks": 707,
  "discountKopecks": 100,
  "totalKopecks": 1107
}
```

Calculation: uncapped discount = floor(707 × 20 / 100) = 141, capped to 100.
Allocation floors are 42 and 57; remainders are 606/707 and 101/707. The last kopeck goes to product 1,
so discounts are 43 and 57. Net lines are 260, 347, 500; their sum is 1107.

| Fulfillment | Original food | Discount | Required iiko item total | Required food payment | Delivery | MONETA |
|---|---:|---:|---:|---:|---:|---:|
| Pickup | 1207 | 100 | 1107 | 1107 | 0 | 1107 |
| Courier, town price 10000 and free threshold 100000 | 1207 | 100 | 1107 | 1107 | 10000 | 11107 |

The courier fixture retains the existing placeholder `deliveryPoint.address` and actual address in comments.
Delivery stays out of items and the iiko payment, and is recorded as paid in the existing delivery comment.
Its fee is evaluated against food **after** discount. Pickup retains `DeliveryByClient`; courier retains
`DeliveryByCourier`. Existing IDs, customer fields and retry behavior must be preserved.

## Selected RMS mapping

Use one real restaurant RMS discount type configured for assign-amount (FlexibleSum) and selective application.
Configure its UUID through IIKO_PROMO_DISCOUNT_TYPE_ID; do not create an iiko campaign for each website code.
Keep the original item prices and quantities. Assign globally unique position UUIDs once for an outgoing order,
reusing them throughout retries. Send one order.discountsInfo.discounts entry with type=RMS, discountTypeId,
sum equal to the saved total discount in rubles, and selectivePositions containing all saved eligible positions.
Include eligible positions even if their backend allocation rounded to zero; never infer eligibility from a positive
discount or current campaign targets. Task 88 must persist this membership. Omit the RMS entry for zero discount.

For the arithmetic fixture above, send original unit prices 1.01, 2.02 and 5.00 rubles, quantities 3, 2 and 1,
and RMS sum 1.00 applied only to positions for products 1 and 2. Required food payment is 11.07 rubles.
iiko may allocate that 1.00 differently from our saved 0.43/0.57; ineligible product 3 must not receive it.
Our API/history retain backend allocations. No split-price items, fractional-kopeck unit prices, repeated RMS
entries or iikoCard programs are needed. Delivery and payment treatment remain as described above.

## Remaining task 84 work and release checks

Task 84 must add complete pickup/courier HTTP JSON fixtures with clearly labelled fixture UUIDs, verify their
schema fields and reconcile totals. This plan update does not mark those acceptance checks complete.
Restaurant configuration is a deployment requirement: confirm the real ID via /api/1/discounts, FlexibleSum and
selective settings, payment compatibility via /api/1/payment_types, and both fulfillment modes. Confirm no
category/time/minimum restrictions, automatic discounts or tax settings alter the required result.

Task 89 adds and validates IIKO_PROMO_DISCOUNT_TYPE_ID. Missing configuration rejects new promo orders before
writes/payment creation with 503 «Заказы с промокодом временно недоступны.». Existing pending retries keep
saved monetary snapshots; already-paid dispatch failures retain alerts and retry behavior. Keep campaigns
disabled until an explicitly authorized acceptance check verifies the actual RMS discount and food payment.
No real orders are authorized by this documentation task. External payment processing does not imply external
fiscalization; retain existing fiscalization behavior.

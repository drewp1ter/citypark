# Promo RMS integration contract

Verified 2026-10-31. Mandatory input to task 89. Business rules remain in [PROMOCODES.md](PROMOCODES.md).

## Official evidence

- [Documentation UI](https://api-ru.iiko.services/docs).
- [Official OpenAPI document loaded by that UI](https://api-ru.iiko.services/api-docs/docs).

Retrieved both using HTTP GET. HEAD on /docs returned 404 even though GET returned 200.
Schema SHA-256: `022ffac44d581f981a76408c426bd30c3393ff0c5333c7b3d40629dc26bbffb7`.

The schema establishes the wire representation; it does not prove a restaurant's configuration or runtime acceptance.
The relevant component names below share prefix `iikoTransport.PublicApi.Contracts.`:

| Schema / endpoint | Supported contract |
|---|---|
| POST /api/1/deliveries/create | organizationId, terminalGroupId, order and existing createOrderSettings envelope |
| Deliveries.Request.CreateOrder.ProductOrderItem | Original price per unit, inherited amount, productId, positionId UUID unique across the whole system |
| Deliveries.Request.CreateOrder.Discount | type discriminator maps RMS to RmsDiscount |
| Deliveries.Request.CreateOrder.RmsDiscount | discountTypeId UUID, sum, selectivePositions UUID array; sum is supplied only for assign-amount discounts |
| Deliveries.Request.CreateOrder.Payment | paymentTypeKind, sum, paymentTypeId; isProcessedExternally means payment processed outside iiko |
| POST /api/1/discounts | Request organizationIds array; response discounts grouped by organization |
| Discounts.DiscountCardTypeInfo | mode, canBeAppliedSelectively, isCategorisedDiscount, productCategoryDiscounts, minOrderSum, isManual, isAutomatic, isDeleted |
| Discounts.DiscountCardMode | Percent, FlexibleSum, FixedSum |

Use exactly one RMS entry in order.discountsInfo.discounts for positive saved discount. Its sum is the saved total in rubles. selectivePositions contains every saved eligible line, even when backend allocation is zero. For zero discount omit the RMS entry. Do not send response-only selectivePositionsWithSum as a create field.
Generate random globally unique position UUIDs once and persist them with order snapshots; preserve them across retries together with the order UUID. Never reuse fixture UUIDs in real orders. Never reconstruct eligibility from current campaigns. Keep original quantities and offered unit prices; do not split or reprice positions.

## Restaurant setup and deployment acceptance

IIKO_PROMO_DISCOUNT_TYPE_ID must reference a real discount returned for the configured organization by /api/1/discounts. Configure assign amount (FlexibleSum), canBeAppliedSelectively=true, nondeleted, manual and nonautomatic, without category restrictions or a minimum that changes the backend result. The assign-amount requirement is explicit in RmsDiscount; interpreting FlexibleSum as that setting must also be confirmed in restaurant setup. This is one shared RMS discount, not one per campaign.
Confirm activation, time windows, VAT/tax treatment, discount combinations and both fulfillment modes in restaurant configuration. The dictionary does not expose every configuration condition. Confirm the existing configured payment ID/kind via /api/1/payment_types, processing type External or Both and availability for the terminal group, and its compatibility with this discount and external online payments. Do not infer compatibility from the schema alone.
Task 89 must reject new promo orders before writes/payment creation when integration configuration is missing, using the shared contract's 503. Retain paid-order alert/retry handling. Deploy backend tasks 85–89 before enabling checkout campaigns.
Keep campaigns disabled until an explicitly authorized restaurant acceptance check verifies pickup and courier creation, exact total discount, unchanged ineligible items and exact food payment. Inspect command completion and the resulting iikoFront order. Backend and iiko allocations across eligible lines may differ; this alone is acceptable. No real order or payment was sent for this task.

Unsupported: Percent/FixedSum configuration for this sum override, nonselective discounts, conflicting automatic/category discounts, missing saved eligibility/position IDs, iikoCard substitution, split-price lines, stacking and refunds. A restaurant that cannot honor the exact total and eligible set blocks release; identical per-line rounding is not required.

## Reconciled synthetic fixtures

All UUIDs and contact/address values below are illustrative. Replace organization, terminal, catalog, payment and discount IDs with real configuration for authorized acceptance. Fixtures extend the current integration's legacy-address mapping and externally processed payment behavior.

Campaign: 50%, cap 1.00 RUB (binding), selected product 1 plus its category (overlap counted once), product 2 also eligible. Product 3 is ineligible. Product 4 is eligible with zero backend allocation.

| Product | Unit RUB | Quantity | Gross RUB | Eligible | Backend discount RUB |
|---|---:|---:|---:|---|---:|
| 1 | 100.00 | 3 | 300.00 | yes, overlapping targets | 0.60 |
| 2 | 100.00 | 2 | 200.00 | yes | 0.40 |
| 3 | 50.00 | 1 | 50.00 | no | 0.00 |
| 4 | 0.01 | 1 | 0.01 | yes | 0.00 |

Eligible subtotal 500.01; floor(50001 * 50 / 100) = 25000 kopecks; cap yields 100. Proportional floors are 59, 39, 0; descending remainders assign one extra kopeck each to products 1 and 2. Gross food 550.01 minus discount 1.00 = iiko food total/payment 549.01. Product 2's net 199.60 / 2 divides evenly, but product 1's 299.40 / 3 also divides evenly; the additional fixture below exercises indivisible net unit prices.

Pickup has delivery 0 and MONETA 549.01. Courier assumes the existing town rule yields delivery 100.00 after discount: iiko food/payment remains 549.01, MONETA is 649.01. Delivery remains a paid-delivery comment, with no delivery position or discount.

### Pickup

```json
{
  "organizationId": "00000000-0000-4000-8000-000000000020",
  "terminalGroupId": "00000000-0000-4000-8000-000000000021",
  "createOrderSettings": {
    "transportToFrontTimeout": 180
  },
  "order": {
    "id": "00000000-0000-4000-8000-000000000010",
    "externalNumber": "100",
    "phone": "+79990000000",
    "orderServiceType": "DeliveryByClient",
    "customer": {
      "type": "one-time",
      "name": "Тест"
    },
    "comment": "Заказ с сайта, оплачен онлайн",
    "items": [
      {
        "type": "Product",
        "productId": "00000000-0000-4000-8000-000000000001",
        "positionId": "00000000-0000-4000-8000-000000000101",
        "amount": 3,
        "price": 100
      },
      {
        "type": "Product",
        "productId": "00000000-0000-4000-8000-000000000002",
        "positionId": "00000000-0000-4000-8000-000000000102",
        "amount": 2,
        "price": 100
      },
      {
        "type": "Product",
        "productId": "00000000-0000-4000-8000-000000000003",
        "positionId": "00000000-0000-4000-8000-000000000103",
        "amount": 1,
        "price": 50
      },
      {
        "type": "Product",
        "productId": "00000000-0000-4000-8000-000000000004",
        "positionId": "00000000-0000-4000-8000-000000000104",
        "amount": 1,
        "price": 0.01
      }
    ],
    "discountsInfo": {
      "discounts": [
        {
          "type": "RMS",
          "discountTypeId": "00000000-0000-4000-8000-000000000030",
          "sum": 1,
          "selectivePositions": [
            "00000000-0000-4000-8000-000000000101",
            "00000000-0000-4000-8000-000000000102",
            "00000000-0000-4000-8000-000000000104"
          ]
        }
      ]
    },
    "payments": [
      {
        "paymentTypeKind": "Card",
        "paymentTypeId": "00000000-0000-4000-8000-000000000040",
        "sum": 549.01,
        "isProcessedExternally": true
      }
    ]
  }
}
```

### Courier

```json
{
  "organizationId": "00000000-0000-4000-8000-000000000020",
  "terminalGroupId": "00000000-0000-4000-8000-000000000021",
  "createOrderSettings": {
    "transportToFrontTimeout": 180
  },
  "order": {
    "id": "00000000-0000-4000-8000-000000000011",
    "externalNumber": "101",
    "phone": "+79990000000",
    "orderServiceType": "DeliveryByCourier",
    "customer": {
      "type": "one-time",
      "name": "Тест"
    },
    "comment": "Заказ с сайта, оплачен онлайн\nАдрес: Белореченск, тестовый адрес, 1\nДоставка: 100.00 ₽ (оплачена)",
    "items": [
      {
        "type": "Product",
        "productId": "00000000-0000-4000-8000-000000000001",
        "positionId": "00000000-0000-4000-8000-000000000101",
        "amount": 3,
        "price": 100
      },
      {
        "type": "Product",
        "productId": "00000000-0000-4000-8000-000000000002",
        "positionId": "00000000-0000-4000-8000-000000000102",
        "amount": 2,
        "price": 100
      },
      {
        "type": "Product",
        "productId": "00000000-0000-4000-8000-000000000003",
        "positionId": "00000000-0000-4000-8000-000000000103",
        "amount": 1,
        "price": 50
      },
      {
        "type": "Product",
        "productId": "00000000-0000-4000-8000-000000000004",
        "positionId": "00000000-0000-4000-8000-000000000104",
        "amount": 1,
        "price": 0.01
      }
    ],
    "discountsInfo": {
      "discounts": [
        {
          "type": "RMS",
          "discountTypeId": "00000000-0000-4000-8000-000000000030",
          "sum": 1,
          "selectivePositions": [
            "00000000-0000-4000-8000-000000000101",
            "00000000-0000-4000-8000-000000000102",
            "00000000-0000-4000-8000-000000000104"
          ]
        }
      ]
    },
    "payments": [
      {
        "paymentTypeKind": "Card",
        "paymentTypeId": "00000000-0000-4000-8000-000000000040",
        "sum": 549.01,
        "isProcessedExternally": true
      }
    ],
    "deliveryPoint": {
      "address": {
        "type": "legacy",
        "street": {
          "name": "Ленина",
          "city": "Белореченск"
        },
        "house": "1"
      },
      "comment": "Белореченск, тестовый адрес, 1"
    }
  }
}
```

### Indivisible-unit boundary

Use either fixture with cap 1.01 instead of 1.00: allocation becomes 0.61, 0.40, 0, 0 (same remainder method). Product 1 net is 299.39 RUB for quantity 3, which cannot be represented as an integer-kopeck unit price. Keep price=100 and amount=3; change only RMS sum to 1.01 and payment sum to 549.00. Gross 550.01 - 1.01 = 549.00; MONETA pickup=549.00, courier=649.00. Eligibility still includes product 4. This representation avoids division of net line values into unit prices.

### Zero-discount boundary

For saved discount 0 omit the RMS entry, retain original prices and quantities, and send food payment 550.01; MONETA is 550.01 for pickup or 650.01 with delivery 100.00. Re-evaluate the existing town threshold against discounted food when initially pricing each order; these courier examples assume delivery remains 100.00. Retries use saved totals.

Verification: source links retrieved, create request inheritance/discriminators and dictionary fields inspected, all fixture sums reconciled in integer kopecks. No application changes; application tests are unnecessary for this documentation-only task. Restaurant setup and authorized runtime acceptance remain release checks.

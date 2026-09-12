const SITE_URL = process.env.NEXT_PUBLIC_SITE_URL ?? "https://pay.rexio.pro";

/**
 * llms.txt convention: plain Markdown at the site root describing the product
 * for crawler summarization. Written per HUMANIZER: plain claims, no hype.
 */
export function GET(): Response {
  const body = `# RexiO Pay

RexiO Pay is a hosted payment gateway for Bangladesh that lets online businesses accept bKash and Nagad payments automatically. It works without official MFS APIs: an Android app on the merchant's own phone reads the payment SMS, forwards it to RexiO Pay, and the service matches it to a pending checkout session using the transaction ID, sender number, and a time window. When a payment verifies, the customer's checkout page confirms it and the merchant's site receives a signed webhook.

RexiO Pay does not hold, move, or pool merchant funds. Money goes directly from the customer's MFS account to the merchant's own account. Merchants are responsible for compliance with their MFS provider's terms and Bangladesh Bank regulations.

## Pages

- [Home](${SITE_URL}/): product overview, plans, and frequently asked questions.
- [API docs](${SITE_URL}/docs): REST API reference generated from the OpenAPI specification.

## Plans

Starter is free. Pro is 1,000 BDT per month. Business is 3,000 BDT per month. There is no per-transaction fee.

## Private and dynamic routes

- /app (merchant dashboard), /rexio-admin (platform admin), and /pay/* (customer checkout sessions) require authentication or a per-session ID. They are not meant for crawling or summarization.
`;
  return new Response(body, {
    headers: { "content-type": "text/plain; charset=utf-8" },
  });
}

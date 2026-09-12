/**
 * Entity graph per REQUIREMENT §20: Organization (SpritexAI), Person (founder),
 * SoftwareApplication (RexiO Pay, brand RexiO), WebSite/WebPage, FAQPage.
 */

export const SITE_URL =
  process.env.NEXT_PUBLIC_SITE_URL ?? "https://pay.rexio.pro";

export const ORGS = {
  spritexai: {
    "@type": "Organization",
    "@id": "https://spritexai.pro.bd/#organization",
    name: "SpritexAI",
    url: "https://spritexai.pro.bd",
  },
  founder: {
    "@type": "Person",
    "@id": "https://sijan.pro.bd/#person",
    name: "Mohammad Sijan",
    url: "https://sijan.pro.bd",
  },
  rexioBrand: {
    "@type": "Brand",
    "@id": "https://rexio.pro/#brand",
    name: "RexiO",
    url: "https://rexio.pro",
  },
};

export const PRODUCT_ID = `${SITE_URL}/#product`;

export const PLANS = [
  { code: "starter", price: 0 },
  { code: "pro", price: 1000 },
  { code: "business", price: 3000 },
] as const;

export function offersGraph() {
  return PLANS.map((plan) => ({
    "@type": "Offer",
    name: `RexiO Pay ${plan.code[0].toUpperCase()}${plan.code.slice(1)}`,
    price: plan.price,
    priceCurrency: "BDT",
    availability: "https://schema.org/InStock",
    url: `${SITE_URL}/#pricing`,
  }));
}

export function landingJsonLd(faq: { q: string; a: string }[]) {
  return {
    "@context": "https://schema.org",
    "@graph": [
      {
        ...ORGS.spritexai,
        founder: { "@id": ORGS.founder["@id"] },
      },
      {
        ...ORGS.founder,
      },
      {
        "@type": "SoftwareApplication",
        "@id": PRODUCT_ID,
        name: "RexiO Pay",
        alternateName: "RexiO Pay by SpritexAI",
        applicationCategory: "FinanceApplication",
        operatingSystem: "Web, Android",
        url: SITE_URL,
        description:
          "SMS-verified bKash and Nagad payment gateway for Bangladesh. RexiO Pay verifies payments from the merchant's own phone and notifies their site automatically.",
        brand: { "@id": ORGS.rexioBrand["@id"] },
        isPartOf: { "@id": ORGS.spritexai["@id"] },
        parentOrganization: { "@id": ORGS.spritexai["@id"] },
        offers: offersGraph(),
      },
      {
        "@type": "WebSite",
        "@id": `${SITE_URL}/#website`,
        url: SITE_URL,
        name: "RexiO Pay",
        publisher: { "@id": ORGS.spritexai["@id"] },
        inLanguage: ["en", "bn"],
      },
      {
        "@type": "WebPage",
        "@id": `${SITE_URL}/#webpage`,
        url: SITE_URL,
        name: "RexiO Pay: bKash and Nagad payments, verified automatically",
        isPartOf: { "@id": `${SITE_URL}/#website` },
        about: { "@id": PRODUCT_ID },
        inLanguage: "en",
      },
      {
        "@type": "FAQPage",
        "@id": `${SITE_URL}/#faq`,
        isPartOf: { "@id": `${SITE_URL}/#webpage` },
        mainEntity: faq.map((item) => ({
          "@type": "Question",
          name: item.q,
          acceptedAnswer: { "@type": "Answer", text: item.a },
        })),
      },
    ],
  };
}

export function breadcrumbJsonLd(items: { name: string; path: string }[]) {
  return {
    "@context": "https://schema.org",
    "@type": "BreadcrumbList",
    itemListElement: items.map((item, i) => ({
      "@type": "ListItem",
      position: i + 1,
      name: item.name,
      item: `${SITE_URL}${item.path}`,
    })),
  };
}

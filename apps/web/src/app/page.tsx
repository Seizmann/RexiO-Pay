export default function Home() {
  return (
    <main className="min-h-screen bg-canvas">
      <div className="mx-auto max-w-6xl px-6 py-24">
        <p className="text-xs font-normal uppercase tracking-[0.1px] text-primary-deep">
          RexiO Pay — a SpritexAI product
        </p>
        <h1 className="mt-4 text-5xl font-light tracking-[-1.4px] text-ink">
          Take bKash and Nagad payments, verified automatically.
        </h1>
        <p className="mt-6 max-w-xl text-base font-light text-ink-mute">
          Your customer pays your own MFS number. Your phone forwards the
          payment SMS. RexiO Pay matches it and tells your site within
          seconds — no manual checking, no official API needed.
        </p>
        <a
          href="#"
          className="mt-8 inline-block rounded-full bg-primary px-4 py-2 text-base font-normal text-white"
        >
          Get started
        </a>
      </div>
    </main>
  );
}

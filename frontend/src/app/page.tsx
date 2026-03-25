import Link from 'next/link';

export default function Home() {
  return (
    <main className="flex min-h-screen flex-col items-center justify-center p-24 bg-black">
      <div className="space-y-8 text-center">
        <h1 className="text-8xl font-black text-white italic tracking-tighter shadow-brand-gold/20 drop-shadow-2xl">
          NEXUS
        </h1>
        <p className="text-xl text-slate-500 font-bold uppercase tracking-[0.3em]">
          The Power to Win. The Power to Create.
        </p>
        <div className="flex gap-4 justify-center">
          <Link href="/dashboard" className="gold-gradient text-black px-8 py-4 rounded-2xl font-black uppercase tracking-widest shadow-xl hover:scale-105 transition-all active:scale-95">
            Open Dashboard
          </Link>
          <Link href="/studio" className="bg-white/5 text-white border border-white/10 px-8 py-4 rounded-2xl font-black uppercase tracking-widest hover:bg-white/10 transition-all">
            Studio
          </Link>
        </div>
      </div>
    </main>
  );
}

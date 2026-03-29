import React from "react";
import Link from "next/link";
import { Zap } from "lucide-react";
import type { Metadata } from "next";

export const metadata: Metadata = {
  title: "Privacy Policy | Loyalty Nexus",
  description: "How Loyalty Nexus collects, uses, and protects your personal data.",
};

export default function PrivacyPage() {
  return (
    <main className="min-h-screen" style={{ background: "#0a0b0e", color: "#f0f2ff" }}>
      {/* Nav */}
      <div className="border-b border-white/[0.07]">
        <div className="max-w-6xl mx-auto px-4 sm:px-6 h-16 flex items-center justify-between">
          <Link href="/" className="flex items-center gap-2">
            <div className="w-7 h-7 rounded-lg bg-yellow-400 flex items-center justify-center">
              <Zap className="w-3.5 h-3.5 text-black" />
            </div>
            <span className="font-black text-[14px]">
              <span className="text-yellow-400">Loyalty</span><span className="text-white"> Nexus</span>
            </span>
          </Link>
          <Link href="/" className="text-[13px] text-white/40 hover:text-white transition-colors">← Back to Home</Link>
        </div>
      </div>

      <article className="max-w-3xl mx-auto px-4 py-16 prose prose-invert prose-sm">
        <div className="mb-10">
          <h1 className="text-3xl font-black mb-2">Privacy Policy</h1>
          <p className="text-white/40 text-[13px]">Last updated: 1 March 2026</p>
        </div>

        <div className="space-y-8 text-[14px] text-white/60 leading-relaxed">

          <section>
            <h2 className="text-white font-black text-[18px] mb-3">1. Who We Are</h2>
            <p>
              Loyalty Nexus (&quot;we&quot;, &quot;our&quot;, or &quot;the Company&quot;) operates the Loyalty Nexus platform accessible at
              loyaltynexus.ng and via our mobile application. We are incorporated under the laws of the
              Federal Republic of Nigeria and are committed to protecting the privacy of every user.
            </p>
          </section>

          <section>
            <h2 className="text-white font-black text-[18px] mb-3">2. Data We Collect</h2>
            <p className="mb-3">We collect the following categories of data:</p>
            <ul className="list-disc pl-5 space-y-2">
              <li><strong className="text-white">Account Data:</strong> Your phone number (MSISDN), name, email address, and password hash when you register.</li>
              <li><strong className="text-white">Recharge Data:</strong> MTN recharge events including amount, timestamp, and MSISDN, transmitted to us by MTN via secure API. We do not store your USSD PINs.</li>
              <li><strong className="text-white">Usage Data:</strong> Pages visited, AI tools used, prompts submitted (anonymised after 90 days), spin outcomes, and prize claims.</li>
              <li><strong className="text-white">Device Data:</strong> IP address, device type, OS version, and browser/app version for security and fraud prevention.</li>
              <li><strong className="text-white">Payment Data:</strong> For cash prize payouts we collect your bank account details or mobile money wallet. We do not store full card numbers.</li>
            </ul>
          </section>

          <section>
            <h2 className="text-white font-black text-[18px] mb-3">3. How We Use Your Data</h2>
            <ul className="list-disc pl-5 space-y-2">
              <li>To detect qualifying recharges and award Pulse Points in real time.</li>
              <li>To operate the Spin &amp; Win wheel, Regional Wars leaderboards, and Daily Draw.</li>
              <li>To provide access to AI Studio tools and store your generation history.</li>
              <li>To process prize payouts via bank transfer or mobile money.</li>
              <li>To send transactional SMS/push notifications (recharge confirmed, prize won, points balance).</li>
              <li>To prevent fraud, detect abuse, and enforce our Terms of Service.</li>
              <li>To improve our product using aggregated, anonymised analytics.</li>
            </ul>
          </section>

          <section>
            <h2 className="text-white font-black text-[18px] mb-3">4. Legal Basis for Processing</h2>
            <p>
              We process your data based on: (a) <strong className="text-white">Contract performance</strong> — to deliver the loyalty programme you signed up for;
              (b) <strong className="text-white">Legitimate interests</strong> — fraud prevention and product improvement;
              (c) <strong className="text-white">Consent</strong> — for marketing communications, which you may withdraw at any time.
            </p>
          </section>

          <section>
            <h2 className="text-white font-black text-[18px] mb-3">5. Data Sharing</h2>
            <p className="mb-3">We share your data only with:</p>
            <ul className="list-disc pl-5 space-y-2">
              <li><strong className="text-white">MTN Nigeria:</strong> To verify recharge eligibility and subscriber status.</li>
              <li><strong className="text-white">AI Providers:</strong> Your prompts are sent to Groq, Google Gemini, ElevenLabs, and Pollinations to generate outputs. These providers process data under their own privacy policies and do not retain prompts beyond the API request lifecycle.</li>
              <li><strong className="text-white">Payment Partners:</strong> Your bank details are shared with our payment processor solely for prize disbursement.</li>
              <li><strong className="text-white">Legal Authorities:</strong> When required by Nigerian law or a valid court order.</li>
            </ul>
            <p className="mt-3">We do <strong className="text-white">not</strong> sell, rent, or trade your personal data to third-party advertisers.</p>
          </section>

          <section>
            <h2 className="text-white font-black text-[18px] mb-3">6. Data Retention</h2>
            <p>
              Account data is retained for the lifetime of your account plus 2 years after closure.
              AI generation prompts are anonymised after 90 days. Recharge event logs are retained for 7 years
              for financial compliance purposes.
            </p>
          </section>

          <section>
            <h2 className="text-white font-black text-[18px] mb-3">7. Your Rights</h2>
            <p className="mb-3">Under applicable Nigerian data protection law (NDPR 2019) you have the right to:</p>
            <ul className="list-disc pl-5 space-y-2">
              <li>Access a copy of the data we hold about you.</li>
              <li>Correct inaccurate personal data.</li>
              <li>Request deletion of your account and associated data.</li>
              <li>Object to processing for marketing purposes.</li>
              <li>Lodge a complaint with the Nigeria Data Protection Commission (NDPC).</li>
            </ul>
            <p className="mt-3">To exercise your rights, email <a href="mailto:privacy@loyaltynexus.ng" className="text-yellow-400 hover:underline">privacy@loyaltynexus.ng</a> or visit Settings → Privacy in the app.</p>
          </section>

          <section>
            <h2 className="text-white font-black text-[18px] mb-3">8. Security</h2>
            <p>
              We use AES-256 encryption for data at rest, TLS 1.3 for data in transit, JWT-based session tokens,
              and rate-limiting on all API endpoints. We conduct quarterly security audits and operate a responsible
              disclosure programme at <a href="mailto:security@loyaltynexus.ng" className="text-yellow-400 hover:underline">security@loyaltynexus.ng</a>.
            </p>
          </section>

          <section>
            <h2 className="text-white font-black text-[18px] mb-3">9. Cookies</h2>
            <p>
              Our web platform uses essential cookies only (session token, CSRF protection). We do not use
              tracking cookies or third-party advertising pixels. You can disable non-essential cookies in
              your browser settings without affecting platform functionality.
            </p>
          </section>

          <section>
            <h2 className="text-white font-black text-[18px] mb-3">10. Changes to This Policy</h2>
            <p>
              We may update this policy from time to time. Significant changes will be communicated via
              in-app notification and email. Continued use of the platform after notification constitutes
              acceptance of the updated policy.
            </p>
          </section>

          <section>
            <h2 className="text-white font-black text-[18px] mb-3">11. Contact</h2>
            <p>
              Data Protection Officer: <a href="mailto:privacy@loyaltynexus.ng" className="text-yellow-400 hover:underline">privacy@loyaltynexus.ng</a><br/>
              Loyalty Nexus Limited, 4th Floor, 1234 Victoria Island, Lagos, Nigeria.
            </p>
          </section>
        </div>
      </article>
    </main>
  );
}

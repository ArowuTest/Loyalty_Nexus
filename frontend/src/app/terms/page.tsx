import React from "react";
import Link from "next/link";
import { Zap } from "lucide-react";
import type { Metadata } from "next";

export const metadata: Metadata = {
  title: "Terms of Service | Loyalty Nexus",
  description: "Terms and conditions governing use of the Loyalty Nexus platform.",
};

export default function TermsPage() {
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

      <article className="max-w-3xl mx-auto px-4 py-16">
        <div className="mb-10">
          <h1 className="text-3xl font-black mb-2">Terms of Service</h1>
          <p className="text-white/40 text-[13px]">Last updated: 1 March 2026 · Effective: 1 April 2026</p>
        </div>

        <div className="space-y-8 text-[14px] text-white/60 leading-relaxed">

          <section>
            <h2 className="text-white font-black text-[18px] mb-3">1. Acceptance of Terms</h2>
            <p>
              By registering for, accessing, or using the Loyalty Nexus platform (&quot;the Service&quot;) — including our
              website, mobile application, USSD channel, and API — you agree to be bound by these Terms of
              Service (&quot;Terms&quot;). If you do not agree, do not use the Service.
            </p>
          </section>

          <section>
            <h2 className="text-white font-black text-[18px] mb-3">2. Eligibility</h2>
            <ul className="list-disc pl-5 space-y-2">
              <li>You must be at least 18 years of age.</li>
              <li>You must be the registered owner or authorised user of the MTN Nigeria line used for recharges.</li>
              <li>You must reside in Nigeria or be a Nigerian citizen recharging a Nigerian MSISDN.</li>
              <li>Employees, contractors, and family members of Loyalty Nexus Limited and MTN Nigeria are not eligible to participate in cash prize draws.</li>
            </ul>
          </section>

          <section>
            <h2 className="text-white font-black text-[18px] mb-3">3. Accounts</h2>
            <p>
              You are responsible for maintaining the confidentiality of your login credentials. You must notify
              us immediately at <a href="mailto:security@loyaltynexus.ng" className="text-yellow-400 hover:underline">security@loyaltynexus.ng</a> if you suspect unauthorised access.
              One account per person is permitted. Duplicate accounts will be merged or suspended.
            </p>
          </section>

          <section>
            <h2 className="text-white font-black text-[18px] mb-3">4. Pulse Points</h2>
            <ul className="list-disc pl-5 space-y-2">
              <li>Pulse Points are earned at a rate of <strong className="text-white">1 Pulse Point per ₦250 recharged</strong> on a qualifying MTN line.</li>
              <li>The minimum qualifying recharge is <strong className="text-white">₦1,000</strong>.</li>
              <li>Pulse Points have no cash value and cannot be transferred, sold, or exchanged for cash.</li>
              <li>Points expire <strong className="text-white">12 months</strong> after the recharge event that generated them.</li>
              <li>We reserve the right to adjust the earning rate with 14 days' notice published in-app.</li>
            </ul>
          </section>

          <section>
            <h2 className="text-white font-black text-[18px] mb-3">5. Spin &amp; Win</h2>
            <ul className="list-disc pl-5 space-y-2">
              <li>Each qualifying recharge of ₦1,000 or more earns one (1) free wheel spin.</li>
              <li>Wheel outcomes are determined by a provably fair random number generator audited quarterly.</li>
              <li>Cash prizes are disbursed within 5 working days via bank transfer to the account registered in your profile.</li>
              <li>Unclaimed prizes expire after <strong className="text-white">30 days</strong> from the spin date.</li>
              <li>Loyalty Nexus reserves the right to withhold prizes pending identity verification if fraud is suspected.</li>
            </ul>
          </section>

          <section>
            <h2 className="text-white font-black text-[18px] mb-3">6. Daily Draw</h2>
            <ul className="list-disc pl-5 space-y-2">
              <li>One grand prize draw is conducted daily at 23:59 WAT among all eligible users.</li>
              <li>Eligibility requires at least one qualifying recharge in the previous 24 hours.</li>
              <li>Winners are announced via in-app notification and SMS within 30 minutes of the draw.</li>
              <li>Grand prize values and structures are published on the prizes page and may change monthly.</li>
            </ul>
          </section>

          <section>
            <h2 className="text-white font-black text-[18px] mb-3">7. AI Studio</h2>
            <ul className="list-disc pl-5 space-y-2">
              <li>AI Studio tools are unlocked using Pulse Points at published rates which may change with notice.</li>
              <li>You retain ownership of outputs you generate. However, you grant Loyalty Nexus a limited licence to display your outputs within the platform (e.g., public galleries) unless you set them to private.</li>
              <li>You may not use AI Studio to generate content that is illegal, defamatory, sexually explicit, or infringes third-party intellectual property.</li>
              <li>AI-generated outputs may be inaccurate. Do not rely on them for medical, legal, or financial decisions without independent verification.</li>
              <li>We reserve the right to rate-limit or suspend AI Studio access in cases of suspected abuse.</li>
            </ul>
          </section>

          <section>
            <h2 className="text-white font-black text-[18px] mb-3">8. Regional Wars</h2>
            <ul className="list-disc pl-5 space-y-2">
              <li>Regional Wars seasons run for defined periods published in-app.</li>
              <li>Leaderboard rankings are based on Pulse Points earned during the season period only.</li>
              <li>Loyalty Nexus reserves the right to disqualify users who engage in coordinated manipulation, bot activity, or abuse of the earning system.</li>
              <li>Prize pools and distributions are published before each season commences.</li>
            </ul>
          </section>

          <section>
            <h2 className="text-white font-black text-[18px] mb-3">9. Prohibited Conduct</h2>
            <p>You agree not to:</p>
            <ul className="list-disc pl-5 mt-2 space-y-2">
              <li>Use automated tools, bots, or scripts to earn Points or spins.</li>
              <li>Create multiple accounts to circumvent limits or gain unfair advantage.</li>
              <li>Attempt to reverse-engineer, scrape, or interfere with our platform.</li>
              <li>Use the Service for any unlawful purpose under Nigerian law.</li>
              <li>Submit false bank or identity details to fraudulently claim prizes.</li>
            </ul>
          </section>

          <section>
            <h2 className="text-white font-black text-[18px] mb-3">10. Termination</h2>
            <p>
              We may suspend or terminate your account at any time for material breach of these Terms.
              You may delete your account at any time via Settings → Delete Account.
              Upon termination, unspent Pulse Points are forfeited and unclaimed prizes expire after 30 days.
            </p>
          </section>

          <section>
            <h2 className="text-white font-black text-[18px] mb-3">11. Liability</h2>
            <p>
              The Service is provided &quot;as is&quot;. To the maximum extent permitted by Nigerian law, Loyalty Nexus
              excludes liability for indirect, incidental, or consequential damages. Our total liability to
              you in any 12-month period shall not exceed ₦50,000 or the value of prizes legitimately won,
              whichever is higher.
            </p>
          </section>

          <section>
            <h2 className="text-white font-black text-[18px] mb-3">12. Governing Law</h2>
            <p>
              These Terms are governed by the laws of the Federal Republic of Nigeria.
              Any disputes shall be resolved by the courts of Lagos State, Nigeria.
            </p>
          </section>

          <section>
            <h2 className="text-white font-black text-[18px] mb-3">13. Contact</h2>
            <p>
              For questions about these Terms: <a href="mailto:legal@loyaltynexus.ng" className="text-yellow-400 hover:underline">legal@loyaltynexus.ng</a><br />
              Loyalty Nexus Limited, 4th Floor, 1234 Victoria Island, Lagos, Nigeria.
            </p>
          </section>
        </div>
      </article>
    </main>
  );
}

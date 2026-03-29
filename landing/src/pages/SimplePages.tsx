import React from "react";
import NavBar from "@/components/NavBar";
import Footer from "@/components/Footer";
import { Zap } from "lucide-react";

interface SimplePageProps {
  title: string;
  emoji: string;
  body: string;
}

export function SimplePage({ title, emoji, body }: SimplePageProps) {
  return (
    <div className="min-h-screen bg-surface-0 dark">
      <NavBar />
      <div className="max-w-3xl mx-auto px-4 sm:px-6 pt-32 pb-24">
        <div className="text-center mb-12">
          <div className="text-6xl mb-4">{emoji}</div>
          <h1 className="text-4xl sm:text-5xl font-black text-foreground mb-4">{title}</h1>
          <p className="text-base text-muted-foreground leading-relaxed">{body}</p>
        </div>
        <div className="glass rounded-3xl border border-white/[0.09] p-8 text-center">
          <div className="w-12 h-12 rounded-2xl bg-gold flex items-center justify-center glow-gold mx-auto mb-4">
            <Zap className="w-6 h-6 text-black" />
          </div>
          <p className="text-muted-foreground text-sm leading-relaxed">
            This page is currently being built. Please check back soon or reach out to us at{" "}
            <a href="mailto:hello@loyaltynexus.ng" className="text-primary hover:underline font-semibold">
              hello@loyaltynexus.ng
            </a>
          </p>
        </div>
      </div>
      <Footer />
    </div>
  );
}

export function AboutPage() {
  return <SimplePage title="About Us" emoji="🇳🇬" body="We're building the most rewarding MTN loyalty platform in Africa — combining AI tools, instant prizes, and community." />;
}

export function BlogPage() {
  return <SimplePage title="Blog" emoji="✍️" body="Tips, insights, and updates from the Loyalty Nexus team." />;
}

export function CareersPage() {
  return <SimplePage title="Careers" emoji="🚀" body="Help us reward millions of Nigerians. We're always looking for great talent." />;
}

export function PrivacyPage() {
  return <SimplePage title="Privacy Policy" emoji="🔒" body="Your privacy matters. We collect only what's needed to run the platform and never sell your data." />;
}

export function TermsPage() {
  return <SimplePage title="Terms of Service" emoji="📄" body="By using Loyalty Nexus you agree to our terms. Please read them carefully." />;
}

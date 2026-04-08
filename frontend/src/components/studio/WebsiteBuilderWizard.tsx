"use client";
import React, { useState, useRef, useCallback } from "react";
import { motion, AnimatePresence } from "framer-motion";
import {
  X, ArrowRight, ArrowLeft, Upload, Trash2, Globe,
  CheckCircle, Copy, ExternalLink, RotateCcw, Loader2,
  Building2, Briefcase, UtensilsCrossed, Camera, PartyPopper,
  Church, GraduationCap, ShoppingBag, Sparkles, Check
} from "lucide-react";
import api from "@/lib/api";

// ─── Types ────────────────────────────────────────────────────────────────────
interface WebsitePhoto { base64: string; caption: string; preview: string; }

interface SiteType {
  id: string; label: string; icon: React.ReactNode;
  desc: string; color: string;
  fields: Array<{ key: string; label: string; placeholder?: string; required?: boolean; multiline?: boolean }>;
}

// ─── Site types config ────────────────────────────────────────────────────────
const SITE_TYPES: SiteType[] = [
  {
    id: "shop", label: "Shop / Catalogue", color: "#7C3AED",
    icon: <ShoppingBag size={28} />,
    desc: "Sell products, show prices, take orders via WhatsApp",
    fields: [
      { key: "business_name", label: "Business Name", required: true, placeholder: "e.g. Queen's Styles" },
      { key: "tagline", label: "Tagline / Slogan", placeholder: "e.g. Fashion for every queen" },
      { key: "what_you_sell", label: "What do you sell?", required: true, placeholder: "e.g. Women's clothing, Ankara, lace fabrics" },
      { key: "location", label: "Location / Address", required: true, placeholder: "e.g. 23 Bode Thomas, Surulere, Lagos" },
      { key: "whatsapp", label: "WhatsApp Number", required: true, placeholder: "e.g. 08031234567" },
      { key: "phone", label: "Phone Number", placeholder: "e.g. 08031234567" },
      { key: "hours", label: "Opening Hours", placeholder: "e.g. Mon–Sat 9am–7pm" },
      { key: "price_range", label: "Price Range", placeholder: "e.g. ₦5,000 – ₦50,000" },
    ]
  },
  {
    id: "corporate", label: "Corporate / Startup", color: "#1D4ED8",
    icon: <Building2 size={28} />,
    desc: "Professional company website for your business or startup",
    fields: [
      { key: "company_name", label: "Company Name", required: true, placeholder: "e.g. TechVault Solutions" },
      { key: "tagline", label: "Mission / Tagline", required: true, placeholder: "e.g. Powering African businesses" },
      { key: "what_you_do", label: "What does your company do?", required: true, placeholder: "e.g. We build software for SMEs", multiline: true },
      { key: "services", label: "Key Services", required: true, placeholder: "e.g. Web development, IT consulting, Cloud hosting" },
      { key: "location", label: "Office Location", placeholder: "e.g. Victoria Island, Lagos" },
      { key: "email", label: "Email Address", placeholder: "e.g. hello@company.com" },
      { key: "phone", label: "Phone Number", placeholder: "e.g. 08031234567" },
      { key: "whatsapp", label: "WhatsApp Number", placeholder: "e.g. 08031234567" },
    ]
  },
  {
    id: "professional", label: "Professional Services", color: "#065F46",
    icon: <Briefcase size={28} />,
    desc: "Lawyers, doctors, consultants, accountants",
    fields: [
      { key: "name", label: "Your Full Name", required: true, placeholder: "e.g. Dr. Adaeze Okonkwo" },
      { key: "title", label: "Professional Title", required: true, placeholder: "e.g. Corporate Lawyer / Medical Doctor" },
      { key: "services", label: "Services You Offer", required: true, placeholder: "e.g. Contract drafting, Family law, Court representation", multiline: true },
      { key: "experience", label: "Years of Experience", placeholder: "e.g. 15 years" },
      { key: "qualifications", label: "Qualifications / Certifications", placeholder: "e.g. LLB, BL, MBA" },
      { key: "location", label: "Office Location", placeholder: "e.g. Ikoyi, Lagos" },
      { key: "whatsapp", label: "WhatsApp / Phone", required: true, placeholder: "e.g. 08031234567" },
      { key: "email", label: "Email Address", placeholder: "e.g. dr.adaeze@email.com" },
    ]
  },
  {
    id: "restaurant", label: "Restaurant / Food", color: "#92400E",
    icon: <UtensilsCrossed size={28} />,
    desc: "Restaurant, caterer, bakery, food delivery",
    fields: [
      { key: "restaurant_name", label: "Restaurant / Business Name", required: true, placeholder: "e.g. Mama Titi's Kitchen" },
      { key: "cuisine_type", label: "Cuisine / Food Type", required: true, placeholder: "e.g. Nigerian, Yoruba dishes, Continental" },
      { key: "signature_dishes", label: "Signature Dishes & Prices", required: true, placeholder: "e.g. Jollof Rice ₦1500, Egusi Soup ₦2000, Grilled Fish ₦3500", multiline: true },
      { key: "location", label: "Location / Address", required: true, placeholder: "e.g. 15 Allen Avenue, Ikeja, Lagos" },
      { key: "hours", label: "Opening Hours", required: true, placeholder: "e.g. Daily 8am–10pm" },
      { key: "whatsapp", label: "WhatsApp / Order Line", required: true, placeholder: "e.g. 08031234567" },
      { key: "delivery", label: "Delivery Info", placeholder: "e.g. Free delivery within 5km, ₦500 elsewhere" },
    ]
  },
  {
    id: "portfolio", label: "Portfolio", color: "#0E7490",
    icon: <Camera size={28} />,
    desc: "Photographers, designers, artists, creatives",
    fields: [
      { key: "name", label: "Your Name / Brand Name", required: true, placeholder: "e.g. Tunde Visuals" },
      { key: "profession", label: "What you do", required: true, placeholder: "e.g. Wedding Photographer / Graphic Designer" },
      { key: "specialties", label: "Specialties / Services", required: true, placeholder: "e.g. Wedding photography, portraits, brand shoots", multiline: true },
      { key: "pricing", label: "Pricing / Packages", placeholder: "e.g. Starting from ₦80,000 for weddings" },
      { key: "location", label: "Based in", placeholder: "e.g. Abuja, available nationwide" },
      { key: "whatsapp", label: "WhatsApp", required: true, placeholder: "e.g. 08031234567" },
      { key: "instagram", label: "Instagram Handle", placeholder: "e.g. @tundevisuals" },
    ]
  },
  {
    id: "events", label: "Events", color: "#86198F",
    icon: <PartyPopper size={28} />,
    desc: "Event planners, DJs, decorators, catering",
    fields: [
      { key: "business_name", label: "Business Name", required: true, placeholder: "e.g. GlamEvents NG" },
      { key: "event_types", label: "Events You Cover", required: true, placeholder: "e.g. Weddings, birthdays, corporate events, burials" },
      { key: "services", label: "Services Offered", required: true, placeholder: "e.g. Decoration, catering, photography, MC, DJ", multiline: true },
      { key: "pricing", label: "Starting Prices", placeholder: "e.g. Decoration from ₦50,000, full package from ₦200,000" },
      { key: "location", label: "Location / Coverage Area", placeholder: "e.g. Lagos & Ogun State" },
      { key: "whatsapp", label: "WhatsApp", required: true, placeholder: "e.g. 08031234567" },
    ]
  },
  {
    id: "church", label: "Church / Religious", color: "#1E40AF",
    icon: <Church size={28} />,
    desc: "Church, ministry, mosque, religious organisation",
    fields: [
      { key: "church_name", label: "Church / Ministry Name", required: true, placeholder: "e.g. Grace Assembly Church" },
      { key: "tagline", label: "Mission Statement", placeholder: "e.g. A place of hope, healing and transformation" },
      { key: "service_times", label: "Service Times", required: true, placeholder: "e.g. Sunday 8am & 10:30am, Wednesday 6pm", multiline: true },
      { key: "location", label: "Address", required: true, placeholder: "e.g. 45 Lagos Road, Ibadan" },
      { key: "pastor", label: "Senior Pastor / Leader", placeholder: "e.g. Pastor John & Mrs. Faith Adeyemi" },
      { key: "phone", label: "Phone Number", placeholder: "e.g. 08031234567" },
      { key: "whatsapp", label: "WhatsApp", required: true, placeholder: "e.g. 08031234567" },
    ]
  },
  {
    id: "education", label: "Education", color: "#065F46",
    icon: <GraduationCap size={28} />,
    desc: "Schools, lesson teachers, training centres",
    fields: [
      { key: "school_name", label: "School / Centre Name", required: true, placeholder: "e.g. Bright Minds Tutorial" },
      { key: "what_you_teach", label: "Subjects / Courses", required: true, placeholder: "e.g. Mathematics, English, Sciences for JSS/SSS" },
      { key: "schedule", label: "Class Schedule", placeholder: "e.g. Mon–Fri after school, Sat 8am–2pm" },
      { key: "fees", label: "Fees", placeholder: "e.g. ₦8,000/month per subject, ₦20,000 full package" },
      { key: "location", label: "Location", required: true, placeholder: "e.g. 12 School Road, Yaba, Lagos" },
      { key: "whatsapp", label: "WhatsApp / Enrolment Line", required: true, placeholder: "e.g. 08031234567" },
      { key: "qualifications", label: "Teacher Qualifications", placeholder: "e.g. B.Ed Mathematics, 10 years experience" },
    ]
  },
];

// ─── Image compression ────────────────────────────────────────────────────────
async function compressImage(file: File): Promise<string> {
  return new Promise((resolve, reject) => {
    const img = new Image();
    const url = URL.createObjectURL(file);
    img.onload = () => {
      const canvas = document.createElement("canvas");
      const MAX = 480;
      let { width, height } = img;
      if (width > height) { if (width > MAX) { height = (height * MAX) / width; width = MAX; } }
      else { if (height > MAX) { width = (width * MAX) / height; height = MAX; } }
      canvas.width = width; canvas.height = height;
      const ctx = canvas.getContext("2d")!;
      ctx.drawImage(img, 0, 0, width, height);
      // Strip data URI prefix — we only want the base64 string
      const dataURL = canvas.toDataURL("image/jpeg", 0.75);
      URL.revokeObjectURL(url);
      resolve(dataURL.replace(/^data:image\/jpeg;base64,/, ""));
    };
    img.onerror = reject;
    img.src = url;
  });
}

// ─── Main component ───────────────────────────────────────────────────────────
interface WebsiteBuilderWizardProps {
  pointBalance: number;
  onClose: () => void;
  onSuccess?: (genId: string) => void;
}

export default function WebsiteBuilderWizard({ pointBalance, onClose, onSuccess }: WebsiteBuilderWizardProps) {
  const [step, setStep] = useState<1 | 2 | 3 | 4 | 5>(1);
  const [selectedType, setSelectedType] = useState<SiteType | null>(null);
  const [fields, setFields] = useState<Record<string, string>>({});
  const [photos, setPhotos] = useState<WebsitePhoto[]>([]);
  const [isGenerating, setIsGenerating] = useState(false);
  const [genId, setGenId] = useState<string | null>(null);
  const [siteStatus, setSiteStatus] = useState<"pending" | "processing" | "completed" | "failed">("pending");
  const [error, setError] = useState<string | null>(null);
  const [copied, setCopied] = useState(false);
  const fileRef = useRef<HTMLInputElement>(null);
  const pollRef = useRef<NodeJS.Timeout | null>(null);

  const POINT_COST = 25;
  const canAfford = pointBalance >= POINT_COST;

  // ── Step validation ──────────────────────────────────────────────────────
  const requiredFilled = useCallback(() => {
    if (!selectedType) return false;
    return selectedType.fields
      .filter(f => f.required)
      .every(f => (fields[f.key] || "").trim().length > 0);
  }, [selectedType, fields]);

  // ── Photo upload ─────────────────────────────────────────────────────────
  const handlePhotoAdd = async (e: React.ChangeEvent<HTMLInputElement>) => {
    const files = Array.from(e.target.files || []);
    const remaining = 6 - photos.length;
    const toProcess = files.slice(0, remaining);
    for (const file of toProcess) {
      try {
        const b64 = await compressImage(file);
        const preview = `data:image/jpeg;base64,${b64}`;
        setPhotos(prev => [...prev, { base64: b64, caption: "", preview }]);
      } catch { /* skip failed */ }
    }
    e.target.value = "";
  };

  const updateCaption = (idx: number, caption: string) =>
    setPhotos(prev => prev.map((p, i) => i === idx ? { ...p, caption } : p));

  const removePhoto = (idx: number) =>
    setPhotos(prev => prev.filter((_, i) => i !== idx));

  // ── Polling ──────────────────────────────────────────────────────────────
  const startPolling = (id: string) => {
    if (pollRef.current) clearInterval(pollRef.current);
    pollRef.current = setInterval(async () => {
      try {
        const result = await api.getGenerationStatus(id) as { status: string; error_message?: string };
        if (result.status === "completed") {
          clearInterval(pollRef.current!);
          setSiteStatus("completed");
          setStep(5);
        } else if (result.status === "failed") {
          clearInterval(pollRef.current!);
          setSiteStatus("failed");
          const reason = result.error_message || "Generation failed. Please try again.";
          setError(reason);
          setStep(4);
          setIsGenerating(false);
        }
      } catch { /* ignore transient errors */ }
    }, 3000);
  };

  // ── Generate ─────────────────────────────────────────────────────────────
  const handleGenerate = async () => {
    if (!selectedType || !canAfford) return;
    setIsGenerating(true);
    setError(null);
    try {
      const result = await api.buildWebsite({
        site_type: selectedType.id,
        fields,
        photos: photos.map(p => ({ base64: p.base64, caption: p.caption })),
      }) as { generation_id: string; status: string };
      setGenId(result.generation_id);
      setSiteStatus("pending");
      setStep(5);
      startPolling(result.generation_id);
    } catch (e: unknown) {
      const msg = e instanceof Error ? e.message : "Generation failed";
      setError(msg);
      setIsGenerating(false);
    }
  };

  const siteURL = genId
    ? `${window.location.origin}/s/${genId}`
    : null;
  const backendSiteURL = genId
    ? `https://loyalty-nexus-api.onrender.com/s/${genId}`
    : null;

  const copyLink = () => {
    if (backendSiteURL) {
      navigator.clipboard.writeText(backendSiteURL);
      setCopied(true);
      setTimeout(() => setCopied(false), 2000);
    }
  };

  // ── Render ────────────────────────────────────────────────────────────────
  return (
    <div className="fixed inset-0 z-50 flex items-end md:items-center justify-center"
      style={{ background: "rgba(0,0,0,0.85)", backdropFilter: "blur(8px)" }}>
      <motion.div
        initial={{ opacity: 0, y: 40 }} animate={{ opacity: 1, y: 0 }} exit={{ opacity: 0, y: 40 }}
        className="relative w-full max-w-lg mx-auto flex flex-col"
        style={{
          background: "linear-gradient(135deg, #13141f 0%, #0d0f1a 100%)",
          border: "1px solid rgba(255,255,255,0.08)",
          borderRadius: "24px 24px 0 0",
          maxHeight: "92dvh",
          overflow: "hidden",
        }}
      >
        {/* ── Header ─────────────────────────────────────────────────────── */}
        <div className="flex items-center justify-between px-5 pt-5 pb-3 flex-shrink-0"
          style={{ borderBottom: "1px solid rgba(255,255,255,0.06)" }}>
          <div className="flex items-center gap-3">
            <div className="w-9 h-9 rounded-xl flex items-center justify-center text-lg"
              style={{ background: "rgba(99,102,241,0.15)", border: "1px solid rgba(99,102,241,0.25)" }}>
              🌐
            </div>
            <div>
              <p className="text-white font-black text-sm">Website Builder</p>
              <p className="text-white/40 text-[10px]">Step {step} of {step < 5 ? "4" : "5"}</p>
            </div>
          </div>
          <div className="flex items-center gap-3">
            {step < 5 && (
              <span className="text-[11px] px-2.5 py-1 rounded-full font-bold"
                style={{ background: canAfford ? "rgba(245,166,35,0.12)" : "rgba(239,68,68,0.12)", color: canAfford ? "#F5A623" : "#ef4444", border: `1px solid ${canAfford ? "rgba(245,166,35,0.2)" : "rgba(239,68,68,0.2)"}` }}>
                {POINT_COST} pts
              </span>
            )}
            <button onClick={onClose} className="w-8 h-8 rounded-full flex items-center justify-center hover:bg-white/10 transition-colors">
              <X size={16} className="text-white/50" />
            </button>
          </div>
        </div>

        {/* ── Step indicator ─────────────────────────────────────────────── */}
        {step < 5 && (
          <div className="px-5 pt-3 pb-2 flex gap-1.5 flex-shrink-0">
            {[1, 2, 3, 4].map(s => (
              <div key={s} className="h-1 flex-1 rounded-full transition-all duration-500"
                style={{ background: s <= step ? "#6366f1" : "rgba(255,255,255,0.08)" }} />
            ))}
          </div>
        )}

        {/* ── Body ───────────────────────────────────────────────────────── */}
        <div className="flex-1 overflow-y-auto" style={{ scrollbarWidth: "none" }}>
          <AnimatePresence mode="wait">

            {/* ─── Step 1: Choose type ──────────────────────────────────── */}
            {step === 1 && (
              <motion.div key="s1" initial={{ opacity: 0, x: 20 }} animate={{ opacity: 1, x: 0 }} exit={{ opacity: 0, x: -20 }}
                className="p-5 space-y-4">
                <div>
                  <h2 className="text-white font-black text-xl">What type of website?</h2>
                  <p className="text-white/40 text-sm mt-1">Choose the one that best matches your business</p>
                </div>
                <div className="grid grid-cols-2 gap-2.5">
                  {SITE_TYPES.map(type => (
                    <button key={type.id} onClick={() => { setSelectedType(type); setFields({}); setStep(2); }}
                      className="relative flex flex-col items-start gap-2 p-3.5 rounded-2xl text-left transition-all active:scale-95"
                      style={{
                        background: selectedType?.id === type.id ? `${type.color}20` : "rgba(255,255,255,0.04)",
                        border: `1px solid ${selectedType?.id === type.id ? type.color + "50" : "rgba(255,255,255,0.08)"}`,
                      }}>
                      <div className="w-10 h-10 rounded-xl flex items-center justify-center"
                        style={{ background: `${type.color}20`, color: type.color }}>
                        {type.icon}
                      </div>
                      <div>
                        <p className="text-white font-bold text-sm leading-tight">{type.label}</p>
                        <p className="text-white/40 text-[10px] mt-0.5 leading-snug">{type.desc}</p>
                      </div>
                    </button>
                  ))}
                </div>
              </motion.div>
            )}

            {/* ─── Step 2: Details form ─────────────────────────────────── */}
            {step === 2 && selectedType && (
              <motion.div key="s2" initial={{ opacity: 0, x: 20 }} animate={{ opacity: 1, x: 0 }} exit={{ opacity: 0, x: -20 }}
                className="p-5 space-y-4">
                <div>
                  <div className="flex items-center gap-2 mb-1">
                    <span className="text-xl">{selectedType.id === "shop" ? "🛍️" : selectedType.id === "corporate" ? "🏢" : "✏️"}</span>
                    <h2 className="text-white font-black text-xl">Your details</h2>
                  </div>
                  <p className="text-white/40 text-sm">Fill in as much as you can — the more detail, the better your website</p>
                </div>
                <div className="space-y-3">
                  {selectedType.fields.map(f => (
                    <div key={f.key}>
                      <label className="text-white/60 text-xs font-bold uppercase tracking-wider block mb-1.5">
                        {f.label} {f.required && <span className="text-red-400">*</span>}
                      </label>
                      {f.multiline ? (
                        <textarea
                          rows={3}
                          className="w-full rounded-xl px-3.5 py-3 text-sm text-white placeholder:text-white/20 outline-none resize-none"
                          style={{ background: "rgba(255,255,255,0.05)", border: "1px solid rgba(255,255,255,0.1)" }}
                          placeholder={f.placeholder}
                          value={fields[f.key] || ""}
                          onChange={e => setFields(p => ({ ...p, [f.key]: e.target.value }))}
                        />
                      ) : (
                        <input
                          type="text"
                          className="w-full rounded-xl px-3.5 py-3 text-sm text-white placeholder:text-white/20 outline-none"
                          style={{ background: "rgba(255,255,255,0.05)", border: "1px solid rgba(255,255,255,0.1)" }}
                          placeholder={f.placeholder}
                          value={fields[f.key] || ""}
                          onChange={e => setFields(p => ({ ...p, [f.key]: e.target.value }))}
                        />
                      )}
                    </div>
                  ))}
                </div>
              </motion.div>
            )}

            {/* ─── Step 3: Photos ───────────────────────────────────────── */}
            {step === 3 && (
              <motion.div key="s3" initial={{ opacity: 0, x: 20 }} animate={{ opacity: 1, x: 0 }} exit={{ opacity: 0, x: -20 }}
                className="p-5 space-y-4">
                <div>
                  <h2 className="text-white font-black text-xl">Add photos <span className="text-white/30 font-normal">(optional)</span></h2>
                  <p className="text-white/40 text-sm mt-1">Up to 6 photos. Add a short description for each one.</p>
                </div>
                <input ref={fileRef} type="file" accept="image/*" multiple className="hidden" onChange={handlePhotoAdd} />
                {photos.length < 6 && (
                  <button onClick={() => fileRef.current?.click()}
                    className="w-full rounded-2xl flex flex-col items-center justify-center gap-2 py-8 transition-colors active:scale-95"
                    style={{ background: "rgba(255,255,255,0.03)", border: "2px dashed rgba(255,255,255,0.12)" }}>
                    <div className="w-12 h-12 rounded-2xl flex items-center justify-center"
                      style={{ background: "rgba(99,102,241,0.15)" }}>
                      <Upload size={22} className="text-indigo-400" />
                    </div>
                    <div>
                      <p className="text-white/70 text-sm font-bold">Tap to add photos</p>
                      <p className="text-white/30 text-xs">{photos.length}/6 added · Auto-compressed</p>
                    </div>
                  </button>
                )}
                {photos.map((photo, idx) => (
                  <div key={idx} className="flex gap-3 items-start p-3 rounded-2xl"
                    style={{ background: "rgba(255,255,255,0.04)", border: "1px solid rgba(255,255,255,0.08)" }}>
                    <img src={photo.preview} alt="" className="w-16 h-16 object-cover rounded-xl flex-shrink-0" />
                    <div className="flex-1 min-w-0">
                      <input
                        className="w-full rounded-xl px-3 py-2 text-sm text-white placeholder:text-white/30 outline-none mb-2"
                        style={{ background: "rgba(255,255,255,0.06)", border: "1px solid rgba(255,255,255,0.1)" }}
                        placeholder="Describe this photo (product name, price...)"
                        value={photo.caption}
                        onChange={e => updateCaption(idx, e.target.value)}
                      />
                      <p className="text-white/25 text-[10px]">Photo {idx + 1} of {photos.length}</p>
                    </div>
                    <button onClick={() => removePhoto(idx)} className="text-white/30 hover:text-red-400 transition-colors mt-1">
                      <Trash2 size={15} />
                    </button>
                  </div>
                ))}
              </motion.div>
            )}

            {/* ─── Step 4: Review ───────────────────────────────────────── */}
            {step === 4 && selectedType && (
              <motion.div key="s4" initial={{ opacity: 0, x: 20 }} animate={{ opacity: 1, x: 0 }} exit={{ opacity: 0, x: -20 }}
                className="p-5 space-y-4">
                <div>
                  <h2 className="text-white font-black text-xl">Ready to build!</h2>
                  <p className="text-white/40 text-sm mt-1">Here's what we'll create for you</p>
                </div>

                {/* Summary card */}
                <div className="rounded-2xl p-4 space-y-3"
                  style={{ background: "rgba(255,255,255,0.04)", border: "1px solid rgba(255,255,255,0.08)" }}>
                  <div className="flex items-center gap-3">
                    <div className="w-10 h-10 rounded-xl flex items-center justify-center"
                      style={{ background: `${selectedType.color}20`, color: selectedType.color }}>
                      {selectedType.icon}
                    </div>
                    <div>
                      <p className="text-white font-black text-sm">{selectedType.label} Website</p>
                      <p className="text-white/40 text-xs">{fields[Object.keys(fields)[0]] || "Your business"}</p>
                    </div>
                  </div>
                  <div className="grid grid-cols-3 gap-2">
                    {[
                      { label: "Fields", value: Object.values(fields).filter(Boolean).length },
                      { label: "Photos", value: photos.length },
                      { label: "Cost", value: `${POINT_COST}pts` },
                    ].map(stat => (
                      <div key={stat.label} className="text-center py-2 rounded-xl"
                        style={{ background: "rgba(255,255,255,0.04)" }}>
                        <p className="text-white font-black text-lg">{stat.value}</p>
                        <p className="text-white/40 text-[10px]">{stat.label}</p>
                      </div>
                    ))}
                  </div>
                </div>

                {/* What to expect */}
                <div className="rounded-2xl p-4 space-y-2"
                  style={{ background: "rgba(99,102,241,0.06)", border: "1px solid rgba(99,102,241,0.15)" }}>
                  <p className="text-indigo-300 font-bold text-xs uppercase tracking-wider mb-2">What you'll get</p>
                  {[
                    "✨ Professional, mobile-first design",
                    "💬 WhatsApp contact buttons throughout",
                    "🌐 Instant shareable link",
                    "📱 Works on any phone browser",
                  ].map(item => (
                    <p key={item} className="text-white/70 text-sm flex items-center gap-2">{item}</p>
                  ))}
                </div>

                {!canAfford && (
                  <div className="rounded-2xl p-3 text-center"
                    style={{ background: "rgba(239,68,68,0.08)", border: "1px solid rgba(239,68,68,0.2)" }}>
                    <p className="text-red-400 text-sm font-bold">Not enough Pulse Points</p>
                    <p className="text-red-400/60 text-xs mt-0.5">You need {POINT_COST} pts · You have {pointBalance} pts</p>
                  </div>
                )}
                {error && (
                  <div className="rounded-2xl p-3" style={{ background: "rgba(239,68,68,0.08)", border: "1px solid rgba(239,68,68,0.2)" }}>
                    <p className="text-red-400 text-sm">{error}</p>
                  </div>
                )}
              </motion.div>
            )}

            {/* ─── Step 5: Result ───────────────────────────────────────── */}
            {step === 5 && (
              <motion.div key="s5" initial={{ opacity: 0, scale: 0.97 }} animate={{ opacity: 1, scale: 1 }}
                className="p-5 space-y-4">
                {siteStatus !== "completed" ? (
                  <div className="flex flex-col items-center text-center py-8 space-y-4">
                    <div className="w-16 h-16 rounded-2xl flex items-center justify-center"
                      style={{ background: "rgba(99,102,241,0.15)" }}>
                      <Loader2 size={32} className="text-indigo-400 animate-spin" />
                    </div>
                    <div>
                      <h2 className="text-white font-black text-xl">Building your website...</h2>
                      <p className="text-white/40 text-sm mt-1">AI is designing your site — usually 10–15 seconds</p>
                    </div>
                    <div className="flex gap-1">
                      {[0.1, 0.2, 0.3].map(d => (
                        <motion.div key={d} className="w-2 h-2 rounded-full bg-indigo-400"
                          animate={{ scale: [1, 1.4, 1] }} transition={{ duration: 0.8, delay: d, repeat: Infinity }} />
                      ))}
                    </div>
                  </div>
                ) : (
                  <>
                    <div className="flex items-center gap-3">
                      <div className="w-10 h-10 rounded-xl flex items-center justify-center"
                        style={{ background: "rgba(16,185,129,0.15)" }}>
                        <CheckCircle size={22} className="text-emerald-400" />
                      </div>
                      <div>
                        <h2 className="text-white font-black text-xl">Your website is live! 🎉</h2>
                        <p className="text-white/40 text-sm">Share the link with your customers</p>
                      </div>
                    </div>

                    {/* Live preview */}
                    <div className="rounded-2xl overflow-hidden"
                      style={{ border: "1px solid rgba(255,255,255,0.1)", background: "#000" }}>
                      <div className="flex items-center gap-2 px-3 py-2"
                        style={{ background: "rgba(255,255,255,0.05)", borderBottom: "1px solid rgba(255,255,255,0.08)" }}>
                        <div className="flex gap-1.5">
                          {["#ef4444", "#f59e0b", "#10b981"].map(c => (
                            <div key={c} className="w-2.5 h-2.5 rounded-full" style={{ background: c }} />
                          ))}
                        </div>
                        <div className="flex-1 mx-2 rounded-md text-[10px] text-white/30 px-2 py-1 text-center truncate"
                          style={{ background: "rgba(255,255,255,0.05)" }}>
                          {backendSiteURL}
                        </div>
                      </div>
                      <iframe
                        src={`${backendSiteURL}`}
                        className="w-full"
                        style={{ height: "340px", border: "none" }}
                        title="Website preview"
                      />
                    </div>

                    {/* Action buttons */}
                    <div className="grid grid-cols-2 gap-2.5">
                      <button onClick={copyLink}
                        className="flex items-center justify-center gap-2 py-3 rounded-2xl font-bold text-sm transition-all active:scale-95"
                        style={{ background: copied ? "rgba(16,185,129,0.15)" : "rgba(255,255,255,0.08)", color: copied ? "#10b981" : "#fff", border: `1px solid ${copied ? "rgba(16,185,129,0.3)" : "rgba(255,255,255,0.1)"}` }}>
                        {copied ? <><Check size={16} /> Copied!</> : <><Copy size={16} /> Copy Link</>}
                      </button>
                      <a href={backendSiteURL || "#"} target="_blank" rel="noreferrer"
                        className="flex items-center justify-center gap-2 py-3 rounded-2xl font-bold text-sm transition-all active:scale-95"
                        style={{ background: "rgba(99,102,241,0.15)", color: "#818cf8", border: "1px solid rgba(99,102,241,0.25)" }}>
                        <ExternalLink size={16} /> Open Site
                      </a>
                    </div>
                    <a href={`https://wa.me/?text=${encodeURIComponent(`Check out my website: ${backendSiteURL}`)}`}
                      target="_blank" rel="noreferrer"
                      className="flex items-center justify-center gap-2 w-full py-3.5 rounded-2xl font-bold text-sm transition-all active:scale-95"
                      style={{ background: "rgba(37,211,102,0.15)", color: "#25D366", border: "1px solid rgba(37,211,102,0.25)" }}>
                      <span className="text-base">💬</span> Share on WhatsApp
                    </a>
                    <button onClick={() => { setStep(1); setSelectedType(null); setFields({}); setPhotos([]); setGenId(null); setSiteStatus("pending"); setIsGenerating(false); setError(null); }}
                      className="flex items-center justify-center gap-2 w-full py-3 rounded-2xl font-bold text-sm text-white/40 hover:text-white/70 transition-colors">
                      <RotateCcw size={14} /> Build another website
                    </button>
                  </>
                )}
              </motion.div>
            )}
          </AnimatePresence>
        </div>

        {/* ── Footer navigation ──────────────────────────────────────────── */}
        {step < 5 && (
          <div className="flex items-center justify-between px-5 py-4 flex-shrink-0"
            style={{ borderTop: "1px solid rgba(255,255,255,0.06)" }}>
            {step > 1 ? (
              <button onClick={() => setStep(s => (s - 1) as typeof step)}
                className="flex items-center gap-2 text-white/50 hover:text-white transition-colors text-sm font-bold">
                <ArrowLeft size={16} /> Back
              </button>
            ) : <div />}

            {step === 1 && (
              <p className="text-white/30 text-xs">Select a type to continue</p>
            )}

            {step === 2 && (
              <button onClick={() => setStep(3)} disabled={!requiredFilled()}
                className="flex items-center gap-2 px-5 py-2.5 rounded-xl font-bold text-sm transition-all disabled:opacity-40 disabled:cursor-not-allowed active:scale-95"
                style={{ background: "linear-gradient(135deg, #6366f1, #8b5cf6)", color: "#fff" }}>
                Next <ArrowRight size={16} />
              </button>
            )}

            {step === 3 && (
              <div className="flex items-center gap-2">
                <button onClick={() => setStep(4)}
                  className="text-white/40 text-sm font-bold hover:text-white/70 transition-colors px-3">
                  Skip →
                </button>
                <button onClick={() => setStep(4)}
                  className="flex items-center gap-2 px-5 py-2.5 rounded-xl font-bold text-sm transition-all active:scale-95"
                  style={{ background: "linear-gradient(135deg, #6366f1, #8b5cf6)", color: "#fff" }}>
                  {photos.length > 0 ? `Continue (${photos.length} photo${photos.length > 1 ? "s" : ""})` : "Continue"} <ArrowRight size={16} />
                </button>
              </div>
            )}

            {step === 4 && (
              <button onClick={handleGenerate}
                disabled={!canAfford || isGenerating || !requiredFilled()}
                className="flex items-center gap-2 px-5 py-2.5 rounded-xl font-bold text-sm transition-all disabled:opacity-40 disabled:cursor-not-allowed active:scale-95"
                style={{ background: "linear-gradient(135deg, #F5A623, #f97316)", color: "#000" }}>
                {isGenerating ? <><Loader2 size={16} className="animate-spin" /> Building...</> : <><Sparkles size={16} /> Generate Website</>}
              </button>
            )}
          </div>
        )}
      </motion.div>
    </div>
  );
}

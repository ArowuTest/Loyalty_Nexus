import type { AITool, SpinPrize, Tier } from "@/lib";

// ─── AI Tools ──────────────────────────────────────────────────
export const AI_TOOLS: AITool[] = [
  // FREE
  { slug:"ask-nexus",      name:"Ask Nexus",       emoji:"🤖", category:"chat",   point_cost:0,  is_free:true,  is_popular:true,  ui_template:"chat",        description:"Chat with Nexus AI — ask anything, get instant smart answers. No points needed." },
  { slug:"web-search",     name:"Web Search AI",   emoji:"🌐", category:"chat",   point_cost:0,  is_free:true,  is_popular:true,  ui_template:"chat",        description:"Real-time web search powered by AI. Get summarised answers with sources. Free forever." },
  { slug:"code-helper",    name:"Code Helper",     emoji:"💻", category:"chat",   point_cost:0,  is_free:true,  is_new:true,      ui_template:"code",        description:"Write, debug and explain code in any language. Persistent session with syntax highlighting." },
  // CREATE
  { slug:"ai-photo",       name:"AI Photo",        emoji:"📸", category:"create", point_cost:10, is_free:false, is_popular:true,  ui_template:"image-gen",   description:"Transform text prompts into breathtaking photorealistic images using Flux & DALL-E." },
  { slug:"ai-photo-dream", name:"AI Photo Dream",  emoji:"🎨", category:"create", point_cost:12, is_free:false, is_new:true,      ui_template:"image-dream", description:"Advanced dream-style AI image generation with ultra-high fidelity and artistic styles." },
  { slug:"bg-remover",     name:"BG Remover",      emoji:"✂️", category:"create", point_cost:3,  is_free:false, is_popular:true,  ui_template:"image-edit",  description:"Remove image backgrounds in one click — perfect results every time. Batch processing available." },
  { slug:"video-cinematic",name:"Video Cinematic", emoji:"🎬", category:"create", point_cost:65, is_free:false, is_new:true,      ui_template:"video-gen",   description:"Generate stunning cinematic videos from text prompts using Kling AI and Wan." },
  { slug:"avatar-gen",     name:"AI Avatar",       emoji:"🧑‍🎨",category:"create", point_cost:15, is_free:false,                   ui_template:"image-gen",   description:"Create professional AI avatars in any style — business, anime, fantasy and more." },
  // LEARN
  { slug:"study-guide",    name:"Study Guide AI",  emoji:"📚", category:"learn",  point_cost:5,  is_free:false, is_popular:true,  ui_template:"doc-gen",     description:"Generate comprehensive study guides, summaries and revision notes for any topic." },
  { slug:"quiz-maker",     name:"Quiz Maker",      emoji:"🎯", category:"learn",  point_cost:3,  is_free:false,                   ui_template:"quiz",        description:"Create interactive quizzes from any text, PDF or topic for effective learning." },
  { slug:"mind-map",       name:"Mind Map AI",     emoji:"🧠", category:"learn",  point_cost:5,  is_free:false, is_new:true,      ui_template:"mind-map",    description:"Automatically generate visual mind maps from any topic or document." },
  { slug:"ai-podcast",     name:"AI Podcast",      emoji:"🎙️", category:"learn",  point_cost:50, is_free:false, is_new:true,      ui_template:"podcast",     description:"Transform any content into a professional multi-host podcast with natural conversation." },
  // BUILD
  { slug:"slide-deck",     name:"Slide Deck AI",   emoji:"📊", category:"build",  point_cost:20, is_free:false, is_popular:true,  ui_template:"pptx",        description:"Generate professional slide presentations from a brief or document in seconds." },
  { slug:"bizplan",        name:"Business Plan AI",emoji:"💼", category:"build",  point_cost:30, is_free:false, is_popular:true,  ui_template:"bizplan",     description:"Create comprehensive business plans with financial projections, market analysis and more." },
  { slug:"voice-to-plan",  name:"Voice to Plan",   emoji:"🎤", category:"build",  point_cost:35, is_free:false, is_new:true,      ui_template:"voice-plan",  description:"Record your idea as voice, and Nexus AI converts it into a polished business plan." },
  { slug:"narrate",        name:"AI Narrate",       emoji:"🔊", category:"build",  point_cost:2,  is_free:false,                   ui_template:"tts",         description:"Convert any text to natural-sounding speech using ElevenLabs premium voices." },
  { slug:"jingle",         name:"Marketing Jingle",emoji:"🎵", category:"build",  point_cost:45, is_free:false, is_new:true,      ui_template:"music-gen",   description:"Generate catchy marketing jingles and background music for your brand or campaign." },
];

// ─── Testimonials ──────────────────────────────────────────────
export const TESTIMONIALS = [
  {
    name:    "Chioma Adeyemi",
    avatar:  "C",
    location:"Lagos Island",
    tier:    "gold",
    quote:   "I've been recharging MTN for years — never imagined I could earn from it. Won ₦3,500 last month and created 20 AI photos for my Instagram page. This is mad!",
  },
  {
    name:    "Tunde Okafor",
    avatar:  "T",
    location:"Abuja, FCT",
    tier:    "platinum",
    quote:   "The AI Studio is genuinely incredible. I used the Business Plan tool to write my proposal — saved me ₦150,000 in consulting fees. Then I spun the wheel and won 3GB data! 🔥",
  },
  {
    name:    "Amina Kalani",
    avatar:  "A",
    location:"Kano State",
    tier:    "platinum",
    quote:   "At first I thought it was too good to be true. But I literally earned ₦5,000 cash from a single spin after reaching Gold tier. Now I recharge daily just for the points!",
  },
  {
    name:    "Fatima Bello",
    avatar:  "F",
    location:"Port Harcourt",
    tier:    "gold",
    quote:   "The Voice to Plan feature is everything. I recorded a 2-minute voice note about my catering idea and it generated a full business plan. I've already shown 5 investors.",
  },
  {
    name:    "Biodun Salami",
    avatar:  "B",
    location:"Ibadan, Oyo State",
    tier:    "silver",
    quote:   "I referred 12 friends this month. That's 6,000 bonus points just from referrals, plus my normal daily earnings. The compound growth is unbelievable.",
  },
  {
    name:    "Emeka Nwachukwu",
    avatar:  "E",
    location:"Enugu State",
    tier:    "silver",
    quote:   "Free AI chat is honestly better than ChatGPT for Nigerian context. Ask Nexus understands pidgin too! That alone keeps me coming back every single day.",
  },
];

// ─── Spin prizes ──────────────────────────────────────────────
export const SPIN_PRIZES: SpinPrize[] = [
  { id:"p1", label:"₦5,000",   color:"#F5A623", value:"5000 NGN",   type:"cash",   weight:1 },
  { id:"p2", label:"₦500",     color:"#FFE066", value:"500 NGN",    type:"cash",   weight:8 },
  { id:"p3", label:"5GB Data", color:"#00D4FF", value:"5GB",        type:"data",   weight:3 },
  { id:"p4", label:"₦1,000",   color:"#10B981", value:"1000 NGN",   type:"cash",   weight:5 },
  { id:"p5", label:"2× Spin",  color:"#8B5CF6", value:"extra_spin", type:"spin",   weight:6 },
  { id:"p6", label:"500 pts",  color:"#F472B6", value:"500",        type:"points", weight:10 },
  { id:"p7", label:"₦100 Air", color:"#FB923C", value:"100 NGN",    type:"airtime",weight:12 },
  { id:"p8", label:"1,000 pts",color:"#60A5FA", value:"1000",       type:"points", weight:7 },
  { id:"p9", label:"₦250",     color:"#34D399", value:"250 NGN",    type:"cash",   weight:9 },
  { id:"p10",label:"2GB Data", color:"#A78BFA", value:"2GB",        type:"data",   weight:4 },
];

// ─── Admin stats ──────────────────────────────────────────────
export const ADMIN_STATS = {
  total_users:        84231,
  active_today:       12943,
  total_generations:  1247903,
  revenue_month:      4850000,
  points_issued:      92_000_000,
  avg_points_per_user:1098,
  top_tools:          ["ask-nexus","ai-photo","bg-remover","bizplan","study-guide"],
  provider_health: {
    groq:       { latency_ms: 142,  calls_today: 8420 },
    gemini:     { latency_ms: 198,  calls_today: 3150 },
    pollinations:{ latency_ms: 2100, calls_today: 1840 },
    elevenlabs: { latency_ms: 890,  calls_today: 420  },
    assemblyai: { latency_ms: 1200, calls_today: 95   },
  },
};

// ─── Mock logged-in user ───────────────────────────────────────
export const MOCK_USER = {
  display_name: "Chioma A.",
  msisdn:       "+234 801 234 5678",
  tier:         "gold" as Tier,
  pulse_points: 3850,
  spins_today:  1,
  spins_used:   0,
  total_earned: 12400,
  referral_code:"NEXUS-CHIOMA",
};

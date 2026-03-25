"use client";

import { useState, useRef, useEffect } from "react";
import { motion, AnimatePresence } from "framer-motion";
import useSWR from "swr";
import AppShell from "@/components/layout/AppShell";
import api from "@/lib/api";
import toast, { Toaster } from "react-hot-toast";
import { Send, Bot, User, Loader2, Wand2, Image as ImageIcon, BookOpen, Mic, FileText, Music, Globe, ChevronRight } from "lucide-react";
import { cn } from "@/lib/utils";

const CATEGORY_ICONS: Record<string, React.ReactNode> = {
  "Knowledge & Research": <BookOpen size={14} />,
  "Image & Visual":       <ImageIcon size={14} />,
  "Audio & Voice":        <Mic size={14} />,
  "Document & Business":  <FileText size={14} />,
  "Music & Entertainment":<Music size={14} />,
  "Language & Translation":<Globe size={14} />,
};

const CATEGORY_COLORS: Record<string, string> = {
  "Knowledge & Research": "bg-blue-500/20 text-blue-400",
  "Image & Visual":       "bg-pink-500/20 text-pink-400",
  "Audio & Voice":        "bg-green-500/20 text-green-400",
  "Document & Business":  "bg-orange-500/20 text-orange-400",
  "Music & Entertainment":"bg-purple-500/20 text-purple-400",
  "Language & Translation":"bg-cyan-500/20 text-cyan-400",
};

interface Tool {
  id: string;
  name: string;
  description: string;
  category: string;
  point_cost: number;
  is_chat: boolean;
}

interface Message {
  role: "user" | "assistant";
  content: string;
}

const fetcher = () => api.getStudioTools() as Promise<{ tools: Tool[] }>;

export default function StudioPage() {
  const { data } = useSWR("/studio/tools", fetcher);
  const tools = data?.tools || [];

  const [activeTab, setActiveTab] = useState<"tools" | "chat">("chat");
  const [messages, setMessages] = useState<Message[]>([
    { role: "assistant", content: "Hi! I'm Nexus AI. Ask me anything — from business advice to everyday questions. What's on your mind? 🧠" }
  ]);
  const [input, setInput] = useState("");
  const [sending, setSending] = useState(false);
  const [selectedTool, setSelectedTool] = useState<Tool | null>(null);
  const [toolPrompt, setToolPrompt] = useState("");
  const [generating, setGenerating] = useState(false);
  const [generationId, setGenerationId] = useState<string | null>(null);
  const messagesEndRef = useRef<HTMLDivElement>(null);

  const categories = [...new Set(tools.map(t => t.category))];

  useEffect(() => {
    messagesEndRef.current?.scrollIntoView({ behavior: "smooth" });
  }, [messages]);

  const handleChat = async () => {
    if (!input.trim() || sending) return;
    const msg = input.trim();
    setInput("");
    setMessages(m => [...m, { role: "user", content: msg }]);
    setSending(true);
    try {
      const resp = await api.sendChat(msg) as { response: string };
      setMessages(m => [...m, { role: "assistant", content: resp.response }]);
    } catch {
      setMessages(m => [...m, { role: "assistant", content: "Sorry, I'm having trouble right now. Please try again in a moment." }]);
    } finally {
      setSending(false);
    }
  };

  const handleGenerate = async () => {
    if (!selectedTool || !toolPrompt.trim()) return;
    setGenerating(true);
    try {
      const result = await api.generateTool(selectedTool.id, toolPrompt);
      setGenerationId(result.generation_id);
      toast.success("Generation started! You'll be notified when ready.");
      setSelectedTool(null);
      setToolPrompt("");
    } catch (e: unknown) {
      toast.error(e instanceof Error ? e.message : "Generation failed");
    } finally {
      setGenerating(false);
    }
  };

  return (
    <AppShell>
      <Toaster position="top-center" toastOptions={{ style: { background: "#1c2038", color: "#fff" } }} />

      <div className="max-w-2xl mx-auto px-4 py-6 space-y-4">
        {/* Header */}
        <div className="flex items-center gap-3">
          <Wand2 className="text-purple-400" size={24} />
          <div>
            <h1 className="text-2xl font-bold font-display text-white">Nexus AI Studio</h1>
            <p className="text-[rgb(130_140_180)] text-sm">17 free AI tools — powered by NotebookLM, Groq & more</p>
          </div>
        </div>

        {/* Tab switcher */}
        <div className="flex nexus-card p-1 gap-1">
          {(["chat", "tools"] as const).map(tab => (
            <button
              key={tab}
              onClick={() => setActiveTab(tab)}
              className={cn(
                "flex-1 py-2 px-4 rounded-xl text-sm font-medium transition-all capitalize",
                activeTab === tab
                  ? "bg-nexus-600 text-white shadow"
                  : "text-[rgb(130_140_180)] hover:text-white"
              )}
            >
              {tab === "chat" ? "💬 Chat" : "🛠 Tools"}
            </button>
          ))}
        </div>

        {/* Chat tab */}
        <AnimatePresence mode="wait">
          {activeTab === "chat" && (
            <motion.div key="chat" initial={{ opacity: 0 }} animate={{ opacity: 1 }}>
              {/* Message list */}
              <div className="nexus-card h-96 overflow-y-auto p-4 space-y-4">
                {messages.map((msg, i) => (
                  <div key={i} className={cn("flex gap-3", msg.role === "user" && "flex-row-reverse")}>
                    <div className={cn(
                      "w-7 h-7 rounded-full flex items-center justify-center flex-shrink-0",
                      msg.role === "assistant" ? "bg-nexus-600/30" : "bg-purple-600/30"
                    )}>
                      {msg.role === "assistant" ? <Bot size={14} className="text-nexus-400" /> : <User size={14} className="text-purple-400" />}
                    </div>
                    <div className={cn(
                      "max-w-xs rounded-2xl px-4 py-2.5 text-sm",
                      msg.role === "assistant"
                        ? "bg-[rgb(28_32_58)] text-white rounded-tl-none"
                        : "bg-nexus-600 text-white rounded-tr-none"
                    )}>
                      {msg.content}
                    </div>
                  </div>
                ))}
                {sending && (
                  <div className="flex gap-3">
                    <div className="w-7 h-7 rounded-full bg-nexus-600/30 flex items-center justify-center">
                      <Bot size={14} className="text-nexus-400" />
                    </div>
                    <div className="nexus-card px-4 py-2.5 rounded-tl-none">
                      <Loader2 size={14} className="animate-spin text-nexus-400" />
                    </div>
                  </div>
                )}
                <div ref={messagesEndRef} />
              </div>
              {/* Input */}
              <div className="flex gap-2 mt-2">
                <input
                  value={input}
                  onChange={(e) => setInput(e.target.value)}
                  onKeyDown={(e) => e.key === "Enter" && !e.shiftKey && handleChat()}
                  placeholder="Ask Nexus anything…"
                  className="nexus-input flex-1"
                />
                <button
                  onClick={handleChat}
                  disabled={sending || !input.trim()}
                  className="nexus-btn-primary px-4 py-3"
                >
                  <Send size={16} />
                </button>
              </div>
            </motion.div>
          )}

          {/* Tools tab */}
          {activeTab === "tools" && (
            <motion.div key="tools" initial={{ opacity: 0 }} animate={{ opacity: 1 }}>
              {categories.map(cat => (
                <div key={cat} className="mb-4">
                  <div className={cn("flex items-center gap-2 mb-2 px-1", CATEGORY_COLORS[cat] || "text-white")}>
                    {CATEGORY_ICONS[cat] || <Wand2 size={14} />}
                    <h3 className="text-sm font-semibold uppercase tracking-wider">{cat}</h3>
                  </div>
                  <div className="grid grid-cols-1 gap-2">
                    {tools.filter(t => t.category === cat).map(tool => (
                      <button
                        key={tool.id}
                        onClick={() => setSelectedTool(tool)}
                        className="nexus-card p-3 flex items-center gap-3 text-left hover:border-nexus-500/30 transition-all"
                      >
                        <div className={cn("p-2 rounded-xl flex-shrink-0", CATEGORY_COLORS[cat] || "bg-white/10")}>
                          {CATEGORY_ICONS[cat] || <Wand2 size={14} />}
                        </div>
                        <div className="flex-1 min-w-0">
                          <p className="text-white font-medium text-sm">{tool.name}</p>
                          <p className="text-[rgb(130_140_180)] text-xs truncate">{tool.description}</p>
                        </div>
                        <div className="flex flex-col items-end gap-1">
                          <span className="text-xs text-nexus-400 font-semibold">{tool.point_cost === 0 ? "Free" : `${tool.point_cost} pts`}</span>
                          <ChevronRight size={14} className="text-[rgb(130_140_180)]" />
                        </div>
                      </button>
                    ))}
                  </div>
                </div>
              ))}
            </motion.div>
          )}
        </AnimatePresence>
      </div>

      {/* Tool prompt modal */}
      <AnimatePresence>
        {selectedTool && (
          <motion.div
            initial={{ opacity: 0 }}
            animate={{ opacity: 1 }}
            exit={{ opacity: 0 }}
            className="fixed inset-0 bg-black/60 backdrop-blur-sm z-50 flex items-end md:items-center justify-center p-4"
            onClick={() => setSelectedTool(null)}
          >
            <motion.div
              initial={{ y: 50 }}
              animate={{ y: 0 }}
              exit={{ y: 50 }}
              className="nexus-card w-full max-w-md p-6"
              onClick={(e) => e.stopPropagation()}
            >
              <h3 className="text-lg font-bold text-white mb-1">{selectedTool.name}</h3>
              <p className="text-[rgb(130_140_180)] text-sm mb-4">{selectedTool.description}</p>
              <textarea
                placeholder="Describe what you want to generate…"
                value={toolPrompt}
                onChange={(e) => setToolPrompt(e.target.value)}
                rows={4}
                className="nexus-input resize-none mb-4"
              />
              <div className="flex gap-2">
                <button onClick={() => setSelectedTool(null)} className="nexus-btn-outline flex-1">
                  Cancel
                </button>
                <button
                  onClick={handleGenerate}
                  disabled={generating || !toolPrompt.trim()}
                  className="nexus-btn-primary flex-1 flex items-center justify-center gap-2"
                >
                  {generating ? <Loader2 size={16} className="animate-spin" /> : <Wand2 size={16} />}
                  {generating ? "Starting…" : `Generate ${selectedTool.point_cost > 0 ? `(${selectedTool.point_cost} pts)` : "(Free)"}`}
                </button>
              </div>
            </motion.div>
          </motion.div>
        )}
      </AnimatePresence>
    </AppShell>
  );
}

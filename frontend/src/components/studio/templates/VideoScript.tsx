'use client';

/**
 * VideoScript — Script-Driven Animation Builder
 *
 * Inspired by Pika Labs Story Mode, Kaiber, and Kling Story Creator.
 *
 * Workflow:
 *  1. Define characters (name + appearance description + optional reference image)
 *  2. Write a scene-by-scene script (each scene: background image + dialogue lines + direction)
 *  3. Choose visual style (anime, realistic, cartoon, cinematic, etc.)
 *  4. Set duration and aspect ratio
 *  5. Generate — compiles script into a structured prompt + image list for Grok/Kling
 *
 * API contract:
 *  - prompt: compiled story synopsis + scene directions
 *  - extra_params.image_urls: array of scene background image CDN URLs
 *  - extra_params.scene_N_caption: per-scene compiled dialogue + direction
 *  - duration, aspect_ratio: standard fields
 */

import { useState, useRef } from 'react';
import {
  Upload, X, Sparkles, Film, Users, Plus, Trash2,
  ChevronDown, ChevronUp, Mic, MicOff, ImageIcon,
  MessageSquare, Clapperboard, Palette, Clock,
} from 'lucide-react';
import { useSpeechToText } from '@/hooks/useSpeechToText';
import { TemplateProps, GeneratePayload } from './types';
import { cn } from '@/lib/utils';
import api from '@/lib/api';

// ── Types ──────────────────────────────────────────────────────────────────────

interface Character {
  id:          string;
  name:        string;
  appearance:  string; // "A tall woman in a red dress with curly hair"
  voiceNote:   string; // optional description of how they speak
}

interface DialogueLine {
  characterId: string; // '' = narrator / direction
  text:        string;
}

interface SceneSlot {
  id:          string;
  imageFile:   File | null;
  imagePreview: string | null;
  imageUrl:    string;        // remote URL (pasted or CDN)
  uploadedUrl: string;        // CDN URL after upload
  direction:   string;        // scene direction / setting description
  dialogue:    DialogueLine[];
}

// ── Constants ──────────────────────────────────────────────────────────────────

const VISUAL_STYLES = [
  { value: 'cinematic',  label: 'Cinematic',  desc: 'Realistic, film-quality',   emoji: '🎬' },
  { value: 'anime',      label: 'Anime',      desc: 'Japanese animation style',  emoji: '🌸' },
  { value: 'cartoon',    label: 'Cartoon',    desc: 'Colourful, expressive',      emoji: '🎨' },
  { value: 'realistic',  label: 'Realistic',  desc: 'Photorealistic rendering',  emoji: '📷' },
  { value: '3d',         label: '3D Render',  desc: 'CGI / Pixar-style',         emoji: '🎭' },
  { value: 'storybook',  label: 'Storybook',  desc: 'Illustrated, painterly',    emoji: '📖' },
];

const ASPECT_RATIOS = [
  { value: '16:9', label: 'Landscape', icon: '🖥️' },
  { value: '9:16', label: 'Portrait',  icon: '📱' },
  { value: '1:1',  label: 'Square',    icon: '⬜' },
];

const DURATION_OPTIONS = [5, 8, 10, 15];

const EXAMPLE_CHARACTERS: Character[] = [
  { id: 'c1', name: 'Amara', appearance: 'A confident young Nigerian woman in a bright yellow dress, natural hair', voiceNote: 'Speaks warmly and with authority' },
  { id: 'c2', name: 'Emeka', appearance: 'A tall man in a traditional agbada, mid-30s, friendly smile', voiceNote: 'Calm and thoughtful' },
];

const NARRATOR_ID = '__narrator__';

function emptyScene(id: string): SceneSlot {
  return {
    id,
    imageFile:    null,
    imagePreview: null,
    imageUrl:     '',
    uploadedUrl:  '',
    direction:    '',
    dialogue:     [{ characterId: NARRATOR_ID, text: '' }],
  };
}

function emptyCharacter(): Character {
  return { id: String(Date.now()), name: '', appearance: '', voiceNote: '' };
}

// ── Component ──────────────────────────────────────────────────────────────────

export default function VideoScript({ tool, onSubmit, isLoading, userPoints }: TemplateProps) {
  const cfg = tool.ui_config ?? {};
  const maxScenes  = cfg.max_scenes  ?? 6;
  const maxChars   = cfg.max_characters ?? 5;

  // ── Story-level state ──────────────────────────────────────────────────────
  const [synopsis,     setSynopsis]     = useState('');
  const [visualStyle,  setVisualStyle]  = useState('cinematic');
  const [aspectRatio,  setAspectRatio]  = useState(cfg.default_aspect ?? '16:9');
  const [duration,     setDuration]     = useState<number>(cfg.default_duration_script ?? cfg.default_duration_video ?? 10);

  // ── Characters ─────────────────────────────────────────────────────────────
  const [characters,   setCharacters]   = useState<Character[]>([]);
  const [showChars,    setShowChars]    = useState(true);
  const [useExamples,  setUseExamples]  = useState(false);

  // ── Scenes ─────────────────────────────────────────────────────────────────
  const [scenes,       setScenes]       = useState<SceneSlot[]>([emptyScene('s1'), emptyScene('s2')]);
  const [expandedScene, setExpandedScene] = useState<string | null>('s1');
  const [uploading,    setUploading]    = useState(false);

  // ── Voice input for synopsis ───────────────────────────────────────────────
  const { speechState, speechError, interimText, handleMicClick } =
    useSpeechToText({
      onTranscript: (t) => setSynopsis(prev => prev ? prev + ' ' + t : t),
      language: 'en-US',
    });

  const fileRefs = useRef<Record<string, HTMLInputElement | null>>({});

  const canAfford   = tool.is_free || userPoints >= tool.point_cost;
  const filledScenes = scenes.filter(s => s.uploadedUrl || s.imageUrl.trim() || s.imageFile);
  const isValid     = filledScenes.length >= 1 && !uploading;

  // ── Character helpers ──────────────────────────────────────────────────────
  function addCharacter() {
    if (characters.length >= maxChars) return;
    setCharacters(prev => [...prev, emptyCharacter()]);
  }

  function updateCharacter(id: string, patch: Partial<Character>) {
    setCharacters(prev => prev.map(c => c.id === id ? { ...c, ...patch } : c));
  }

  function removeCharacter(id: string) {
    setCharacters(prev => prev.filter(c => c.id !== id));
    // Replace removed character's lines with narrator
    setScenes(prev => prev.map(scene => ({
      ...scene,
      dialogue: scene.dialogue.map(line =>
        line.characterId === id ? { ...line, characterId: NARRATOR_ID } : line,
      ),
    })));
  }

  function loadExamples() {
    setCharacters(EXAMPLE_CHARACTERS);
    setUseExamples(true);
  }

  // ── Scene helpers ──────────────────────────────────────────────────────────
  function addScene() {
    if (scenes.length >= maxScenes) return;
    const id = `s${Date.now()}`;
    setScenes(prev => [...prev, emptyScene(id)]);
    setExpandedScene(id);
  }

  function removeScene(id: string) {
    if (scenes.length <= 2) return;
    setScenes(prev => prev.filter(s => s.id !== id));
  }

  function updateScene(id: string, patch: Partial<SceneSlot>) {
    setScenes(prev => prev.map(s => s.id === id ? { ...s, ...patch } : s));
  }

  // ── Dialogue helpers ───────────────────────────────────────────────────────
  function addDialogueLine(sceneId: string) {
    setScenes(prev => prev.map(s =>
      s.id === sceneId
        ? { ...s, dialogue: [...s.dialogue, { characterId: NARRATOR_ID, text: '' }] }
        : s,
    ));
  }

  function updateDialogueLine(sceneId: string, lineIdx: number, patch: Partial<DialogueLine>) {
    setScenes(prev => prev.map(s => {
      if (s.id !== sceneId) return s;
      const newDialogue = s.dialogue.map((l, i) => i === lineIdx ? { ...l, ...patch } : l);
      return { ...s, dialogue: newDialogue };
    }));
  }

  function removeDialogueLine(sceneId: string, lineIdx: number) {
    setScenes(prev => prev.map(s => {
      if (s.id !== sceneId) return s;
      if (s.dialogue.length <= 1) return s;
      return { ...s, dialogue: s.dialogue.filter((_, i) => i !== lineIdx) };
    }));
  }

  // ── Image upload ───────────────────────────────────────────────────────────
  async function handleImageFile(sceneId: string, file: File) {
    const reader = new FileReader();
    reader.onload = (e) => updateScene(sceneId, { imageFile: file, imagePreview: e.target?.result as string, imageUrl: '' });
    reader.readAsDataURL(file);
  }

  function handleDrop(sceneId: string, e: React.DragEvent) {
    e.preventDefault();
    const file = e.dataTransfer.files[0];
    if (file && file.type.startsWith('image/')) handleImageFile(sceneId, file);
  }

  // ── Build prompt from script ───────────────────────────────────────────────
  function compileSceneCaption(scene: SceneSlot): string {
    const parts: string[] = [];
    if (scene.direction.trim()) parts.push(`[Setting: ${scene.direction.trim()}]`);
    for (const line of scene.dialogue) {
      if (!line.text.trim()) continue;
      if (line.characterId === NARRATOR_ID) {
        parts.push(line.text.trim());
      } else {
        const char = characters.find(c => c.id === line.characterId);
        const name = char?.name || 'Character';
        parts.push(`${name}: "${line.text.trim()}"`);
      }
    }
    return parts.join(' ');
  }

  function compileFullPrompt(): string {
    const styleDesc = VISUAL_STYLES.find(s => s.value === visualStyle)?.label ?? visualStyle;
    const charDescs = characters.map(c => `${c.name} (${c.appearance})`).join(', ');
    const parts: string[] = [];
    if (synopsis.trim()) parts.push(synopsis.trim());
    if (charDescs) parts.push(`Characters: ${charDescs}`);
    parts.push(`Visual style: ${styleDesc} animation`);
    return parts.join('. ');
  }

  // ── Submit ─────────────────────────────────────────────────────────────────
  async function handleSubmit() {
    if (!isValid || isLoading || !canAfford) return;

    setUploading(true);
    const imageUrls: string[] = [];
    const extraParams: Record<string, string> = {};

    try {
      let sceneIdx = 0;
      for (const scene of scenes) {
        const hasMaterial = scene.imageFile || scene.imageUrl.trim() || scene.uploadedUrl;
        if (!hasMaterial) continue;

        let url = scene.uploadedUrl || scene.imageUrl.trim();
        if (scene.imageFile && !scene.uploadedUrl) {
          const result = await api.uploadAsset(scene.imageFile);
          url = result.url;
          updateScene(scene.id, { uploadedUrl: url });
        }
        imageUrls.push(url);
        const caption = compileSceneCaption(scene);
        if (caption) {
          extraParams[`scene_${sceneIdx + 1}_caption`] = caption;
        }
        sceneIdx++;
      }
    } catch (err) {
      console.error('[VideoScript] upload error:', err);
      setUploading(false);
      return;
    }
    setUploading(false);

    const payload: GeneratePayload = {
      prompt:       compileFullPrompt(),
      duration,
      aspect_ratio: aspectRatio,
      extra_params: {
        image_urls:      imageUrls,
        visual_style:    visualStyle,
        generate_audio:  true,   // Kling v2.6 native audio — ambient sound + dialogue hints
        ...extraParams,
      },
    };
    onSubmit(payload);
  }

  // ── Helpers ────────────────────────────────────────────────────────────────
  const allCharacters = [
    { id: NARRATOR_ID, name: 'Narrator / Direction' },
    ...characters,
  ];

  // ── Render ─────────────────────────────────────────────────────────────────
  return (
    <div className="space-y-5">

      {/* ── How it works banner ── */}
      <div className="flex items-start gap-3 bg-violet-500/8 border border-violet-500/20 rounded-xl px-4 py-3">
        <Clapperboard size={16} className="text-violet-400 flex-shrink-0 mt-0.5" />
        <div>
          <p className="text-white/80 text-sm font-medium">Script-Driven Animation</p>
          <p className="text-white/40 text-xs mt-0.5 leading-relaxed">
            Define your characters, write a scene-by-scene script with dialogue, upload background images, and the AI animates your story into a video.
          </p>
        </div>
      </div>

      {/* ── Story Synopsis ── */}
      <div>
        <label className="text-white/35 text-[10px] uppercase tracking-wider font-semibold mb-1.5 block">
          Story Synopsis <span className="normal-case font-normal text-white/20">(optional — sets the overall tone)</span>
        </label>
        <div className="relative">
          <textarea
            value={synopsis}
            onChange={(e) => setSynopsis(e.target.value)}
            placeholder="A heartwarming story about two childhood friends who reunite after 10 years in Lagos…"
            rows={2}
            className="nexus-input w-full text-sm resize-none pr-10"
          />
          <button
            type="button"
            onClick={handleMicClick}
            className={cn(
              'absolute right-2.5 top-2.5 p-1 rounded-md transition-colors',
              speechState === 'listening'
                ? 'text-red-400 bg-red-500/15 animate-pulse'
                : 'text-white/30 hover:text-white/60',
            )}
            title={speechState === 'listening' ? 'Stop recording' : 'Dictate synopsis'}
          >
            {speechState === 'listening' ? <MicOff size={14} /> : <Mic size={14} />}
          </button>
        </div>
        {interimText && (
          <p className="text-white/30 text-xs mt-1 italic">{interimText}</p>
        )}
        {speechError && (
          <p className="text-red-400/70 text-xs mt-1">{speechError}</p>
        )}
      </div>

      {/* ── Characters ── */}
      <div className="border border-white/10 rounded-xl overflow-hidden">
        <button
          onClick={() => setShowChars(!showChars)}
          className="w-full flex items-center gap-2.5 px-4 py-3 hover:bg-white/3 transition-colors text-left"
        >
          <Users size={14} className="text-violet-400 flex-shrink-0" />
          <span className="text-white/70 text-sm font-medium flex-1">
            Characters
            {characters.length > 0 && (
              <span className="ml-2 text-violet-400/70 text-xs font-normal">{characters.length} defined</span>
            )}
          </span>
          {showChars ? <ChevronUp size={14} className="text-white/30" /> : <ChevronDown size={14} className="text-white/30" />}
        </button>

        {showChars && (
          <div className="border-t border-white/8 px-4 pb-4 pt-3 space-y-3">
            {characters.length === 0 && (
              <div className="text-center py-3">
                <p className="text-white/30 text-xs mb-3">No characters yet. Add your own or load examples.</p>
                <div className="flex gap-2 justify-center">
                  <button
                    onClick={addCharacter}
                    className="flex items-center gap-1.5 px-3 py-1.5 bg-violet-600/20 hover:bg-violet-600/30 border border-violet-500/25 rounded-lg text-violet-300 text-xs transition-colors"
                  >
                    <Plus size={12} /> Add Character
                  </button>
                  {!useExamples && (
                    <button
                      onClick={loadExamples}
                      className="flex items-center gap-1.5 px-3 py-1.5 bg-white/5 hover:bg-white/8 border border-white/10 rounded-lg text-white/50 text-xs transition-colors"
                    >
                      Load Examples
                    </button>
                  )}
                </div>
              </div>
            )}

            {characters.map((char, idx) => (
              <div key={char.id} className="bg-white/3 border border-white/8 rounded-lg p-3 space-y-2">
                <div className="flex items-center gap-2">
                  <div className="w-6 h-6 rounded-full bg-violet-600/30 flex items-center justify-center text-[10px] font-bold text-violet-300 flex-shrink-0">
                    {idx + 1}
                  </div>
                  <input
                    type="text"
                    value={char.name}
                    onChange={(e) => updateCharacter(char.id, { name: e.target.value })}
                    placeholder="Character name (e.g. Amara)"
                    className="nexus-input flex-1 text-xs"
                  />
                  <button
                    onClick={() => removeCharacter(char.id)}
                    className="p-1 text-white/25 hover:text-red-400 transition-colors flex-shrink-0"
                  >
                    <Trash2 size={13} />
                  </button>
                </div>
                <input
                  type="text"
                  value={char.appearance}
                  onChange={(e) => updateCharacter(char.id, { appearance: e.target.value })}
                  placeholder="Appearance: e.g. A tall woman in a yellow dress with natural hair"
                  className="nexus-input w-full text-xs"
                />
                <input
                  type="text"
                  value={char.voiceNote}
                  onChange={(e) => updateCharacter(char.id, { voiceNote: e.target.value })}
                  placeholder="Voice / personality note (optional): e.g. Speaks warmly and confidently"
                  className="nexus-input w-full text-xs"
                />
              </div>
            ))}

            {characters.length > 0 && characters.length < maxChars && (
              <button
                onClick={addCharacter}
                className="w-full border border-dashed border-white/12 rounded-lg py-2 flex items-center justify-center gap-1.5 text-white/35 text-xs hover:border-violet-500/30 hover:text-violet-400 transition-colors"
              >
                <Plus size={12} /> Add Another Character
              </button>
            )}
          </div>
        )}
      </div>

      {/* ── Scenes ── */}
      <div>
        <div className="flex items-center justify-between mb-2">
          <label className="text-white/35 text-[10px] uppercase tracking-wider font-semibold">
            Scenes <span className="normal-case font-normal text-white/20">({scenes.length}/{maxScenes})</span>
          </label>
          <p className="text-white/25 text-[10px]">Upload a background image for each scene</p>
        </div>

        <div className="space-y-3">
          {scenes.map((scene, sceneIdx) => {
            const hasImage    = scene.imageFile || scene.imageUrl.trim() || scene.uploadedUrl;
            const isExpanded  = expandedScene === scene.id;
            const hasContent  = hasImage || scene.direction.trim() || scene.dialogue.some(l => l.text.trim());

            return (
              <div
                key={scene.id}
                className={cn(
                  'border rounded-xl overflow-hidden transition-all',
                  hasContent
                    ? 'border-violet-500/25 bg-violet-500/4'
                    : 'border-white/10 bg-white/2',
                )}
              >
                {/* Scene header */}
                <div className="flex items-center gap-2 px-3 py-2.5">
                  <div className={cn(
                    'w-5 h-5 rounded-full flex items-center justify-center text-[10px] font-bold flex-shrink-0',
                    hasContent ? 'bg-violet-600/40 text-violet-300' : 'bg-white/8 text-white/40',
                  )}>
                    {sceneIdx + 1}
                  </div>
                  <span className="text-white/50 text-xs font-medium flex-1 truncate">
                    {scene.direction.trim()
                      ? scene.direction.trim().slice(0, 40) + (scene.direction.length > 40 ? '…' : '')
                      : hasImage
                        ? (scene.imageFile?.name ?? `Scene ${sceneIdx + 1}`)
                        : `Scene ${sceneIdx + 1} — add image & script`}
                  </span>
                  <div className="flex items-center gap-1">
                    <button
                      onClick={() => setExpandedScene(isExpanded ? null : scene.id)}
                      className="p-1 text-white/30 hover:text-white/60 transition-colors"
                    >
                      {isExpanded ? <ChevronUp size={14} /> : <ChevronDown size={14} />}
                    </button>
                    {scenes.length > 2 && (
                      <button
                        onClick={() => removeScene(scene.id)}
                        className="p-1 text-white/25 hover:text-red-400 transition-colors"
                      >
                        <X size={14} />
                      </button>
                    )}
                  </div>
                </div>

                {/* Scene body */}
                {isExpanded && (
                  <div className="border-t border-white/6 px-3 pb-3 pt-3 space-y-3">

                    {/* Background image */}
                    <div>
                      <label className="text-white/30 text-[10px] uppercase tracking-wider font-semibold mb-1.5 block flex items-center gap-1">
                        <ImageIcon size={10} /> Background Image
                      </label>
                      {!hasImage ? (
                        <>
                          <div
                            onDrop={(e) => handleDrop(scene.id, e)}
                            onDragOver={(e) => e.preventDefault()}
                            onClick={() => fileRefs.current[scene.id]?.click()}
                            className="border-2 border-dashed border-white/12 rounded-lg p-5 flex flex-col items-center gap-2
                                       cursor-pointer hover:border-violet-500/35 hover:bg-violet-500/4 transition-all text-center"
                          >
                            <Upload size={16} className="text-white/30" />
                            <p className="text-white/40 text-xs">Click or drag image here</p>
                            <p className="text-white/20 text-[10px]">PNG, JPG, WebP</p>
                          </div>
                          <input
                            ref={(el) => { fileRefs.current[scene.id] = el; }}
                            type="file"
                            accept="image/png,image/jpeg,image/webp"
                            className="hidden"
                            onChange={(e) => {
                              const f = e.target.files?.[0];
                              if (f) handleImageFile(scene.id, f);
                            }}
                          />
                          <p className="text-white/20 text-[10px] text-center mt-1.5">— or paste a URL —</p>
                          <input
                            type="url"
                            value={scene.imageUrl}
                            onChange={(e) => updateScene(scene.id, { imageUrl: e.target.value })}
                            placeholder="https://example.com/background.jpg"
                            className="nexus-input w-full text-xs mt-1"
                          />
                        </>
                      ) : (
                        <div className="relative rounded-lg overflow-hidden border border-white/10 bg-black/40">
                          <img
                            src={scene.imagePreview ?? scene.imageUrl ?? scene.uploadedUrl}
                            alt={`Scene ${sceneIdx + 1} background`}
                            className="w-full max-h-36 object-cover"
                          />
                          <button
                            onClick={() => updateScene(scene.id, { imageFile: null, imagePreview: null, imageUrl: '', uploadedUrl: '' })}
                            className="absolute top-1.5 right-1.5 p-1 bg-black/70 rounded-full text-white/50 hover:text-white transition-colors"
                          >
                            <X size={12} />
                          </button>
                        </div>
                      )}
                    </div>

                    {/* Scene direction */}
                    <div>
                      <label className="text-white/30 text-[10px] uppercase tracking-wider font-semibold mb-1.5 block flex items-center gap-1">
                        <Film size={10} /> Scene Direction
                      </label>
                      <input
                        type="text"
                        value={scene.direction}
                        onChange={(e) => updateScene(scene.id, { direction: e.target.value })}
                        placeholder="e.g. A busy Lagos market at sunset. Amara walks through the crowd."
                        className="nexus-input w-full text-xs"
                      />
                    </div>

                    {/* Dialogue */}
                    <div>
                      <label className="text-white/30 text-[10px] uppercase tracking-wider font-semibold mb-1.5 block flex items-center gap-1">
                        <MessageSquare size={10} /> Dialogue & Narration
                      </label>
                      <div className="space-y-2">
                        {scene.dialogue.map((line, lineIdx) => (
                          <div key={lineIdx} className="flex gap-2 items-start">
                            {/* Character selector */}
                            <select
                              value={line.characterId}
                              onChange={(e) => updateDialogueLine(scene.id, lineIdx, { characterId: e.target.value })}
                              className="nexus-input text-xs w-32 flex-shrink-0 py-1.5"
                            >
                              {allCharacters.map(c => (
                                <option key={c.id} value={c.id}>{c.name}</option>
                              ))}
                            </select>
                            {/* Line text */}
                            <input
                              type="text"
                              value={line.text}
                              onChange={(e) => updateDialogueLine(scene.id, lineIdx, { text: e.target.value })}
                              placeholder={line.characterId === NARRATOR_ID
                                ? 'Narration or stage direction…'
                                : `${allCharacters.find(c => c.id === line.characterId)?.name ?? 'Character'} says…`}
                              className="nexus-input flex-1 text-xs"
                            />
                            {scene.dialogue.length > 1 && (
                              <button
                                onClick={() => removeDialogueLine(scene.id, lineIdx)}
                                className="p-1.5 text-white/20 hover:text-red-400 transition-colors flex-shrink-0"
                              >
                                <X size={12} />
                              </button>
                            )}
                          </div>
                        ))}
                        <button
                          onClick={() => addDialogueLine(scene.id)}
                          className="flex items-center gap-1.5 text-white/30 hover:text-violet-400 text-xs transition-colors mt-1"
                        >
                          <Plus size={11} /> Add line
                        </button>
                      </div>
                    </div>
                  </div>
                )}
              </div>
            );
          })}

          {/* Add scene */}
          {scenes.length < maxScenes && (
            <button
              onClick={addScene}
              className="w-full border-2 border-dashed border-white/10 rounded-xl py-3 flex items-center justify-center gap-2
                         text-white/35 text-xs hover:border-violet-500/30 hover:text-violet-400 transition-all"
            >
              <Plus size={14} /> Add Scene
            </button>
          )}
        </div>
      </div>

      {/* ── Visual Style ── */}
      <div>
        <label className="text-white/35 text-[10px] uppercase tracking-wider font-semibold mb-2 block flex items-center gap-1">
          <Palette size={10} /> Visual Style
        </label>
        <div className="grid grid-cols-3 gap-2">
          {VISUAL_STYLES.map(style => (
            <button
              key={style.value}
              onClick={() => setVisualStyle(style.value)}
              className={cn(
                'flex flex-col items-center gap-1 py-2.5 px-2 rounded-xl border text-center transition-all',
                visualStyle === style.value
                  ? 'border-violet-500/50 bg-violet-500/12 text-violet-300'
                  : 'border-white/8 bg-white/2 text-white/40 hover:border-white/15 hover:text-white/60',
              )}
            >
              <span className="text-lg leading-none">{style.emoji}</span>
              <span className="text-[11px] font-medium">{style.label}</span>
              <span className="text-[9px] text-white/25">{style.desc}</span>
            </button>
          ))}
        </div>
      </div>

      {/* ── Aspect Ratio + Duration ── */}
      <div className="grid grid-cols-2 gap-3">
        <div>
          <label className="text-white/35 text-[10px] uppercase tracking-wider font-semibold mb-1.5 block">
            Aspect Ratio
          </label>
          <div className="flex gap-1.5">
            {ASPECT_RATIOS.map(ar => (
              <button
                key={ar.value}
                onClick={() => setAspectRatio(ar.value)}
                className={cn(
                  'flex-1 py-2 rounded-lg border text-xs font-medium transition-all flex flex-col items-center gap-0.5',
                  aspectRatio === ar.value
                    ? 'border-violet-500/50 bg-violet-500/12 text-violet-300'
                    : 'border-white/8 bg-white/2 text-white/35 hover:border-white/15',
                )}
              >
                <span className="text-sm">{ar.icon}</span>
                <span className="text-[9px]">{ar.label}</span>
              </button>
            ))}
          </div>
        </div>
        <div>
          <label className="text-white/35 text-[10px] uppercase tracking-wider font-semibold mb-1.5 block flex items-center gap-1">
            <Clock size={10} /> Duration
          </label>
          <div className="flex gap-1.5">
            {DURATION_OPTIONS.map(d => (
              <button
                key={d}
                onClick={() => setDuration(d)}
                className={cn(
                  'flex-1 py-2 rounded-lg border text-xs font-medium transition-all',
                  duration === d
                    ? 'border-violet-500/50 bg-violet-500/12 text-violet-300'
                    : 'border-white/8 bg-white/2 text-white/35 hover:border-white/15',
                )}
              >
                {d}s
              </button>
            ))}
          </div>
        </div>
      </div>

      {/* ── Validation hint ── */}
      {!isValid && (
        <p className="text-amber-400/60 text-xs text-center">
          Upload a background image to at least 1 scene to generate your video
        </p>
      )}

      {/* ── Generate button ── */}
      <button
        onClick={handleSubmit}
        disabled={!isValid || isLoading || !canAfford || uploading}
        className={cn(
          'w-full py-3.5 rounded-xl font-semibold text-sm flex items-center justify-center gap-2 transition-all',
          isValid && canAfford && !isLoading && !uploading
            ? 'bg-gradient-to-r from-violet-600 to-purple-600 hover:from-violet-500 hover:to-purple-500 text-white shadow-lg shadow-violet-500/20'
            : 'bg-white/5 text-white/25 cursor-not-allowed',
        )}
      >
        {uploading ? (
          <>
            <span className="w-4 h-4 border-2 border-white/30 border-t-white rounded-full animate-spin" />
            Uploading scenes…
          </>
        ) : isLoading ? (
          <>
            <span className="w-4 h-4 border-2 border-white/30 border-t-white rounded-full animate-spin" />
            Animating your story…
          </>
        ) : (
          <>
            <Sparkles size={16} />
            {!canAfford
              ? `Need ${tool.point_cost} PulsePoints`
              : `Animate Story · ${tool.is_free ? 'Free' : `${tool.point_cost} PP`}`}
          </>
        )}
      </button>

      {!canAfford && (
        <p className="text-center text-xs text-white/30">
          You have {userPoints} PulsePoints. This tool costs {tool.point_cost} PP.
        </p>
      )}
    </div>
  );
}

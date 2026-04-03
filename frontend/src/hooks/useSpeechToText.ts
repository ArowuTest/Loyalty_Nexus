'use client';

/**
 * useSpeechToText
 * ───────────────
 * Browser-native voice-to-text using the Web Speech API (SpeechRecognition).
 * Runs 100% client-side — no uploads, no backend calls, no API keys, no rate limits.
 * Scales to any number of concurrent users because each browser talks directly
 * to Google/Apple/Microsoft's speech servers, not yours.
 *
 * Supported browsers: Chrome, Edge, Safari (≥ iOS 14.5), Samsung Internet.
 * Firefox: not supported — a graceful "not supported" error is shown.
 *
 * Language: defaults to English (en-US). The caller can pass any BCP-47 language
 * tag supported by the browser (e.g. 'fr-FR', 'yo', 'ha', 'ig', 'pt-BR', etc.)
 */

import { useState, useRef, useCallback, useEffect } from 'react';

export type SpeechState = 'idle' | 'listening' | 'processing' | 'error';

// ── Web Speech API type declarations ──────────────────────────────────────────
// These are not in the default TypeScript lib; we declare them manually so the
// hook compiles without requiring @types/dom-speech-recognition.

interface ISpeechRecognitionResult {
  readonly isFinal: boolean;
  readonly length: number;
  item(index: number): SpeechRecognitionAlternative;
  [index: number]: SpeechRecognitionAlternative;
}

interface SpeechRecognitionAlternative {
  readonly transcript: string;
  readonly confidence: number;
}

interface ISpeechRecognitionResultList {
  readonly length: number;
  item(index: number): ISpeechRecognitionResult;
  [index: number]: ISpeechRecognitionResult;
}

interface ISpeechRecognitionEvent extends Event {
  readonly resultIndex: number;
  readonly results: ISpeechRecognitionResultList;
}

interface ISpeechRecognitionErrorEvent extends Event {
  readonly error: string;
  readonly message: string;
}

interface ISpeechRecognition extends EventTarget {
  lang: string;
  continuous: boolean;
  interimResults: boolean;
  maxAlternatives: number;
  onstart: ((this: ISpeechRecognition, ev: Event) => void) | null;
  onend: ((this: ISpeechRecognition, ev: Event) => void) | null;
  onresult: ((this: ISpeechRecognition, ev: ISpeechRecognitionEvent) => void) | null;
  onerror: ((this: ISpeechRecognition, ev: ISpeechRecognitionErrorEvent) => void) | null;
  start(): void;
  stop(): void;
  abort(): void;
}

interface ISpeechRecognitionConstructor {
  new (): ISpeechRecognition;
}

// Extend Window to include vendor-prefixed SpeechRecognition
declare global {
  interface Window {
    SpeechRecognition: ISpeechRecognitionConstructor | undefined;
    webkitSpeechRecognition: ISpeechRecognitionConstructor | undefined;
  }
}

// ── Hook types ────────────────────────────────────────────────────────────────

export interface UseSpeechToTextOptions {
  /** Called with the final transcript text when speech ends */
  onTranscript: (text: string) => void;
  /**
   * BCP-47 language tag. Defaults to 'en-US'.
   * Pass any tag the browser supports — 'fr-FR', 'yo', 'ha', 'ig', 'ar-SA', etc.
   */
  language?: string;
  /** If true, interim (in-progress) results are also passed to onTranscript */
  interimResults?: boolean;
}

export interface UseSpeechToTextReturn {
  speechState: SpeechState;
  speechError: string;
  interimText: string;
  /** Toggle: starts listening if idle/error, stops if listening */
  handleMicClick: () => void;
  /** True while the mic is active (listening or processing) */
  isMicBusy: boolean;
  /** True if the current browser supports the Web Speech API */
  isSupported: boolean;
}

// ── Hook implementation ───────────────────────────────────────────────────────

export function useSpeechToText({
  onTranscript,
  language = 'en-US',
  interimResults = true,
}: UseSpeechToTextOptions): UseSpeechToTextReturn {
  const [speechState, setSpeechState] = useState<SpeechState>('idle');
  const [speechError, setSpeechError] = useState<string>('');
  const [interimText, setInterimText] = useState<string>('');

  const recognitionRef = useRef<ISpeechRecognition | null>(null);
  const finalTextRef   = useRef<string>('');

  // Detect support once on mount (SSR-safe)
  const isSupported =
    typeof window !== 'undefined' &&
    !!(window.SpeechRecognition || window.webkitSpeechRecognition);

  // Cleanup on unmount
  useEffect(() => {
    return () => {
      recognitionRef.current?.abort();
    };
  }, []);

  const startListening = useCallback(() => {
    if (!isSupported) {
      setSpeechError('Voice input is not supported in this browser. Please use Chrome, Edge, or Safari.');
      setSpeechState('error');
      setTimeout(() => { setSpeechState('idle'); setSpeechError(''); }, 5000);
      return;
    }

    setSpeechError('');
    setInterimText('');
    finalTextRef.current = '';

    const SpeechRecognitionClass =
      (window.SpeechRecognition || window.webkitSpeechRecognition)!;
    const recognition = new SpeechRecognitionClass();

    recognition.lang            = language;
    recognition.interimResults  = interimResults;
    recognition.maxAlternatives = 1;
    // continuous = false: stops automatically after a natural pause
    recognition.continuous      = false;

    recognition.onstart = () => {
      setSpeechState('listening');
    };

    recognition.onresult = (event: ISpeechRecognitionEvent) => {
      let interim = '';
      let final   = '';
      for (let i = event.resultIndex; i < event.results.length; i++) {
        const result = event.results[i];
        if (result.isFinal) {
          final += result[0].transcript;
        } else {
          interim += result[0].transcript;
        }
      }
      if (final) finalTextRef.current += final;
      setInterimText(interim);
    };

    recognition.onend = () => {
      setInterimText('');
      const transcript = finalTextRef.current.trim();
      if (transcript) {
        onTranscript(transcript);
      }
      setSpeechState('idle');
    };

    recognition.onerror = (event: ISpeechRecognitionErrorEvent) => {
      setInterimText('');
      let msg = '';
      switch (event.error) {
        case 'not-allowed':
        case 'permission-denied':
          msg = 'Microphone permission denied — please allow access in your browser settings.';
          break;
        case 'no-speech':
          msg = 'No speech detected — please try again.';
          break;
        case 'network':
          msg = 'Network error during voice recognition — please check your connection.';
          break;
        case 'audio-capture':
          msg = 'No microphone found — please connect a microphone and try again.';
          break;
        case 'aborted':
          // User stopped manually — not an error
          setSpeechState('idle');
          return;
        default:
          msg = `Voice recognition error: ${event.error}`;
      }
      setSpeechError(msg);
      setSpeechState('error');
      setTimeout(() => { setSpeechState('idle'); setSpeechError(''); }, 5000);
    };

    recognitionRef.current = recognition;
    recognition.start();
  }, [isSupported, language, interimResults, onTranscript]);

  const stopListening = useCallback(() => {
    if (recognitionRef.current) {
      recognitionRef.current.stop(); // triggers onend → onTranscript
    }
  }, []);

  const handleMicClick = useCallback(() => {
    if (speechState === 'listening') {
      stopListening();
    } else if (speechState === 'idle' || speechState === 'error') {
      startListening();
    }
  }, [speechState, startListening, stopListening]);

  return {
    speechState,
    speechError,
    interimText,
    handleMicClick,
    isMicBusy: speechState === 'listening' || speechState === 'processing',
    isSupported,
  };
}

/**
 * Code download utilities for AI Studio
 * Maps programming languages to file extensions and provides download helpers
 */

// Language to file extension mapping
export const LANG_TO_EXT: Record<string, string> = {
  python: 'py',
  javascript: 'js',
  typescript: 'ts',
  js: 'js',
  ts: 'ts',
  jsx: 'jsx',
  tsx: 'tsx',
  html: 'html',
  css: 'css',
  scss: 'scss',
  sass: 'sass',
  sql: 'sql',
  bash: 'sh',
  sh: 'sh',
  shell: 'sh',
  go: 'go',
  golang: 'go',
  rust: 'rs',
  java: 'java',
  kotlin: 'kt',
  swift: 'swift',
  dart: 'dart',
  json: 'json',
  yaml: 'yaml',
  yml: 'yml',
  markdown: 'md',
  md: 'md',
  xml: 'xml',
  php: 'php',
  ruby: 'rb',
  c: 'c',
  cpp: 'cpp',
  'c++': 'cpp',
  csharp: 'cs',
  'c#': 'cs',
  r: 'r',
  matlab: 'm',
  lua: 'lua',
  perl: 'pl',
  scala: 'scala',
  haskell: 'hs',
  elixir: 'ex',
  clojure: 'clj',
  vue: 'vue',
  svelte: 'svelte',
  dockerfile: 'Dockerfile',
  docker: 'Dockerfile',
  makefile: 'Makefile',
  code: 'txt', // fallback
};

/**
 * Get file extension for a programming language
 */
export function getExtensionForLanguage(lang: string): string {
  const normalized = lang.toLowerCase().trim();
  return LANG_TO_EXT[normalized] || 'txt';
}

/**
 * Download code as a file with proper extension
 */
export function downloadCode(code: string, language: string, filename?: string): void {
  const extension = getExtensionForLanguage(language);
  const defaultFilename = `code.${extension}`;
  const finalFilename = filename || defaultFilename;
  
  const blob = new Blob([code], { type: 'text/plain;charset=utf-8' });
  const url = URL.createObjectURL(blob);
  const a = document.createElement('a');
  a.href = url;
  a.download = finalFilename;
  document.body.appendChild(a);
  a.click();
  document.body.removeChild(a);
  URL.revokeObjectURL(url);
}

/**
 * Download multiple code files as a zip (requires JSZip)
 * For now, we'll just download them individually with numbered filenames
 */
export function downloadMultipleCodeFiles(
  files: Array<{ code: string; language: string; filename?: string }>
): void {
  files.forEach((file, index) => {
    const extension = getExtensionForLanguage(file.language);
    const filename = file.filename || `file_${index + 1}.${extension}`;
    setTimeout(() => {
      downloadCode(file.code, file.language, filename);
    }, index * 200); // Stagger downloads to avoid browser blocking
  });
}

/**
 * Detect programming language from code content
 * Simple heuristic-based detection
 */
export function detectLanguageFromCode(code: string): string {
  const trimmed = code.trim();
  
  // Python
  if (/^(def |class |import |from .* import|if __name__ == ['"]__main__['"])/.test(trimmed)) {
    return 'python';
  }
  
  // JavaScript/TypeScript
  if (/^(const |let |var |function |import |export |async |class |interface |type )/.test(trimmed)) {
    if (/: (string|number|boolean|any)\b/.test(code)) return 'typescript';
    return 'javascript';
  }
  
  // SQL
  if (/^(SELECT|INSERT|UPDATE|DELETE|CREATE|ALTER|DROP|WITH)\b/i.test(trimmed)) {
    return 'sql';
  }
  
  // HTML
  if (/^<!DOCTYPE html>|^<html|^<\!--/.test(trimmed)) {
    return 'html';
  }
  
  // CSS
  if (/^[.#]?[\w-]+\s*\{/.test(trimmed) && /:\s*[^;]+;/.test(code)) {
    return 'css';
  }
  
  // JSON
  if (/^\{[\s\S]*\}$|^\[[\s\S]*\]$/.test(trimmed)) {
    try {
      JSON.parse(trimmed);
      return 'json';
    } catch {
      // Not valid JSON
    }
  }
  
  // Go
  if (/^package |^func |^import \(/.test(trimmed)) {
    return 'go';
  }
  
  // Bash/Shell
  if (/^#!\/bin\/(ba)?sh|^#!/.test(trimmed)) {
    return 'bash';
  }
  
  // Java
  if (/^(public |private |protected )?(class|interface|enum) /.test(trimmed)) {
    return 'java';
  }
  
  // Fallback
  return 'code';
}

/**
 * PDF generation utilities for AI Studio document tools
 * Uses browser's print API for PDF generation (no external dependencies needed)
 */

// Document tool slugs that should offer PDF download
export const DOCUMENT_TOOL_SLUGS = new Set([
  'bizplan',
  'business-plan',
  'business-plan-summary',
  'study-guide',
  'research-brief',
  'deep-research-brief',
  'summary',
  'translate',
  'local-translation',
  'transcribe',
  'transcribe-african',
  'voice-to-text',
]);

// Tool slug to document title mapping
export const TOOL_DOCUMENT_TITLES: Record<string, string> = {
  'bizplan': 'Business Plan',
  'business-plan': 'Business Plan',
  'business-plan-summary': 'Business Plan',
  'study-guide': 'Study Guide',
  'research-brief': 'Research Brief',
  'deep-research-brief': 'Research Brief',
  'summary': 'Summary',
  'translate': 'Translation',
  'local-translation': 'Translation',
  'transcribe': 'Transcript',
  'transcribe-african': 'Transcript',
  'voice-to-text': 'Transcript',
};

/**
 * Convert markdown-like text to HTML for PDF rendering
 */
function markdownToHtml(text: string): string {
  let html = text
    // Escape HTML
    .replace(/&/g, '&amp;')
    .replace(/</g, '&lt;')
    .replace(/>/g, '&gt;')
    // Headers
    .replace(/^### (.+)$/gm, '<h3>$1</h3>')
    .replace(/^## (.+)$/gm, '<h2>$1</h2>')
    .replace(/^# (.+)$/gm, '<h1>$1</h1>')
    // Bold
    .replace(/\*\*(.+?)\*\*/g, '<strong>$1</strong>')
    // Italic
    .replace(/\*(.+?)\*/g, '<em>$1</em>')
    // Inline code
    .replace(/`(.+?)`/g, '<code>$1</code>')
    // Bullet lists
    .replace(/^[-*•] (.+)$/gm, '<li>$1</li>')
    // Numbered lists
    .replace(/^\d+\. (.+)$/gm, '<li>$1</li>')
    // Horizontal rules
    .replace(/^---+$/gm, '<hr>')
    // Paragraphs (double newline)
    .replace(/\n\n/g, '</p><p>')
    // Single newlines
    .replace(/\n/g, '<br>');

  // Wrap consecutive <li> items in <ul> (using gi flags for compatibility)
  html = html.replace(/(<li>[^<]*<\/li>(<br>)?)+/gi, (match) => {
    return `<ul>${match.replace(/<br>/g, '')}</ul>`;
  });

  return `<p>${html}</p>`;
}

/**
 * Generate and download a PDF from text content
 * Uses browser's print-to-PDF functionality
 */
export function downloadAsPDF(
  content: string,
  toolSlug: string,
  toolName: string,
  prompt?: string
): void {
  const title = TOOL_DOCUMENT_TITLES[toolSlug] || toolName || 'Document';
  const date = new Date().toLocaleDateString('en-US', {
    year: 'numeric',
    month: 'long',
    day: 'numeric',
  });

  const htmlContent = markdownToHtml(content);

  const printWindow = window.open('', '_blank');
  if (!printWindow) {
    // Fallback: download as text if popup blocked
    downloadAsMarkdown(content, toolSlug, toolName);
    return;
  }

  printWindow.document.write(`
    <!DOCTYPE html>
    <html>
    <head>
      <meta charset="UTF-8">
      <title>${title}</title>
      <style>
        * { margin: 0; padding: 0; box-sizing: border-box; }
        
        body {
          font-family: 'Georgia', 'Times New Roman', serif;
          font-size: 12pt;
          line-height: 1.7;
          color: #1a1a2e;
          background: #ffffff;
          padding: 0;
          margin: 0;
        }
        
        .document-wrapper {
          max-width: 800px;
          margin: 0 auto;
          padding: 60px 80px;
        }
        
        /* Header */
        .doc-header {
          border-bottom: 3px solid #c9a227;
          padding-bottom: 24px;
          margin-bottom: 32px;
        }
        
        .brand {
          font-size: 10pt;
          color: #c9a227;
          font-weight: 700;
          letter-spacing: 2px;
          text-transform: uppercase;
          margin-bottom: 8px;
        }
        
        .doc-title {
          font-size: 28pt;
          font-weight: 700;
          color: #1a1a2e;
          line-height: 1.2;
          margin-bottom: 8px;
        }
        
        .doc-meta {
          font-size: 10pt;
          color: #666;
          display: flex;
          gap: 16px;
        }
        
        /* Prompt box */
        .prompt-box {
          background: #f8f4e8;
          border-left: 4px solid #c9a227;
          padding: 12px 16px;
          margin-bottom: 28px;
          border-radius: 0 8px 8px 0;
        }
        
        .prompt-label {
          font-size: 9pt;
          font-weight: 700;
          color: #c9a227;
          text-transform: uppercase;
          letter-spacing: 1px;
          margin-bottom: 4px;
        }
        
        .prompt-text {
          font-size: 11pt;
          color: #444;
          font-style: italic;
        }
        
        /* Content */
        .doc-content {
          font-size: 11pt;
          line-height: 1.8;
          color: #2d2d2d;
        }
        
        h1 {
          font-size: 20pt;
          font-weight: 700;
          color: #1a1a2e;
          margin: 28px 0 12px;
          padding-bottom: 6px;
          border-bottom: 1px solid #e0d5b0;
        }
        
        h2 {
          font-size: 16pt;
          font-weight: 700;
          color: #1a1a2e;
          margin: 24px 0 10px;
        }
        
        h3 {
          font-size: 13pt;
          font-weight: 700;
          color: #333;
          margin: 20px 0 8px;
        }
        
        p {
          margin-bottom: 12px;
        }
        
        ul, ol {
          margin: 8px 0 12px 24px;
        }
        
        li {
          margin-bottom: 6px;
        }
        
        strong {
          font-weight: 700;
          color: #1a1a2e;
        }
        
        em {
          font-style: italic;
          color: #555;
        }
        
        code {
          font-family: 'Courier New', monospace;
          font-size: 10pt;
          background: #f4f4f4;
          padding: 1px 4px;
          border-radius: 3px;
          color: #c7254e;
        }
        
        hr {
          border: none;
          border-top: 1px solid #e0d5b0;
          margin: 24px 0;
        }
        
        /* Footer */
        .doc-footer {
          margin-top: 48px;
          padding-top: 16px;
          border-top: 1px solid #e0d5b0;
          font-size: 9pt;
          color: #999;
          display: flex;
          justify-content: space-between;
        }
        
        @media print {
          body { padding: 0; }
          .document-wrapper { padding: 40px 60px; }
          .no-print { display: none !important; }
          
          @page {
            margin: 20mm 25mm;
            size: A4;
          }
        }
        
        /* Print button */
        .print-btn {
          position: fixed;
          top: 20px;
          right: 20px;
          background: #c9a227;
          color: white;
          border: none;
          padding: 10px 20px;
          border-radius: 8px;
          font-size: 13px;
          font-weight: 600;
          cursor: pointer;
          box-shadow: 0 4px 12px rgba(0,0,0,0.2);
          z-index: 1000;
        }
        
        .print-btn:hover {
          background: #b8911f;
        }
        
        @media print {
          .print-btn { display: none; }
        }
      </style>
    </head>
    <body>
      <button class="print-btn no-print" onclick="window.print()">
        ⬇ Save as PDF
      </button>
      
      <div class="document-wrapper">
        <!-- Header -->
        <div class="doc-header">
          <div class="brand">Nexus AI Studio</div>
          <div class="doc-title">${title}</div>
          <div class="doc-meta">
            <span>📅 ${date}</span>
            <span>🤖 Generated by Nexus AI</span>
          </div>
        </div>
        
        ${prompt ? `
        <!-- Prompt box -->
        <div class="prompt-box">
          <div class="prompt-label">Your Request</div>
          <div class="prompt-text">${prompt.replace(/</g, '&lt;').replace(/>/g, '&gt;')}</div>
        </div>
        ` : ''}
        
        <!-- Main content -->
        <div class="doc-content">
          ${htmlContent}
        </div>
        
        <!-- Footer -->
        <div class="doc-footer">
          <span>Generated by Nexus AI Studio · loyalty-nexus.com</span>
          <span>${date}</span>
        </div>
      </div>
      
      <script>
        // Auto-trigger print dialog after a short delay for better UX
        setTimeout(() => {
          window.print();
        }, 800);
      </script>
    </body>
    </html>
  `);

  printWindow.document.close();
}

/**
 * Download content as a markdown file
 */
export function downloadAsMarkdown(
  content: string,
  toolSlug: string,
  toolName: string,
  prompt?: string
): void {
  const title = TOOL_DOCUMENT_TITLES[toolSlug] || toolName || 'Document';
  const date = new Date().toISOString().split('T')[0];
  
  const header = `# ${title}\n\n**Generated by Nexus AI Studio** · ${new Date().toLocaleDateString()}\n\n${prompt ? `> **Your request:** ${prompt}\n\n` : ''}---\n\n`;
  const fullContent = header + content;
  
  const blob = new Blob([fullContent], { type: 'text/markdown;charset=utf-8' });
  const url = URL.createObjectURL(blob);
  const a = document.createElement('a');
  a.href = url;
  a.download = `${toolSlug || 'nexus'}-${date}.md`;
  document.body.appendChild(a);
  a.click();
  document.body.removeChild(a);
  URL.revokeObjectURL(url);
}

/**
 * Download content as a plain text file
 */
export function downloadAsText(
  content: string,
  toolSlug: string,
  toolName: string
): void {
  const date = new Date().toISOString().split('T')[0];
  const blob = new Blob([content], { type: 'text/plain;charset=utf-8' });
  const url = URL.createObjectURL(blob);
  const a = document.createElement('a');
  a.href = url;
  a.download = `${toolSlug || 'nexus'}-${date}.txt`;
  document.body.appendChild(a);
  a.click();
  document.body.removeChild(a);
  URL.revokeObjectURL(url);
}

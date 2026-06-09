export function normalizeRichTextInput(value: string): string {
  const unix = value.replace(/\r\n/g, '\n');
  const trimmedEdges = unix.replace(/^\n+/, '').replace(/\n+$/, '');
  return trimmedEdges.replace(/\n{3,}/g, '\n\n');
}

function escapeHtml(value: string): string {
  return value
    .replace(/&/g, '&amp;')
    .replace(/</g, '&lt;')
    .replace(/>/g, '&gt;')
    .replace(/"/g, '&quot;')
    .replace(/'/g, '&#39;');
}

export function renderRichText(value: string): string {
  const normalized = normalizeRichTextInput(value);
  let html = escapeHtml(normalized);
  html = html.replace(/\*\*([^\n*][\s\S]*?[^\n*])\*\*/g, '<strong>$1</strong>');
  html = html.replace(/\+\+([^\n+][\s\S]*?[^\n+])\+\+/g, '<u>$1</u>');
  html = html.replace(/(^|[^*])\*([^*\n][\s\S]*?[^*\n])\*(?=$|[^*])/g, '$1<em>$2</em>');
  return html.replace(/\n/g, '<br>');
}

export function renderRichTextForEditor(value: string): string {
  const html = renderRichText(value);
  return html || '<br>';
}

export function editorHtmlToRichText(html: string): string {
  const root = document.createElement('div');
  root.innerHTML = html;

  const wrapStyledContent = (el: HTMLElement, content: string) => {
    let next = content;
    const fontWeight = el.style.fontWeight || '';
    const fontStyle = el.style.fontStyle || '';
    const textDecoration = el.style.textDecoration || el.style.textDecorationLine || '';

    if (fontWeight === 'bold' || Number.parseInt(fontWeight, 10) >= 600) {
      next = `**${next}**`;
    }
    if (fontStyle === 'italic') {
      next = `*${next}*`;
    }
    if (textDecoration.includes('underline')) {
      next = `++${next}++`;
    }
    return next;
  };

  const walk = (node: Node): string => {
    if (node.nodeType === Node.TEXT_NODE) {
      return (node.textContent || '').replace(/\u00A0/g, ' ');
    }
    if (node.nodeType !== Node.ELEMENT_NODE) {
      return '';
    }

    const el = node as HTMLElement;
    const tag = el.tagName.toLowerCase();
    const content = Array.from(el.childNodes).map(walk).join('');

    if (tag === 'br') return '\n';
    if (tag === 'strong' || tag === 'b') return `**${content}**`;
    if (tag === 'em' || tag === 'i') return `*${content}*`;
    if (tag === 'u') return `++${content}++`;
    if (tag === 'div' || tag === 'p') return `${content}\n`;
    if (tag === 'span') return wrapStyledContent(el, content);
    return content;
  };

  const raw = Array.from(root.childNodes).map(walk).join('');
  return normalizeRichTextInput(raw);
}

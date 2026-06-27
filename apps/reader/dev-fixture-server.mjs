// Throwaway fixture server for manually exercising the reader's connected mode without a
// real ComicHub server. Serves the docs/03-api.md reader endpoints with generated SVG
// pages. Run: node apps/reader/dev-fixture-server.mjs  (listens on 127.0.0.1:8099)
import { createServer } from 'node:http';

const PAGE_COUNT = 12;
const BOOK_ID = 'test';
let progress = { page: 0, status: 'in_progress', updatedAt: new Date().toISOString() };

function pageSvg(idx, w, h, label) {
  const hue = (idx * 37) % 360;
  return `<svg xmlns="http://www.w3.org/2000/svg" width="${w}" height="${h}" viewBox="0 0 ${w} ${h}">
    <rect width="100%" height="100%" fill="hsl(${hue} 45% 18%)"/>
    <rect x="20" y="20" width="${w - 40}" height="${h - 40}" fill="none" stroke="hsl(${hue} 70% 60%)" stroke-width="6"/>
    <text x="50%" y="50%" font-family="monospace" font-size="${Math.round(w / 5)}" fill="hsl(${hue} 80% 75%)" text-anchor="middle" dominant-baseline="middle">${label}</text>
  </svg>`;
}

function meta(idx) {
  const double = idx === 6;
  return { idx, w: double ? 1600 : 800, h: 1200, type: idx === 0 ? 'FrontCover' : 'Story', double };
}

const cors = {
  'Access-Control-Allow-Origin': '*',
  'Access-Control-Allow-Methods': 'GET,PUT,POST,OPTIONS',
  'Access-Control-Allow-Headers': 'Authorization,Content-Type,Accept',
};

const server = createServer((req, res) => {
  const url = new URL(req.url, 'http://x');
  const send = (code, type, body) => {
    res.writeHead(code, { 'Content-Type': type, ...cors });
    res.end(body);
  };
  if (req.method === 'OPTIONS') return send(204, 'text/plain', '');

  const p = url.pathname;
  if (p === `/api/v1/books/${BOOK_ID}/manifest`) {
    return send(200, 'application/json', JSON.stringify({
      bookId: BOOK_ID, pageCount: PAGE_COUNT, readingDir: 'ltr',
      pages: Array.from({ length: PAGE_COUNT }, (_, i) => meta(i)),
    }));
  }
  let m = p.match(new RegExp(`^/api/v1/books/${BOOK_ID}/pages/(\\d+)/thumb$`));
  if (m) { const i = +m[1]; const d = meta(i); return send(200, 'image/svg+xml', pageSvg(i, d.w / 4, d.h / 4, String(i + 1))); }
  m = p.match(new RegExp(`^/api/v1/books/${BOOK_ID}/pages/(\\d+)$`));
  if (m) { const i = +m[1]; const d = meta(i); return send(200, 'image/svg+xml', pageSvg(i, d.w, d.h, `PAGE ${i + 1}`)); }
  if (p === `/api/v1/books/${BOOK_ID}/prefetch`) return send(200, 'application/json', '{}');
  if (p === `/api/v1/me/progress/${BOOK_ID}`) {
    if (req.method === 'PUT') {
      let body = '';
      req.on('data', (c) => (body += c));
      req.on('end', () => {
        try { const j = JSON.parse(body || '{}'); progress = { ...progress, ...j, updatedAt: new Date().toISOString() }; } catch { /* ignore */ }
        send(200, 'application/json', JSON.stringify(progress));
      });
      return;
    }
    return send(200, 'application/json', JSON.stringify(progress));
  }
  send(404, 'application/json', JSON.stringify({ error: { code: 'not_found', message: p } }));
});

server.listen(8099, '127.0.0.1', () => console.log('fixture server on http://127.0.0.1:8099'));

// Tab panel switching
document.querySelectorAll('.tab[data-tab]').forEach(tab => {
  tab.addEventListener('click', () => {
    const target = tab.dataset.tab;
    document.querySelectorAll('.tab').forEach(t => t.classList.remove('active'));
    tab.classList.add('active');
    document.querySelectorAll('.tab-panel').forEach(p => p.classList.remove('active'));
    const panel = document.getElementById('panel-' + target);
    if (panel) panel.classList.add('active');
  });
});

// ── App status checks ───────────────────────────────────────────────────────
//
// Each card carries one of four states from two independent reachability probes:
//   online  = server OK && client OK   → opening allowed
//   partial = exactly one OK           → likely a network/provider block
//   offline = both bad
// Only `online` cards may open. The colour dot is the only indicator (no text).

const STATES = ['checking', 'online', 'partial', 'offline'];

function setState(card, state) {
  card.classList.remove(...STATES);
  card.classList.add(state);
  card.classList.toggle('blocked', state !== 'online');
}

function currentState(card) {
  return STATES.find(s => card.classList.contains(s));
}

function computeState(srvOk, clientOk) {
  return srvOk && clientOk ? 'online'
       : !srvOk && !clientOk ? 'offline'
       : 'partial';
}

// Client-side ping: no-cors fetch resolves (opaque) when reachable from the
// user's network, rejects on DNS/connection/provider-block failures.
async function pingClient(url) {
  try {
    await fetch(url, { mode: 'no-cors', signal: AbortSignal.timeout(4000) });
    return true;
  } catch {
    return false;
  }
}

// server-side reachability of every app at once
function fetchServer() {
  return fetch('/status').then(r => r.json()).catch(() => ({}));
}

async function checkStatus() {
  const cards = [...document.querySelectorAll('.app-card')];
  if (!cards.length) return;

  const refresh = document.getElementById('appRefresh');
  if (refresh) refresh.classList.add('spinning');
  cards.forEach(c => setState(c, 'checking'));

  const server = await fetchServer();
  await Promise.all(cards.map(async card => {
    const clientOk = await pingClient(card.dataset.url);
    setState(card, computeState(!!server[card.dataset.slug], clientOk));
  }));

  if (refresh) refresh.classList.remove('spinning');
}

// ── Status popup (glassy) ────────────────────────────────────────────────────

const STATUS_COPY = {
  partial: {
    kind: 'warn',
    title: 'Соединение не проходит',
    text: 'Сервис работает, но ваша сеть, похоже, блокирует доступ. Скорее всего, дело не в нас, а в провайдере.',
  },
  offline: {
    kind: 'bad',
    title: 'Сервис недоступен',
    text: 'Упс! Приложение сейчас не отвечает. Мы уже работаем над этим — загляните чуть позже.',
  },
};

const statusModal = document.getElementById('statusModal');

function showStatusPopup(state) {
  const c = STATUS_COPY[state];
  if (!c || !statusModal) return;
  document.getElementById('statusModalTitle').textContent = c.title;
  document.getElementById('statusModalText').textContent = c.text;
  const box = statusModal.querySelector('.status-modal');
  box.classList.remove('warn', 'bad');
  box.classList.add(c.kind);
  statusModal.classList.add('open');
}

function closeStatusModal() {
  if (statusModal) statusModal.classList.remove('open');
}

if (statusModal) {
  statusModal.addEventListener('click', e => { if (e.target === statusModal) closeStatusModal(); });
  const closeBtn = document.getElementById('statusModalClose');
  if (closeBtn) closeBtn.addEventListener('click', closeStatusModal);
}

// ── Open flow with pre-redirect re-check ─────────────────────────────────────
//
// The page status may be stale (e.g. user idle 5 min on a green card). So on a
// click we re-verify reachability at that exact moment; only a fresh `online`
// result navigates, otherwise we surface the matching popup.

let opening = false;

document.querySelectorAll('.app-open').forEach(link => {
  link.addEventListener('click', async e => {
    e.preventDefault();
    if (opening) return;

    const card = link.closest('.app-card');
    const state = currentState(card);

    if (state === 'checking') return;                 // still resolving — ignore
    if (state === 'partial' || state === 'offline') { // already known bad
      showStatusPopup(state);
      return;
    }

    // online → re-verify right now before letting the user through
    opening = true;
    setState(card, 'checking');
    const [server, clientOk] = await Promise.all([fetchServer(), pingClient(card.dataset.url)]);
    const fresh = computeState(!!server[card.dataset.slug], clientOk);
    setState(card, fresh);
    opening = false;

    if (fresh === 'online') {
      window.location.href = link.getAttribute('href');
    } else {
      showStatusPopup(fresh);
    }
  });
});

// ── Info modal ──────────────────────────────────────────────────────────────

const modal = document.getElementById('appModal');

function openModal(btn) {
  document.getElementById('appModalName').textContent = btn.dataset.name;
  document.getElementById('appModalDesc').textContent = btn.dataset.desc;

  const ul = document.getElementById('appModalFeatures');
  ul.innerHTML = '';
  const feats = (btn.dataset.features || '').split('|').map(s => s.trim()).filter(Boolean);
  feats.forEach(f => {
    const li = document.createElement('li');
    li.textContent = f;
    ul.appendChild(li);
  });
  ul.style.display = feats.length ? '' : 'none';

  modal.classList.add('open');
}
function closeModal() {
  if (modal) modal.classList.remove('open');
}

document.querySelectorAll('.app-info-btn').forEach(btn => {
  btn.addEventListener('click', () => openModal(btn));
});

if (modal) {
  modal.addEventListener('click', e => { if (e.target === modal) closeModal(); });
  const closeBtn = document.getElementById('appModalClose');
  if (closeBtn) closeBtn.addEventListener('click', closeModal);
}

// Esc closes whichever popup is open
document.addEventListener('keydown', e => {
  if (e.key === 'Escape') { closeModal(); closeStatusModal(); }
});

// ── Copy email to clipboard (info tab) ───────────────────────────────────────

const toast = document.getElementById('toast');
let toastTimer;
function showToast(msg) {
  if (!toast) return;
  toast.textContent = msg;
  toast.classList.add('show');
  clearTimeout(toastTimer);
  toastTimer = setTimeout(() => toast.classList.remove('show'), 2400);
}

async function copyText(text) {
  try {
    if (navigator.clipboard) { await navigator.clipboard.writeText(text); return true; }
  } catch { /* fall through to legacy path */ }
  try {
    const ta = document.createElement('textarea');
    ta.value = text;
    ta.style.position = 'fixed';
    ta.style.opacity = '0';
    document.body.appendChild(ta);
    ta.select();
    const ok = document.execCommand('copy');
    document.body.removeChild(ta);
    return ok;
  } catch {
    return false;
  }
}

const copyEmailBtn = document.getElementById('copyEmail');
if (copyEmailBtn) {
  copyEmailBtn.addEventListener('click', async () => {
    const email = copyEmailBtn.dataset.email;
    const ok = await copyText(email);
    showToast(ok ? 'Email скопирован в буфер обмена' : email);
  });
}

// kick off the first check
checkStatus();
const refreshBtn = document.getElementById('appRefresh');
if (refreshBtn) refreshBtn.addEventListener('click', checkStatus);

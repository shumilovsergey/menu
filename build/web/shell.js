// ── Profile button: set initial letter + toggle popover ──────────────────
const profileBtn = document.getElementById('profileBtn');
if (profileBtn) {
  const name    = profileBtn.dataset.name || '';
  const initial = profileBtn.querySelector('.profile-initial');
  if (initial) initial.textContent = (name[0] || '?').toUpperCase();

  const popover = document.getElementById('profilePopover');
  profileBtn.addEventListener('click', () => {
    const open = popover.classList.toggle('open');
    profileBtn.classList.toggle('open', open);
  });
  document.addEventListener('click', e => {
    if (!profileBtn.contains(e.target) && !popover.contains(e.target)) {
      popover.classList.remove('open');
      profileBtn.classList.remove('open');
    }
  });
}

// ── Tab switching ─────────────────────────────────────────────────────────
document.querySelectorAll('.tab').forEach(tab => {
  tab.addEventListener('click', () => {
    document.querySelectorAll('.tab').forEach(t => t.classList.remove('active'));
    tab.classList.add('active');
  });
});


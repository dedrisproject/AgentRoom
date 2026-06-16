/* AgentRoom — vanilla JS, no framework */

// ---- i18n helper ----
function t(key) {
  return (window.AR && window.AR.t && window.AR.t[key]) || key;
}

// ---- Modal helpers ----

function openModal(id) {
  document.getElementById(id).classList.add('open');
}

function closeModal(id) {
  document.getElementById(id).classList.remove('open');
}

document.addEventListener('click', function(e) {
  if (e.target.classList.contains('modal-overlay')) {
    e.target.classList.remove('open');
  }
});

document.addEventListener('keydown', function(e) {
  if (e.key === 'Escape') {
    document.querySelectorAll('.modal-overlay.open').forEach(function(m) {
      m.classList.remove('open');
    });
  }
});

// ---- Data-action event delegation ----
// Instead of inline onclick, buttons use data-action attributes.
document.addEventListener('click', function(e) {
  var btn = e.target.closest('[data-action]');
  if (!btn) return;
  var action = btn.dataset.action;

  switch (action) {
    case 'delete-room':
      deleteRoom(btn.dataset.id, btn.dataset.name);
      break;
    case 'edit-agent':
      openEditAgentModal(btn.dataset.id, btn.dataset.name, btn.dataset.role, btn.dataset.repo);
      break;
    case 'view-instructions':
      viewInstructions(btn.dataset.id);
      break;
    case 'delete-agent':
      deleteAgent(btn.dataset.id, btn.dataset.name);
      break;
    case 'open-reply':
      openReplyModal(btn.dataset.id, btn.dataset.subject);
      break;
    case 'close-thread':
      closeThread(btn.dataset.id);
      break;
  }
});

// ---- Error helpers ----

function showError(id, msg) {
  var el = document.getElementById(id);
  if (el) { el.textContent = msg; el.classList.remove('hidden'); }
}

function hideError(id) {
  var el = document.getElementById(id);
  if (el) { el.textContent = ''; el.classList.add('hidden'); }
}

// ---- Auth ----

function logout() {
  fetch('/api/admin/logout', { method: 'POST' })
    .then(function() { window.location.href = '/login'; });
}

// ---- Login page ----

var loginForm = document.getElementById('login-form');
if (loginForm) {
  loginForm.addEventListener('submit', function(e) {
    e.preventDefault();
    hideError('login-error');
    var password = document.getElementById('password').value;
    fetch('/api/admin/login', {
      method: 'POST',
      headers: { 'Content-Type': 'application/x-www-form-urlencoded' },
      body: 'password=' + encodeURIComponent(password)
    })
    .then(function(r) { return r.json(); })
    .then(function(data) {
      if (data.success) {
        window.location.href = '/';
      } else {
        showError('login-error', data.message || t('login.error'));
      }
    })
    .catch(function() {
      showError('login-error', t('error.network'));
    });
  });
}

// ---- Dashboard: create room ----

function submitCreateRoom(e) {
  e.preventDefault();
  hideError('create-room-error');
  var name = document.getElementById('room-name').value;
  fetch('/api/admin/rooms', {
    method: 'POST',
    headers: { 'Content-Type': 'application/x-www-form-urlencoded' },
    body: 'name=' + encodeURIComponent(name)
  })
  .then(function(r) { return r.json(); })
  .then(function(data) {
    if (data.success) {
      window.location.href = '/rooms/' + data.room.id;
    } else {
      showError('create-room-error', data.message || t('error.create_room'));
    }
  })
  .catch(function() {
    showError('create-room-error', t('error.network'));
  });
}

function deleteRoom(id, name) {
  var msg = t('confirm.delete_room').replace('%s', name);
  if (!confirm(msg)) return;
  fetch('/api/admin/rooms/' + id, { method: 'DELETE' })
  .then(function(r) { return r.json(); })
  .then(function(data) {
    if (data.success) {
      window.location.reload();
    } else {
      alert(data.message || t('error.delete_room'));
    }
  });
}

// ---- Room view: agents ----

function openAddAgentModal() {
  document.getElementById('add-agent-modal-title').textContent = t('modal.agent.add_title');
  document.getElementById('add-agent-submit').textContent = t('modal.agent.add_submit');
  document.getElementById('agent-id').value = '';
  document.getElementById('agent-name').value = '';
  document.getElementById('agent-name').disabled = false;
  document.getElementById('agent-role').value = '';
  document.getElementById('agent-repo').value = '';
  hideError('add-agent-error');
  openModal('add-agent-modal');
}

function openEditAgentModal(id, name, role, repo) {
  document.getElementById('add-agent-modal-title').textContent = t('modal.agent.edit_title');
  document.getElementById('add-agent-submit').textContent = t('modal.agent.edit_submit');
  document.getElementById('agent-id').value = id;
  document.getElementById('agent-name').value = name;
  document.getElementById('agent-name').disabled = false;
  document.getElementById('agent-role').value = role;
  document.getElementById('agent-repo').value = repo;
  hideError('add-agent-error');
  openModal('add-agent-modal');
}

function submitAgent(e) {
  e.preventDefault();
  hideError('add-agent-error');
  var id = document.getElementById('agent-id').value;
  var name = document.getElementById('agent-name').value;
  var role = document.getElementById('agent-role').value;
  var repo = document.getElementById('agent-repo').value;
  var body = 'name=' + encodeURIComponent(name) +
             '&role=' + encodeURIComponent(role) +
             '&repo=' + encodeURIComponent(repo);

  var method = id ? 'PUT' : 'POST';
  var url = id ? '/api/admin/agents/' + id : '/api/admin/rooms/' + ROOM_ID + '/agents';

  fetch(url, {
    method: method,
    headers: { 'Content-Type': 'application/x-www-form-urlencoded' },
    body: body
  })
  .then(function(r) { return r.json(); })
  .then(function(data) {
    if (data.success) {
      closeModal('add-agent-modal');
      if (data.instructions) {
        showInstructionsText(data.instructions);
      } else {
        window.location.reload();
      }
    } else {
      showError('add-agent-error', data.message || t('error.save_agent'));
    }
  })
  .catch(function() {
    showError('add-agent-error', t('error.network'));
  });
}

function deleteAgent(id, name) {
  var msg = t('confirm.delete_agent').replace('%s', name);
  if (!confirm(msg)) return;
  fetch('/api/admin/agents/' + id, { method: 'DELETE' })
  .then(function(r) { return r.json(); })
  .then(function(data) {
    if (data.success) {
      window.location.reload();
    } else {
      alert(data.message || t('error.delete_agent'));
    }
  });
}

function viewInstructions(agentId) {
  fetch('/api/admin/agents/' + agentId + '/instructions')
  .then(function(r) { return r.json(); })
  .then(function(data) {
    if (data.success) {
      showInstructionsText(data.instructions);
    } else {
      alert(data.message || t('error.instructions'));
    }
  });
}

function showInstructionsText(text) {
  document.getElementById('instructions-text').value = text;
  openModal('instructions-modal');
  window._pendingReload = true;
}

// Reload page when instructions modal closes (to show new agent in table)
(function() {
  var modal = document.getElementById('instructions-modal');
  if (!modal) return;
  new MutationObserver(function() {
    if (!modal.classList.contains('open') && window._pendingReload) {
      window._pendingReload = false;
      window.location.reload();
    }
  }).observe(modal, { attributes: true, attributeFilter: ['class'] });
})();

function copyInstructions() {
  var text = document.getElementById('instructions-text').value;
  var btn = document.getElementById('copy-instructions-btn');
  if (navigator.clipboard) {
    navigator.clipboard.writeText(text).then(function() {
      btn.textContent = t('modal.instructions.copied');
      setTimeout(function() { btn.textContent = t('modal.instructions.copy'); }, 2000);
    });
  } else {
    document.getElementById('instructions-text').select();
    document.execCommand('copy');
  }
}

function downloadInstructions() {
  var text = document.getElementById('instructions-text').value;
  var blob = new Blob([text], { type: 'text/markdown' });
  var url = URL.createObjectURL(blob);
  var a = document.createElement('a');
  a.href = url;
  a.download = 'agent-room.md';
  a.click();
  URL.revokeObjectURL(url);
}

// ---- Room view: messages ----

function openNewMessageModal() {
  hideError('new-message-error');
  document.getElementById('new-message-form').reset();
  openModal('new-message-modal');
}

function submitNewMessage(e) {
  e.preventDefault();
  hideError('new-message-error');
  var form = document.getElementById('new-message-form');
  var body = new URLSearchParams(new FormData(form)).toString();
  fetch('/api/admin/rooms/' + ROOM_ID + '/messages', {
    method: 'POST',
    headers: { 'Content-Type': 'application/x-www-form-urlencoded' },
    body: body
  })
  .then(function(r) { return r.json(); })
  .then(function(data) {
    if (data.success) {
      closeModal('new-message-modal');
      window.location.reload();
    } else {
      showError('new-message-error', data.message || t('error.send_message'));
    }
  })
  .catch(function() {
    showError('new-message-error', t('error.network'));
  });
}

function openReplyModal(msgId, subject) {
  document.getElementById('reply-message-id').value = msgId;
  var ctx = document.getElementById('reply-context');
  if (ctx) ctx.textContent = (window.AR && window.AR.t && window.AR.t['modal.reply.context'] || 'Re:') + ' ' + subject;
  document.getElementById('reply-body').value = '';
  hideError('reply-error');
  openModal('reply-modal');
}

function submitReply(e) {
  e.preventDefault();
  hideError('reply-error');
  var msgId = document.getElementById('reply-message-id').value;
  var body = document.getElementById('reply-body').value;
  fetch('/api/admin/messages/' + msgId + '/reply', {
    method: 'POST',
    headers: { 'Content-Type': 'application/x-www-form-urlencoded' },
    body: 'message=' + encodeURIComponent(body)
  })
  .then(function(r) { return r.json(); })
  .then(function(data) {
    if (data.success) {
      closeModal('reply-modal');
      window.location.reload();
    } else {
      showError('reply-error', data.message || t('error.send_reply'));
    }
  })
  .catch(function() {
    showError('reply-error', t('error.network'));
  });
}

function closeThread(msgId) {
  if (!confirm(t('confirm.close_thread'))) return;
  fetch('/api/admin/messages/' + msgId + '/close', { method: 'POST' })
  .then(function(r) { return r.json(); })
  .then(function(data) {
    if (data.success) {
      window.location.reload();
    } else {
      alert(data.message || t('error.close_thread'));
    }
  });
}

const state = {
  accounts: [],
  messages: [],
};

function el(tag, attrs = {}, children = []) {
  const node = document.createElement(tag);
  for (const [key, value] of Object.entries(attrs)) {
    if (key === "class") node.className = value;
    else if (key === "text") node.textContent = value;
    else node.setAttribute(key, value);
  }
  for (const child of children) node.append(child);
  return node;
}

async function api(path, options = {}) {
  const url = new URL(path, window.location.origin);
  const response = await fetch(url, {
    ...options,
    credentials: "same-origin",
    headers: {
      "Content-Type": "application/json",
      ...(options.headers || {}),
    },
  });
  const payload = await response.json();
  if (!payload.ok) {
    const message = payload.error?.message || "Request failed";
    throw new Error(message);
  }
  return payload.data;
}

function renderError(target, error) {
  target.replaceChildren(el("p", { class: "error", text: error.message }));
}

async function loadSession() {
  const target = document.getElementById("session-status");
  try {
    const session = await api("/api/session");
    target.textContent = `${session.version || "dev"} · ${session.config_path || "default config"}`;
  } catch (error) {
    target.textContent = error.message;
    target.classList.add("error");
  }
}

async function loadAccounts() {
  const target = document.getElementById("accounts");
  try {
    const data = await api("/api/accounts");
    const accounts = data.accounts || [];
    state.accounts = accounts;
    target.replaceChildren(...accounts.map((account) => el("article", { class: "item" }, [
      el("h3", { text: account.name || "(unnamed)" }),
      el("p", { class: "meta", text: `${account.provider || account.driver || "unknown"} · ${account.mailbox || "INBOX"}` }),
      el("p", { class: "meta", text: account.username || "" }),
    ])));
    if (accounts.length === 0) target.replaceChildren(el("p", { class: "muted", text: "No accounts configured." }));
  } catch (error) {
    renderError(target, error);
  }
}

function renderMessages(messages) {
  const target = document.getElementById("messages");
  if (messages.length === 0) {
    target.replaceChildren(el("p", { class: "muted", text: "No indexed messages yet." }));
    return;
  }
  target.replaceChildren(...messages.map((message) => {
    const item = el("button", { class: "item message-item", type: "button" }, [
      el("span", { class: "message-title", text: message.subject || "(no subject)" }),
      el("span", { class: "meta", text: `${message.account || ""} · ${message.from || ""} · ${message.date || ""}` }),
      el("span", { class: "meta", text: message.snippet || "" }),
    ]);
    item.addEventListener("click", () => loadMessage(message.account, message.id));
    return item;
  }));
}

async function loadMessages() {
  const query = document.getElementById("search-query").value || "";
  const target = document.getElementById("messages");
  try {
    const url = query ? `/api/messages?q=${encodeURIComponent(query)}` : "/api/messages";
    const data = await api(url);
    state.messages = data.messages || [];
    renderMessages(state.messages);
  } catch (error) {
    renderError(target, error);
  }
}

async function loadMessage(account, id) {
  const target = document.getElementById("message-detail");
  try {
    const data = await api(`/api/messages/${encodeURIComponent(account)}/${encodeURIComponent(id)}`);
    const message = data.message?.message || data.message || {};
    target.classList.remove("muted");
    target.replaceChildren(
      el("h3", { text: message.meta?.subject || id }),
      el("p", { class: "meta", text: `${message.meta?.from?.address || ""} · ${message.meta?.date || ""}` }),
      el("p", { class: "preline", text: message.content?.body_md || message.content?.snippet || "" }),
    );
  } catch (error) {
    renderError(target, error);
  }
}

async function syncMail() {
  const target = document.getElementById("messages");
  try {
    await api("/api/sync", { method: "POST", body: JSON.stringify({ limit: 20 }) });
    await loadMessages();
  } catch (error) {
    renderError(target, error);
  }
}

async function loadOperations() {
  const target = document.getElementById("operations");
  try {
    const data = await api("/api/operations");
    const operations = data.operations || [];
    target.replaceChildren(...operations.map((operation) => {
      const children = [
        el("h3", { text: `${operation.operation || "operation"} · ${operation.status || "unknown"}` }),
        el("p", { class: "meta", text: operation.intent_id || operation.id || "" }),
      ];
      if (operation.status === "prepared" && operation.operation === "send" && operation.intent_id) {
        const confirm = el("button", { type: "button", text: "Confirm Send" });
        confirm.addEventListener("click", () => confirmOperation(operation.intent_id));
        children.push(confirm);
      }
      return el("article", { class: "item operation-item" }, children);
    }));
    if (operations.length === 0) target.replaceChildren(el("p", { class: "muted", text: "No prepared operations." }));
  } catch (error) {
    renderError(target, error);
  }
}

async function prepareSend() {
  const status = document.getElementById("compose-status");
  const account = state.accounts[0]?.name || "";
  const to = document.getElementById("compose-to").value.trim();
  const subject = document.getElementById("compose-subject").value.trim();
  const body = document.getElementById("compose-body").value.trim();
  if (!to || !subject || !body) {
    status.textContent = "To, subject, and body are required.";
    status.classList.add("error");
    return;
  }
  try {
    const result = await api("/api/send/prepare", {
      method: "POST",
      body: JSON.stringify({
        account,
        to: [{ address: to }],
        subject,
        body_text: body,
      }),
    });
    status.classList.remove("error");
    status.textContent = `Prepared ${result.intent_id}`;
    await loadOperations();
  } catch (error) {
    status.classList.add("error");
    status.textContent = error.message;
  }
}

async function confirmOperation(intentId) {
  const target = document.getElementById("operations");
  try {
    await api(`/api/operations/${encodeURIComponent(intentId)}/confirm`, { method: "POST" });
    await loadOperations();
  } catch (error) {
    renderError(target, error);
  }
}

document.getElementById("refresh-accounts").addEventListener("click", loadAccounts);
document.getElementById("search-mail").addEventListener("click", loadMessages);
document.getElementById("sync-mail").addEventListener("click", syncMail);
document.getElementById("refresh-operations").addEventListener("click", loadOperations);
document.getElementById("prepare-send").addEventListener("click", prepareSend);

loadSession();
loadAccounts();
loadMessages();
loadOperations();

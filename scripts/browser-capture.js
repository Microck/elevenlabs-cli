// Run in the ElevenLabs tab before exercising a workflow. Export __calls before
// a full navigation. Captures API calls, never authorization headers or audio.
(() => {
  if (window.__captureInstalled) return 'already installed';
  window.__captureInstalled = true;
  window.__calls = [];
  // Export completed calls periodically without retaining earlier responses.
  window.__drainCalls = () => window.__calls.splice(0);
  const sensitive = /token|password|secret|authorization|cookie|email|session|api.?key|jwt|captcha|^key$|verification_code|distinct_id/i;
  function scrub(value) {
    if (Array.isArray(value)) return value.map(scrub);
    if (value && typeof value === 'object') {
      return Object.fromEntries(Object.entries(value).map(([key, item]) =>
        [key, sensitive.test(key) ? '[REDACTED]' : scrub(item)]));
    }
    if (typeof value === 'string') {
      return value.replace(/[\w.+-]+@[\w.-]+\.[a-z]+/gi, '[EMAIL]')
        .replace(/eyJ[A-Za-z0-9_-]+\.[A-Za-z0-9_-]+\.[A-Za-z0-9_-]+/g, '[JWT]');
    }
    return value;
  }
  function body(value) {
    if (value == null) return null;
    if (value instanceof FormData) return [...value.entries()].map(([key, item]) =>
      ({key, value: sensitive.test(key) ? '[REDACTED]' : item instanceof File
        ? {name: item.name, type: item.type, size: item.size} : scrub(item)}));
    try { return scrub(JSON.parse(value)); }
    catch { return '[non-JSON body]'; }
  }
  function entry(url, method, payload) {
    const parsed = new URL(url, location.href);
    if (!/(^|\.)elevenlabs\.io$|(^|\.)googleapis\.com$/.test(parsed.hostname)) return null;
    return {time: new Date().toISOString(), method, url: parsed.origin + parsed.pathname,
      query: Object.fromEntries([...parsed.searchParams].map(([key, value]) =>
        [key, sensitive.test(key) ? '[REDACTED]' : scrub(value)])), request: body(payload)};
  }
  const originalFetch = window.fetch;
  window.fetch = async function(input, init) {
    const call = entry(input instanceof Request ? input.url : String(input),
      init?.method || input?.method || 'GET', init?.body);
    const requestBody = call && input instanceof Request && init?.body === undefined
      ? input.clone().text().then(value => { call.request = body(value); }, () => {
          call.request = '[unreadable Request body]';
        })
      : Promise.resolve();
    try {
      const response = await originalFetch.apply(this, arguments);
      if (call) {
        call.status = response.status;
        call.contentType = response.headers.get('content-type') || '';
        // Keep streaming responses streaming. Read only cloned JSON responses.
        if (call.contentType.includes('json')) {
          response.clone().json().then(value => {
            call.response = scrub(value);
          }, () => {
            call.response = '[unreadable JSON]';
          }).then(() => requestBody).finally(() => window.__calls.push(call));
        } else {
          call.response = '[binary or text]';
          requestBody.finally(() => window.__calls.push(call));
        }
      }
      return response;
    } catch (error) {
      if (call) { call.error = String(error); window.__calls.push(call); }
      throw error;
    }
  };
  const originalOpen = XMLHttpRequest.prototype.open;
  const originalSend = XMLHttpRequest.prototype.send;
  XMLHttpRequest.prototype.open = function(method, url) {
    this.__captureMeta = {method, url};
    return originalOpen.apply(this, arguments);
  };
  XMLHttpRequest.prototype.send = function(payload) {
    const call = this.__captureMeta && entry(this.__captureMeta.url, this.__captureMeta.method, payload);
    if (call) this.addEventListener('loadend', () => {
      call.status = this.status;
      call.contentType = this.getResponseHeader('content-type') || '';
      call.response = call.contentType.includes('json')
        ? this.responseType === 'json' ? scrub(this.response) : body(this.responseText)
        : '[binary or text]';
      window.__calls.push(call);
    }, {once: true});
    return originalSend.apply(this, arguments);
  };
  return 'API capture installed';
})();

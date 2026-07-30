const BASE_API = '/api';

function isFormDataBody(body) {
  return typeof FormData !== 'undefined' && !!body && (body instanceof FormData || body.constructor?.name === 'FormData');
}

function buildHeaders(customHeaders = {}, body = null) {
  const token = localStorage.getItem('token');
  const headers = {
    ...(token ? { Authorization: `Bearer ${token}` } : {}),
    ...customHeaders
  };

  if (!isFormDataBody(body)) {
    headers['Content-Type'] = 'application/json';
  }

  return headers;
}

export async function apiFetch(path, options = {}) {
  const response = await fetch(`${BASE_API}${path}`, {
    ...options,
    headers: buildHeaders(options.headers, options.body),
  });

  return response;
}

export async function getJSON(path) {
  const response = await apiFetch(path, { method: 'GET' });
  if (!response.ok) {
    const text = await response.text();
    throw new Error(text || 'Error en la petición');
  }
  return response.json();
}

export async function postJSON(path, body) {
  const response = await apiFetch(path, { method: 'POST', body: JSON.stringify(body) });
  if (!response.ok) {
    const payload = await response.json().catch(() => ({}));
    const err = new Error(payload.message || 'Error en la petición POST');
    try { err.code = payload.code } catch (e) {}
    err.status = response.status;
    throw err;
  }
  return response.json();
}

export async function putJSON(path, body) {
  const response = await apiFetch(path, { method: 'PUT', body: JSON.stringify(body) });
  if (!response.ok) {
    const payload = await response.json().catch(() => ({}));
    const err = new Error(payload.message || 'Error en la petición PUT');
    try { err.code = payload.code } catch (e) {}
    err.status = response.status;
    throw err;
  }
  return response.json();
}

export async function deleteJSON(path) {
  const response = await apiFetch(path, { method: 'DELETE' });
  if (!response.ok) {
    const payload = await response.json().catch(() => ({}));
    const err = new Error(payload.message || 'Error en la petición DELETE');
    try { err.code = payload.code } catch (e) {}
    err.status = response.status;
    throw err;
  }
  if (response.status === 204) {
    return null;
  }
  const text = await response.text();
  return text ? JSON.parse(text) : null;
}

const BASE_API = '/api';

function buildHeaders(customHeaders = {}) {
  const token = localStorage.getItem('token');
  return {
    'Content-Type': 'application/json',
    ...(token ? { Authorization: `Bearer ${token}` } : {}),
    ...customHeaders
  };
}

export async function apiFetch(path, options = {}) {
  const response = await fetch(`${BASE_API}${path}`, {
    headers: buildHeaders(options.headers),
    ...options
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
    throw new Error(payload.message || 'Error en la petición POST');
  }
  return response.json();
}

export async function putJSON(path, body) {
  const response = await apiFetch(path, { method: 'PUT', body: JSON.stringify(body) });
  if (!response.ok) {
    const payload = await response.json().catch(() => ({}));
    throw new Error(payload.message || 'Error en la petición PUT');
  }
  return response.json();
}

export async function deleteJSON(path) {
  const response = await apiFetch(path, { method: 'DELETE' });
  if (!response.ok) {
    const payload = await response.json().catch(() => ({}));
    throw new Error(payload.message || 'Error en la petición DELETE');
  }
  if (response.status === 204) {
    return null;
  }
  const text = await response.text();
  return text ? JSON.parse(text) : null;
}

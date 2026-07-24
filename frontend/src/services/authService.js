import { apiFetch } from './api.js';

export async function login(credentials) {
  const response = await apiFetch('/auth/login', {
    method: 'POST',
    body: JSON.stringify(credentials)
  });

  if (!response.ok) {
    const payload = await response.json().catch(() => ({ message: response.statusText }));
    throw new Error(payload.message || 'No se pudo iniciar sesión');
  }

  return response.json();
}

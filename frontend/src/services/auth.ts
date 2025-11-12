const apiBaseEnv = import.meta.env.VITE_API_BASE_URL ?? '/api';
const API_BASE = apiBaseEnv.replace(/\/$/, '');

type AuthResponse = {
  user: {
    id: string;
    login: string;
  };
  tokens: {
    access_token: string;
    refresh_token: string;
  };
};

async function request<T>(path: string, options: RequestInit): Promise<T> {
  const response = await fetch(`${API_BASE}${path}`, {
    ...options,
    headers: {
      'Content-Type': 'application/json',
      ...(options.headers ?? {}),
    },
  });

  if (!response.ok) {
    const data = await response.json().catch(() => null);
    const message = data?.error ?? 'Произошла ошибка. Попробуйте еще раз.';
    throw new Error(message);
  }

  return (await response.json()) as T;
}

export async function register(login: string, password: string): Promise<AuthResponse> {
  return request<AuthResponse>('/auth/register', {
    method: 'POST',
    body: JSON.stringify({ login, password }),
  });
}

export async function login(login: string, password: string): Promise<AuthResponse> {
  return request<AuthResponse>('/auth/login', {
    method: 'POST',
    body: JSON.stringify({ login, password }),
  });
}


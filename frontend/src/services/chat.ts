const API_BASE = import.meta.env.VITE_API_BASE_URL || '/api';

async function request<T>(path: string, options: RequestInit): Promise<T> {
  const tokens = localStorage.getItem('tt_tokens');
  const token = tokens ? JSON.parse(tokens).access_token : null;

  const response = await fetch(`${API_BASE}${path}`, {
    ...options,
    headers: {
      'Content-Type': 'application/json',
      ...(token ? { Authorization: `Bearer ${token}` } : {}),
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

export interface StartChatResponse {
  session_id: string;
}

export async function startChat(userId: string): Promise<StartChatResponse> {
  try {
    const response = await request<StartChatResponse>('/chat/start', {
      method: 'POST',
      body: JSON.stringify({ user_id: userId }),
    });
    console.log('startChat response:', response);
    return response;
  } catch (error) {
    console.error('startChat error:', error);
    throw error;
  }
}

export async function sendMessage(sessionId: string, userId: string, content: string): Promise<void> {
  return request<void>('/chat/message', {
    method: 'POST',
    body: JSON.stringify({
      session_id: sessionId,
      user_id: userId,
      content: content,
    }),
  });
}

export interface QuestionResponse {
  question: string;
  question_id: string;
  timestamp: string;
}

export async function getNextQuestion(sessionId: string): Promise<QuestionResponse | null> {
  try {
    return await request<QuestionResponse>(`/chat/${sessionId}/question`, {
      method: 'GET',
    });
  } catch (error: any) {
    if (error.message?.includes('no new questions') || error.message?.includes('404')) {
      return null;
    }
    throw error;
  }
}

export interface ResultsResponse {
  score: number;
  feedback: string;
  recommendations: string[];
  completed_at: string;
}

export async function getResults(sessionId: string): Promise<ResultsResponse | null> {
  try {
    return await request<ResultsResponse>(`/chat/${sessionId}/results`, {
      method: 'GET',
    });
  } catch (error: any) {
    if (error.message?.includes('not completed') || error.message?.includes('404')) {
      return null;
    }
    throw error;
  }
}

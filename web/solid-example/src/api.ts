export const API_BASE_URL = "http://localhost:8080";
export const WS_BASE_URL = "ws://localhost:8080/ws";

export interface User {
  id: string;
  email: string;
  name: string;
}

export const fetchCurrentUser = async (): Promise<User | null> => {
  try {
    const response = await fetch(`${API_BASE_URL}/v1/auth/me`, {
      credentials: "include",
    });
    if (!response.ok) return null;
    return response.json();
  } catch (err) {
    console.error("Auth check failed:", err);
    return null;
  }
};

export const setupEventsWebSocket = (onMessage: (data: any) => void) => {
  const ws = new WebSocket(WS_BASE_URL);

  ws.onopen = () => console.log("Connected to Modulith WebSocket");
  ws.onmessage = (event) => {
    try {
      const data = JSON.parse(event.data);
      onMessage(data);
    } catch (err) {
      console.error("Failed to parse WS message:", err);
    }
  };

  ws.onclose = () => console.log("Disconnected from Modulith WebSocket");
  ws.onerror = (err) => console.error("WebSocket error:", err);

  return ws;
};

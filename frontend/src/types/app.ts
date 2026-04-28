export type AppPage = "auth" | "join" | "call";

export interface Participant {
  id: string;
  name: string;
  isSpeaking: boolean;
  isMuted: boolean;
}

export interface AppState {
  isAuthenticated: boolean;
  currentPage: AppPage;
  currentSessionId: string;
  isMuted: boolean;
  isLoading: boolean;
  isBootstrapped: boolean;
  authError: string | null;
  authInfo: string | null;
  joinError: string | null;
  participants: Participant[];
}

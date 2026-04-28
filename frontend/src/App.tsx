import { useEffect, useState } from "react";
import "./App.css";
import {
  Bootstrap,
  CreateSession,
  JoinSession,
  LeaveSession,
  Login,
  Register,
} from "../wailsjs/go/main/App";
import { CallPage } from "./pages/CallPage";
import { AuthPage } from "./pages/AuthPage";
import { JoinPage } from "./pages/JoinPage";
import type { AppState, Participant } from "./types/app";

const INITIAL_STATE: AppState = {
  isAuthenticated: false,
  currentPage: "auth",
  currentSessionId: "",
  isMuted: false,
  isLoading: true,
  isBootstrapped: false,
  authError: null,
  authInfo: null,
  joinError: null,
  participants: [],
};

const buildParticipants = (isMuted: boolean): Participant[] => [
  {
    id: "local-user",
    name: "Вы",
    isSpeaking: false,
    isMuted,
  },
];

const resolveErrorMessage = (error: unknown): string => {
  if (error instanceof Error && error.message) {
    return error.message;
  }

  if (typeof error === "string" && error.trim()) {
    return error;
  }

  return "Произошла ошибка";
};

const resolveBootstrapValue = (
  result: Partial<{ isAuthenticated: boolean; IsAuthenticated: boolean; authError: string; AuthError: string; authInfo: string; AuthInfo: string }>,
) => ({
  isAuthenticated: result.isAuthenticated ?? result.IsAuthenticated ?? false,
  authError: result.authError ?? result.AuthError ?? null,
  authInfo: result.authInfo ?? result.AuthInfo ?? null,
});

const resolveSessionID = (result: Partial<{ id: string; ID: string }>) => result.id ?? result.ID ?? "";

function App() {
  const [appState, setAppState] = useState<AppState>(INITIAL_STATE);

  useEffect(() => {
    let cancelled = false;

    const bootstrap = async () => {
      try {
        const result = await Bootstrap();
        const bootstrapState = resolveBootstrapValue(result);
        if (cancelled) {
          return;
        }

        setAppState((current) => ({
          ...current,
          isAuthenticated: bootstrapState.isAuthenticated,
          currentPage: bootstrapState.isAuthenticated ? "join" : "auth",
          authError: bootstrapState.authError,
          authInfo: bootstrapState.authInfo,
          isLoading: false,
          isBootstrapped: true,
        }));
      } catch (error) {
        if (cancelled) {
          return;
        }

        setAppState((current) => ({
          ...current,
          currentPage: "auth",
          authError: resolveErrorMessage(error),
          authInfo: null,
          isLoading: false,
          isBootstrapped: true,
        }));
      }
    };

    void bootstrap();

    return () => {
      cancelled = true;
    };
  }, []);

  const onLogin = async ({ email, password }: { email: string; password: string }) => {
    const normalizedEmail = email.trim();

    if (!normalizedEmail || password.trim().length < 6) {
      setAppState((current) => ({
        ...current,
        authError: "Укажи корректный email и пароль",
        authInfo: null,
      }));
      return;
    }

    setAppState((current) => ({ ...current, isLoading: true, authError: null, authInfo: null }));

    try {
      await Login(normalizedEmail, password);
      setAppState((current) => ({
        ...current,
        isAuthenticated: true,
        currentPage: "join",
        authError: null,
        authInfo: `Вход выполнен: ${normalizedEmail}`,
        joinError: null,
        isLoading: false,
      }));
    } catch (error) {
      setAppState((current) => ({
        ...current,
        authError: resolveErrorMessage(error),
        authInfo: null,
        isLoading: false,
      }));
    }
  };

  const onRegister = async ({
    username,
    email,
    password,
  }: {
    username: string;
    email: string;
    password: string;
  }) => {
    const normalizedUsername = username.trim();
    const normalizedEmail = email.trim();

    if (!normalizedUsername || !normalizedEmail || password.trim().length < 6) {
      setAppState((current) => ({
        ...current,
        authError: "Для регистрации нужны username, email и пароль",
        authInfo: null,
      }));
      return;
    }

    setAppState((current) => ({ ...current, isLoading: true, authError: null, authInfo: null }));

    try {
      await Register(normalizedUsername, normalizedEmail, password);
      setAppState((current) => ({
        ...current,
        isAuthenticated: true,
        currentPage: "join",
        authError: null,
        authInfo: `Аккаунт создан: ${normalizedUsername}`,
        joinError: null,
        isLoading: false,
      }));
    } catch (error) {
      setAppState((current) => ({
        ...current,
        authError: resolveErrorMessage(error),
        authInfo: null,
        isLoading: false,
      }));
    }
  };

  const onJoinSession = async (sessionId: string) => {
    const normalizedSessionId = sessionId.trim().toUpperCase();

    if (!normalizedSessionId) {
      setAppState((current) => ({
        ...current,
        joinError: "Введите корректный ID сессии",
      }));
      return;
    }

    setAppState((current) => ({ ...current, isLoading: true, joinError: null }));

    try {
      const session = await JoinSession(normalizedSessionId);
      const sessionID = resolveSessionID(session);
      if (!sessionID) {
        throw new Error("Backend не вернул ID сессии");
      }
      setAppState((current) => ({
        ...current,
        currentPage: "call",
        currentSessionId: sessionID,
        joinError: null,
        participants: buildParticipants(current.isMuted),
        isLoading: false,
      }));
    } catch (error) {
      setAppState((current) => ({
        ...current,
        joinError: resolveErrorMessage(error),
        isLoading: false,
      }));
    }
  };

  const onCreateSession = async () => {
    setAppState((current) => ({ ...current, isLoading: true, joinError: null }));

    try {
      const session = await CreateSession();
      const sessionID = resolveSessionID(session);
      if (!sessionID) {
        throw new Error("Backend не вернул ID сессии");
      }
      setAppState((current) => ({
        ...current,
        currentPage: "call",
        currentSessionId: sessionID,
        joinError: null,
        participants: buildParticipants(current.isMuted),
        isLoading: false,
      }));
    } catch (error) {
      setAppState((current) => ({
        ...current,
        joinError: resolveErrorMessage(error),
        isLoading: false,
      }));
    }
  };

  const onToggleMute = () => {
    setAppState((current) => ({
      ...current,
      isMuted: !current.isMuted,
      participants: current.participants.map((participant) =>
        participant.id === "local-user"
          ? { ...participant, isMuted: !current.isMuted }
          : participant,
      ),
    }));
  };

  const onLeaveCall = async () => {
    setAppState((current) => ({ ...current, isLoading: true }));

    try {
      await LeaveSession();
      setAppState((current) => ({
        ...current,
        currentPage: "join",
        currentSessionId: "",
        isMuted: false,
        participants: [],
        joinError: null,
        isLoading: false,
      }));
    } catch (error) {
      setAppState((current) => ({
        ...current,
        joinError: resolveErrorMessage(error),
        isLoading: false,
      }));
    }
  };

  const onBackToAuth = () => {
    setAppState((current) => ({
      ...current,
        currentPage: "auth",
        isAuthenticated: false,
        authError: null,
        joinError: null,
        authInfo: null,
        currentSessionId: "",
        participants: [],
      }));
  };

  return (
    <div className="app-shell">
      <div className="background-orb background-orb--one" />
      <div className="background-orb background-orb--two" />

      <div className={`screen screen--${appState.currentPage}`}>
        {appState.isBootstrapped && appState.currentPage === "auth" ? (
          <AuthPage
            error={appState.authError}
            info={appState.authInfo}
            isLoading={appState.isLoading}
            onLogin={onLogin}
            onRegister={onRegister}
          />
        ) : null}

        {appState.isBootstrapped && appState.currentPage === "join" ? (
          <JoinPage
            error={appState.joinError}
            isLoading={appState.isLoading}
            onJoinSession={onJoinSession}
            onCreateSession={onCreateSession}
            onBack={onBackToAuth}
          />
        ) : null}

        {appState.isBootstrapped && appState.currentPage === "call" ? (
          <CallPage
            sessionId={appState.currentSessionId}
            participants={appState.participants}
            isMuted={appState.isMuted}
            isLoading={appState.isLoading}
            onToggleMute={onToggleMute}
            onLeaveCall={onLeaveCall}
          />
        ) : null}
      </div>
    </div>
  );
}

export default App;

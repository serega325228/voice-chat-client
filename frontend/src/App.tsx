import { useEffect, useState } from "react";
import "./App.css";
import {
  Bootstrap,
  CreateSession,
  JoinSession,
  LeaveSession,
  Login,
  Register,
  SetMicrophoneMuted,
} from "../wailsjs/go/main/App";
import { EventsOn } from "../wailsjs/runtime/runtime";
import { CallPage } from "./pages/CallPage";
import { AuthPage } from "./pages/AuthPage";
import { JoinPage } from "./pages/JoinPage";
import type { AppState, Participant } from "./types/app";

const INITIAL_STATE: AppState = {
  isAuthenticated: false,
  currentPage: "auth",
  currentSessionId: "",
  callStatus: "idle",
  callInfo: null,
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

const resolveBootstrapValue = (result: Partial<{ isAuthenticated: boolean; authError?: string; authInfo?: string }>) => ({
  isAuthenticated: result.isAuthenticated ?? false,
  authError: result.authError ?? null,
  authInfo: result.authInfo ?? null,
});

const resolveCallState = (
  result: Partial<{ sessionId: string; isMuted: boolean; status: "idle" | "active"; message?: string }>,
) => ({
  sessionId: result.sessionId ?? "",
  isMuted: result.isMuted ?? false,
  status: result.status ?? "idle",
  message: result.message ?? null,
});

const resolveCallStatusLabel = (status: "idle" | "active") =>
  status === "active" ? "Активен" : "Не в звонке";

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

  useEffect(() => {
    return EventsOn(
      "call:session",
      (event: Partial<{ kind: string; sessionId?: string; isMuted?: boolean; message?: string }>) => {
        if (!event.kind) {
          return;
        }

        switch (event.kind) {
          case "started":
            setAppState((current) => {
              const isMuted = event.isMuted ?? current.isMuted;

              return {
                ...current,
                currentPage: "call",
                currentSessionId: event.sessionId ?? current.currentSessionId,
                callStatus: "active",
                callInfo: event.message ?? current.callInfo,
                isMuted,
                participants: buildParticipants(isMuted),
              };
            });
            return;
          case "mute_changed":
            setAppState((current) => {
              const isMuted = event.isMuted ?? current.isMuted;

              return {
                ...current,
                isMuted,
                callInfo: event.message ?? current.callInfo,
                participants: buildParticipants(isMuted),
              };
            });
            return;
          case "failed":
            setAppState((current) => ({
              ...current,
              currentPage: "join",
              currentSessionId: "",
              callStatus: "idle",
              callInfo: null,
              isMuted: false,
              participants: [],
              joinError: event.message ?? "Сессия завершилась с ошибкой",
              isLoading: false,
            }));
            return;
          case "left":
            setAppState((current) => ({
              ...current,
              currentPage: "join",
              currentSessionId: "",
              callStatus: "idle",
              callInfo: null,
              isMuted: false,
              participants: [],
              joinError: null,
              isLoading: false,
            }));
            return;
        }
      },
    );
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
    const normalizedSessionId = sessionId.trim();

    if (!normalizedSessionId) {
      setAppState((current) => ({
        ...current,
        joinError: "Введите корректный ID сессии",
      }));
      return;
    }

    setAppState((current) => ({ ...current, isLoading: true, joinError: null }));

    try {
      const callState = resolveCallState(await JoinSession(normalizedSessionId));
      if (!callState.sessionId) {
        throw new Error("Backend не вернул ID сессии");
      }
      setAppState((current) => ({
        ...current,
        currentPage: "call",
        currentSessionId: callState.sessionId,
        callStatus: callState.status,
        callInfo: callState.message,
        isMuted: callState.isMuted,
        joinError: null,
        participants: buildParticipants(callState.isMuted),
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
      const callState = resolveCallState(await CreateSession());
      if (!callState.sessionId) {
        throw new Error("Backend не вернул ID сессии");
      }
      setAppState((current) => ({
        ...current,
        currentPage: "call",
        currentSessionId: callState.sessionId,
        callStatus: callState.status,
        callInfo: callState.message,
        isMuted: callState.isMuted,
        joinError: null,
        participants: buildParticipants(callState.isMuted),
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

  const onToggleMute = async () => {
    const nextMuted = !appState.isMuted;

    setAppState((current) => ({ ...current, isLoading: true }));

    try {
      const callState = resolveCallState(await SetMicrophoneMuted(nextMuted));
      setAppState((current) => ({
        ...current,
        isMuted: callState.isMuted,
        callInfo: callState.message,
        participants: buildParticipants(callState.isMuted),
        isLoading: false,
      }));
    } catch (error) {
      setAppState((current) => ({
        ...current,
        callInfo: resolveErrorMessage(error),
        isLoading: false,
      }));
    }
  };

  const onLeaveCall = async () => {
    setAppState((current) => ({ ...current, isLoading: true }));

    try {
      const callState = resolveCallState(await LeaveSession());
      setAppState((current) => ({
        ...current,
        currentPage: "join",
        currentSessionId: "",
        callStatus: callState.status,
        callInfo: callState.message,
        isMuted: callState.isMuted,
        participants: [],
        joinError: null,
        isLoading: false,
      }));
    } catch (error) {
      setAppState((current) => ({
        ...current,
        callInfo: resolveErrorMessage(error),
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
      callStatus: "idle",
      callInfo: null,
      isMuted: false,
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
            callStatus={resolveCallStatusLabel(appState.callStatus)}
            callInfo={appState.callInfo}
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

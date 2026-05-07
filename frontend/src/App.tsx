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
import type { AppState, AppTheme, CallStatus, Participant } from "./types/app";

interface ThemeOption {
  value: AppTheme;
  label: string;
  description: string;
}

const DEFAULT_THEME: AppTheme = "midnight";
const THEME_STORAGE_KEY = "selfcord.theme";
const THEME_OPTIONS: ThemeOption[] = [
  {
    value: "midnight",
    label: "Midnight",
    description: "Темная холодная палитра для вечерних сессий",
  },
  {
    value: "paper",
    label: "Paper",
    description: "Светлая спокойная палитра с высокой читаемостью",
  },
  {
    value: "forest",
    label: "Forest",
    description: "Темная зеленая палитра с мягкими акцентами",
  },
];

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
    name: "Локальный участник",
    isSpeaking: false,
    isMuted,
  },
];

const isAppTheme = (value: string | null): value is AppTheme =>
  THEME_OPTIONS.some((theme) => theme.value === value);

const readInitialTheme = (): AppTheme => {
  if (typeof window === "undefined") {
    return DEFAULT_THEME;
  }

  try {
    const storedTheme = window.localStorage.getItem(THEME_STORAGE_KEY);
    return isAppTheme(storedTheme) ? storedTheme : DEFAULT_THEME;
  } catch {
    return DEFAULT_THEME;
  }
};

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
  result: Partial<{ sessionId: string; isMuted: boolean; status: string; message?: string }>,
): { sessionId: string; isMuted: boolean; status: CallStatus; message: string | null } => ({
  sessionId: result.sessionId ?? "",
  isMuted: result.isMuted ?? false,
  status: result.status === "active" ? "active" : "idle",
  message: result.message ?? null,
});

const resolveCallStatusLabel = (status: "idle" | "active") =>
  status === "active" ? "Активен" : "Не в звонке";

interface ThemeSwitcherProps {
  theme: AppTheme;
  onThemeChange: (theme: AppTheme) => void;
}

function ThemeSwitcher({ theme, onThemeChange }: ThemeSwitcherProps) {
  return (
    <div className="theme-switcher" aria-label="Выбор темы">
      <span className="theme-switcher__label">Тема</span>
      <div className="theme-switcher__options">
        {THEME_OPTIONS.map((option) => (
          <button
            key={option.value}
            type="button"
            className="theme-chip"
            aria-pressed={theme === option.value}
            title={option.description}
            onClick={() => onThemeChange(option.value)}
          >
            <span className={`theme-chip__swatch theme-chip__swatch--${option.value}`} />
            <span>{option.label}</span>
          </button>
        ))}
      </div>
    </div>
  );
}

function App() {
  const [appState, setAppState] = useState<AppState>(INITIAL_STATE);
  const [theme, setTheme] = useState<AppTheme>(readInitialTheme);

  useEffect(() => {
    document.documentElement.dataset.theme = theme;

    try {
      window.localStorage.setItem(THEME_STORAGE_KEY, theme);
    } catch {
      // The selected theme is cosmetic; storage failures should not block the client.
    }
  }, [theme]);

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

    if (!normalizedEmail || password.trim().length < 8) {
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
        authInfo: "Вход выполнен",
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

    if (!normalizedUsername || !normalizedEmail || password.trim().length < 8) {
      setAppState((current) => ({
        ...current,
        authError: "Для регистрации нужны никнейм, email и пароль",
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
        authInfo: "Аккаунт создан",
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
    <div className="app-shell" data-theme={theme}>
      <div className="background-grid" />
      <div className="background-orb background-orb--one" />
      <div className="background-orb background-orb--two" />

      <header className="app-topbar">
        <div className="brand-lockup" aria-label="selfcord">
          <span className="brand-mark">sc</span>
          <span>
            <strong>selfcord</strong>
            <small>voice sessions</small>
          </span>
        </div>
        <ThemeSwitcher theme={theme} onThemeChange={setTheme} />
      </header>

      <div className={`screen screen--${appState.currentPage}`}>
        {!appState.isBootstrapped ? (
          <section className="panel panel--join">
            <div className="panel__card panel__card--wide panel__card--center">
              <span className="eyebrow">selfcord</span>
              <h1>Загрузка клиента</h1>
              <p>Проверяем локальную сессию и готовим интерфейс.</p>
            </div>
          </section>
        ) : null}

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

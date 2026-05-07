import { useState, type FormEvent } from "react";

interface AuthPageProps {
  error: string | null;
  info: string | null;
  isLoading: boolean;
  onLogin: (payload: { email: string; password: string }) => void;
  onRegister: (payload: { username: string; email: string; password: string }) => void;
}

export function AuthPage({
  error,
  info,
  isLoading,
  onLogin,
  onRegister,
}: AuthPageProps) {
  const [username, setUsername] = useState("");
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");

  const handleLogin = (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    onLogin({ email, password });
  };

  const handleRegister = () => {
    onRegister({ username, email, password });
  };

  return (
    <section className="panel panel--auth">
      <div className="panel__hero">
        <span className="eyebrow">selfcord</span>
        <h1>Голосовые комнаты по ID сессии</h1>
        <p>
          Авторизация, создание комнаты и подключение к существующей сессии
          остаются в одном компактном клиенте.
        </p>
        <div className="hero-metrics">
          <div>
            <strong>Auth</strong>
            <span>Email-вход и регистрация</span>
          </div>
          <div>
            <strong>Rooms</strong>
            <span>Создание и вход по UUID</span>
          </div>
        </div>
      </div>

      <div className="panel__card">
        <div className="card-header">
          <span className="eyebrow">Авторизация</span>
          <h2>Войти в selfcord</h2>
          <p>Для входа используется email, для регистрации нужны никнейм, email и пароль.</p>
        </div>

        <form className="stack" onSubmit={handleLogin}>
          <label className="field">
            <span>Никнейм</span>
            <input
              type="text"
              value={username}
              onChange={(event) => setUsername(event.target.value)}
              placeholder="nova"
              autoComplete="nickname"
            />
          </label>

          <label className="field">
            <span>Email</span>
            <input
              type="email"
              value={email}
              onChange={(event) => setEmail(event.target.value)}
              placeholder="name@example.com"
              autoComplete="username"
            />
          </label>

          <label className="field">
            <span>Пароль</span>
            <input
              type="password"
              value={password}
              onChange={(event) => setPassword(event.target.value)}
              placeholder="Минимум 8 символов"
              autoComplete="current-password"
            />
          </label>

          <div className="form-feedback" aria-live="polite">
            {error ? <span className="form-feedback__error">{error}</span> : null}
            {!error && info ? <span className="form-feedback__info">{info}</span> : null}
            {!error && !info ? (
              <span className="form-feedback__hint">
                Войди в существующий аккаунт или создай новый профиль.
              </span>
            ) : null}
          </div>

          <div className="actions">
            <button type="submit" className="button button--primary" disabled={isLoading}>
              {isLoading ? "Выполняется вход..." : "Войти"}
            </button>
            <button
              type="button"
              className="button button--ghost"
              onClick={handleRegister}
              disabled={isLoading}
            >
              Зарегистрироваться
            </button>
          </div>
        </form>
      </div>
    </section>
  );
}

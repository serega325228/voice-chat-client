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
        <span className="eyebrow">Voice Chat Client</span>
        <h1>Вход в голосовые комнаты</h1>
        <p>
          Авторизуйся, создай комнату или подключись к существующей сессии.
        </p>
        <div className="hero-metrics">
          <div>
            <strong>Wails</strong>
            <span>Go и React в одном клиенте</span>
          </div>
          <div>
            <strong>Сессии</strong>
            <span>Вход, создание и подключение</span>
          </div>
        </div>
      </div>

      <div className="panel__card">
        <div className="card-header">
          <span className="eyebrow">Авторизация</span>
          <h2>Вход и регистрация</h2>
          <p>Для входа используется email, для регистрации нужны username, email и пароль.</p>
        </div>

        <form className="stack" onSubmit={handleLogin}>
          <label className="field">
            <span>Username</span>
            <input
              type="text"
              value={username}
              onChange={(event) => setUsername(event.target.value)}
              placeholder="sereg"
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
              placeholder="Минимум 6 символов"
              autoComplete="current-password"
            />
          </label>

          <div className="form-feedback" aria-live="polite">
            {error ? <span className="form-feedback__error">{error}</span> : null}
            {!error && info ? <span className="form-feedback__info">{info}</span> : null}
            {!error && !info ? (
              <span className="form-feedback__hint">
                Войди в существующий аккаунт или зарегистрируй новый.
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

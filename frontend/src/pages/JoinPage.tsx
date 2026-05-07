import { useState, type FormEvent } from "react";

interface JoinPageProps {
  isLoading: boolean;
  error: string | null;
  onJoinSession: (sessionId: string) => void;
  onCreateSession: () => void;
  onBack: () => void;
}

export function JoinPage({
  isLoading,
  error,
  onJoinSession,
  onCreateSession,
  onBack,
}: JoinPageProps) {
  const [sessionId, setSessionId] = useState("");

  const handleSubmit = (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    onJoinSession(sessionId);
  };

  return (
    <section className="workspace workspace--join">
      <aside className="workspace-sidebar" aria-label="Навигация selfcord">
        <div className="server-rail" aria-hidden="true">
          <span className="server-dot server-dot--active">sc</span>
          <span className="server-dot">+</span>
        </div>
        <div className="channel-panel">
          <span className="sidebar-label">Selfcord</span>
          <h3>Голосовые каналы</h3>
          <div className="channel-list">
            <button type="button" className="channel-item channel-item--active">
              # lobby
            </button>
            <button type="button" className="channel-item" disabled>
              # active-session
            </button>
            <button type="button" className="channel-item" disabled>
              # recent
            </button>
          </div>
        </div>
      </aside>

      <main className="workspace-main">
        <div className="panel__card panel__card--wide connect-card">
          <div className="card-header">
            <span className="eyebrow">Комнаты</span>
            <h2>Подключиться к voice-сессии</h2>
            <p>Вставь ID существующей комнаты или создай новую сессию.</p>
          </div>

          <form className="stack" onSubmit={handleSubmit}>
            <label className="field">
              <span>ID комнаты</span>
              <input
                type="text"
                value={sessionId}
                onChange={(event) => setSessionId(event.target.value)}
                placeholder="9d969273-a5ef-4f2e-89f4-61457cb3a889"
                autoComplete="off"
              />
            </label>

            <div className="form-feedback" aria-live="polite">
              {error ? (
                <span className="form-feedback__error">{error}</span>
              ) : (
                <span className="form-feedback__hint">
                  Поддерживается UUID сессии, который возвращает сервер.
                </span>
              )}
            </div>

            <div className="actions">
              <button type="submit" className="button button--primary" disabled={isLoading}>
                {isLoading ? "Подключение..." : "Подключиться"}
              </button>
              <button
                type="button"
                className="button button--secondary"
                onClick={onCreateSession}
                disabled={isLoading}
              >
                Создать комнату
              </button>
              <button
                type="button"
                className="button button--ghost"
                onClick={onBack}
                disabled={isLoading}
              >
                К авторизации
              </button>
            </div>
          </form>
        </div>
      </main>

      <aside className="inspector-card">
        <span className="eyebrow">Статус</span>
        <h3>Готов к подключению</h3>
        <p>
          selfcord пока работает с одной активной голосовой сессией. Список
          участников появится, когда backend начнет отдавать эти данные.
        </p>
      </aside>
    </section>
  );
}

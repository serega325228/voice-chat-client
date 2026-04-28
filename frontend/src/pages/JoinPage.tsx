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
    <section className="panel panel--join">
      <div className="panel__card panel__card--wide">
        <div className="card-header">
          <span className="eyebrow">Сессии</span>
          <h2>Подключение к комнате</h2>
          <p>Введи ID существующей сессии или создай новую комнату.</p>
        </div>

        <form className="stack" onSubmit={handleSubmit}>
          <label className="field">
            <span>ID сессии</span>
            <input
              type="text"
              value={sessionId}
              onChange={(event) => setSessionId(event.target.value)}
              placeholder="VOICE-2419"
              autoComplete="off"
            />
          </label>

          <div className="form-feedback" aria-live="polite">
            {error ? (
              <span className="form-feedback__error">{error}</span>
            ) : (
              <span className="form-feedback__hint">Сессия создается и ищется через backend.</span>
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
              Создать сессию
            </button>
            <button
              type="button"
              className="button button--ghost"
              onClick={onBack}
              disabled={isLoading}
            >
              Назад
            </button>
          </div>
        </form>
      </div>
    </section>
  );
}

import { ControlsBar } from "../components/ControlsBar";
import { ParticipantItem } from "../components/ParticipantItem";
import type { Participant } from "../types/app";

interface CallPageProps {
  sessionId: string;
  callStatus: string;
  callInfo: string | null;
  participants: Participant[];
  isMuted: boolean;
  isLoading: boolean;
  onToggleMute: () => void;
  onLeaveCall: () => void;
}

export function CallPage({
  sessionId,
  callStatus,
  callInfo,
  participants,
  isMuted,
  isLoading,
  onToggleMute,
  onLeaveCall,
}: CallPageProps) {
  return (
    <section className="workspace workspace--call">
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
              # текущая-сессия
            </button>
            <button type="button" className="channel-item" disabled>
              # lobby
            </button>
            <button type="button" className="channel-item" disabled>
              # direct
            </button>
          </div>
        </div>
      </aside>

      <main className="call-layout">
        <header className="session-banner">
          <div>
            <span className="eyebrow">Текущая комната</span>
            <h1>{sessionId}</h1>
          </div>
          <div className="session-stats">
            <div>
              <strong>{callStatus}</strong>
              <span>Состояние</span>
            </div>
            <div>
              <strong>{isMuted ? "Выкл." : "Вкл."}</strong>
              <span>Микрофон</span>
            </div>
          </div>
        </header>

        <section className="participants-panel">
          <div className="participants-panel__header">
            <div>
              <span className="eyebrow">Voice</span>
              <h2>Участники</h2>
            </div>
            <p>
              {callInfo ??
                "API не отдает список участников комнаты, поэтому клиент показывает локальное состояние звонка и автоматически обрабатывает удаленное аудио."}
            </p>
          </div>

          <div className="participants-grid">
            {participants.map((participant) => (
              <ParticipantItem key={participant.id} participant={participant} />
            ))}
          </div>
        </section>

        <ControlsBar
          isMuted={isMuted}
          isLoading={isLoading}
          onToggleMute={onToggleMute}
          onLeaveCall={onLeaveCall}
        />
      </main>

      <aside className="inspector-card">
        <span className="eyebrow">Session</span>
        <h3>Live details</h3>
        <p>Комната активна, аудиоканал управляется локальным клиентом.</p>
        <div className="mini-stat">
          <span>ID</span>
          <strong>{sessionId}</strong>
        </div>
      </aside>
    </section>
  );
}

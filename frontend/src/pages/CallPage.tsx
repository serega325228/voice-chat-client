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
    <section className="call-layout">
      <header className="session-banner">
        <div>
          <span className="eyebrow">Текущая сессия</span>
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

      <main className="participants-panel">
        <div className="participants-panel__header">
          <h2>Состояние звонка</h2>
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
      </main>

      <ControlsBar
        isMuted={isMuted}
        isLoading={isLoading}
        onToggleMute={onToggleMute}
        onLeaveCall={onLeaveCall}
      />
    </section>
  );
}

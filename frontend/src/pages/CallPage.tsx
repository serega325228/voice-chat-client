import { ControlsBar } from "../components/ControlsBar";
import { ParticipantItem } from "../components/ParticipantItem";
import type { Participant } from "../types/app";

interface CallPageProps {
  sessionId: string;
  participants: Participant[];
  isMuted: boolean;
  isLoading: boolean;
  onToggleMute: () => void;
  onLeaveCall: () => void;
}

export function CallPage({
  sessionId,
  participants,
  isMuted,
  isLoading,
  onToggleMute,
  onLeaveCall,
}: CallPageProps) {
  const speakingCount = participants.filter((participant) => participant.isSpeaking).length;

  return (
    <section className="call-layout">
      <header className="session-banner">
        <div>
          <span className="eyebrow">Текущая сессия</span>
          <h1>{sessionId}</h1>
        </div>
        <div className="session-stats">
          <div>
            <strong>{participants.length}</strong>
            <span>Участников</span>
          </div>
          <div>
            <strong>{speakingCount}</strong>
            <span>Говорят сейчас</span>
          </div>
        </div>
      </header>

      <main className="participants-panel">
        <div className="participants-panel__header">
          <h2>Участники комнаты</h2>
          <p>Сессия создана через backend. Голосовой обмен будет подключен отдельным шагом.</p>
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

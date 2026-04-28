import type { Participant } from "../types/app";

interface ParticipantItemProps {
  participant: Participant;
}

export function ParticipantItem({ participant }: ParticipantItemProps) {
  return (
    <article
      className={`participant-item${participant.isSpeaking ? " participant-item--speaking" : ""}`}
    >
      <div className="participant-item__identity">
        <span
          className={`participant-item__signal${participant.isSpeaking ? " participant-item__signal--active" : ""}`}
          aria-hidden="true"
        />
        <div>
          <h3>{participant.name}</h3>
          <p>{participant.isSpeaking ? "Говорит" : "На связи"}</p>
        </div>
      </div>

      <div className="participant-item__meta">
        <span
          className={`status-pill${participant.isMuted ? " status-pill--muted" : " status-pill--live"}`}
        >
          {participant.isMuted ? "Микрофон выкл." : "Микрофон вкл."}
        </span>
      </div>
    </article>
  );
}

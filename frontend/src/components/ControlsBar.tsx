interface ControlsBarProps {
  isMuted: boolean;
  isLoading: boolean;
  onToggleMute: () => void;
  onLeaveCall: () => void;
}

export function ControlsBar({
  isMuted,
  isLoading,
  onToggleMute,
  onLeaveCall,
}: ControlsBarProps) {
  return (
    <div className="controls-bar">
      <button
        type="button"
        className="button button--secondary"
        onClick={onToggleMute}
        disabled={isLoading}
      >
        {isMuted ? "Включить микрофон" : "Выключить микрофон"}
      </button>

      <button
        type="button"
        className="button button--danger"
        onClick={onLeaveCall}
        disabled={isLoading}
      >
        Покинуть сессию
      </button>
    </div>
  );
}

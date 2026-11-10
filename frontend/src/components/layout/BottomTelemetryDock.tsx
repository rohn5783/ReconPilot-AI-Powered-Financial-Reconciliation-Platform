import React, { useState } from 'react';
import { useWorkspaceStore } from '../../store/useWorkspaceStore';
import { XtermConsole } from '../terminal/XtermConsole';
import { ChevronUp, ChevronDown, Minimize2, Maximize2, X } from 'lucide-react';

interface BottomTelemetryDockProps {
  scanId: string | null;
  dryRun?: boolean;
}

export const BottomTelemetryDock: React.FC<BottomTelemetryDockProps> = ({ scanId, dryRun = true }) => {
  const { isDockOpen, toggleDock, dockHeight, setDockHeight } = useWorkspaceStore();
  const [isMaximized, setIsMaximized] = useState(false);

  if (!isDockOpen) return null;

  const currentHeight = isMaximized ? 540 : dockHeight;

  return (
    <div
      style={{ height: `${currentHeight}px` }}
      className="w-full shrink-0 flex flex-col bg-carbon-canvas hairline-t relative z-20 transition-all duration-150 ease-out"
    >
      {/* Drag & Resize Handle */}
      <div
        className="h-1 bg-carbon-border hover:bg-signal-accent/50 cursor-ns-resize transition-colors w-full"
        onMouseDown={(e) => {
          e.preventDefault();
          const startY = e.clientY;
          const startHeight = dockHeight;

          const onMouseMove = (moveEvent: MouseEvent) => {
            const deltaY = startY - moveEvent.clientY;
            const newHeight = Math.max(160, Math.min(650, startHeight + deltaY));
            setDockHeight(newHeight);
          };

          const onMouseUp = () => {
            window.removeEventListener('mousemove', onMouseMove);
            window.removeEventListener('mouseup', onMouseUp);
          };

          window.addEventListener('mousemove', onMouseMove);
          window.addEventListener('mouseup', onMouseUp);
        }}
      />

      {/* Dock Window Controls */}
      <div className="absolute right-3 top-2 z-30 flex items-center gap-1.5">
        <button
          onClick={() => setIsMaximized(!isMaximized)}
          className="p-1 rounded-micro border border-carbon-border bg-carbon-card hover:bg-carbon-hover text-typography-muted hover:text-typography-primary transition-colors"
          title={isMaximized ? 'Restore Height' : 'Maximize Terminal'}
        >
          {isMaximized ? <Minimize2 className="w-3 h-3" /> : <Maximize2 className="w-3 h-3" />}
        </button>

        <button
          onClick={toggleDock}
          className="p-1 rounded-micro border border-carbon-border bg-carbon-card hover:bg-carbon-hover text-typography-muted hover:text-typography-primary transition-colors"
          title="Close Terminal Dock"
        >
          <X className="w-3 h-3" />
        </button>
      </div>

      {/* Embedded Xterm Console */}
      <div className="flex-1 overflow-hidden">
        <XtermConsole scanId={scanId} dryRun={dryRun} />
      </div>
    </div>
  );
};


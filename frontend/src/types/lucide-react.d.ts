declare module 'lucide-react' {
  import * as React from 'react';

  export interface LucideProps extends React.SVGAttributes<SVGElement> {
    size?: string | number;
    color?: string;
    strokeWidth?: string | number;
    className?: string;
  }

  export type LucideIcon = React.FC<LucideProps>;

  export const CheckCircle2: LucideIcon;
  export const Clock: LucideIcon;
  export const PlayCircle: LucideIcon;
  export const XCircle: LucideIcon;
  export const ArrowRight: LucideIcon;
  export const Activity: LucideIcon;
  export const ShieldAlert: LucideIcon;
  export const Cpu: LucideIcon;
  export const ChevronUp: LucideIcon;
  export const ChevronDown: LucideIcon;
  export const ChevronRight: LucideIcon;
  export const Minimize2: LucideIcon;
  export const Maximize2: LucideIcon;
  export const X: LucideIcon;
  export const Play: LucideIcon;
  export const Shield: LucideIcon;
  export const Sliders: LucideIcon;
  export const Zap: LucideIcon;
  export const Terminal: LucideIcon;
  export const Trash2: LucideIcon;
  export const ArrowDownCircle: LucideIcon;
  export const Copy: LucideIcon;
  export const Check: LucideIcon;
  export const Download: LucideIcon;
  export const RefreshCw: LucideIcon;
  export const AlertTriangle: LucideIcon;
  export const ArrowUpRight: LucideIcon;
  export const ShieldCheck: LucideIcon;
  export const FileText: LucideIcon;
  export const Layers: LucideIcon;
  export const Sparkles: LucideIcon;
  export const Globe: LucideIcon;
  export const Server: LucideIcon;
  export const ArrowUpDown: LucideIcon;
  export const ExternalLink: LucideIcon;
  export const PlusCircle: LucideIcon;
  export const AlertCircle: LucideIcon;
  export const Code: LucideIcon;
  export const Plus: LucideIcon;
  export const Network: LucideIcon;
  export const Bot: LucideIcon;
  export const Database: LucideIcon;
  export const Radio: LucideIcon;
  export const Search: LucideIcon;
  export const Command: LucideIcon;
}


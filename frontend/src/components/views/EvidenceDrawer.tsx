import React from 'react';
import { Asset, Finding } from '../../types';
import { X, Copy, Check, ShieldAlert, Globe, Server, Code, FileText } from 'lucide-react';
import { useWorkspaceStore } from '../../store/useWorkspaceStore';

interface EvidenceDrawerProps {
  asset: Asset | null;
  finding: Finding | null;
  onClose: () => void;
}

export const EvidenceDrawer: React.FC<EvidenceDrawerProps> = ({ asset, finding, onClose }) => {
  const { activeInspectorTab, setActiveInspectorTab } = useWorkspaceStore();
  const [copied, setCopied] = React.useState(false);

  if (!asset && !finding) return null;

  const targetTitle = finding ? finding.title : asset?.fqdn;
  const targetSubtitle = finding ? `Finding on ${finding.fqdn}` : `Asset Profile (${asset?.ip_addresses?.join(', ') || 'Unresolved'})`;

  const copyToClipboard = (text: string) => {
    navigator.clipboard.writeText(text);
    setCopied(true);
    setTimeout(() => setCopied(false), 1500);
  };

  return (
    <div className="fixed inset-y-0 right-0 z-40 w-full max-w-xl bg-carbon-panel border-l border-carbon-border shadow-2xl flex flex-col select-text">
      {/* Drawer Header */}
      <div className="flex items-center justify-between p-4 hairline-b bg-carbon-card/50">
        <div className="flex flex-col">
          <div className="flex items-center gap-2">
            {finding ? (
              <ShieldAlert className="w-4 h-4 text-signal-critical" />
            ) : (
              <Globe className="w-4 h-4 text-signal-accent" />
            )}
            <span className="font-mono font-bold text-xs text-typography-primary tracking-tight">
              {targetTitle}
            </span>
          </div>
          <span className="text-[11px] font-mono text-typography-muted">
            {targetSubtitle}
          </span>
        </div>

        <button
          onClick={onClose}
          className="p-1 rounded-micro border border-carbon-border bg-carbon-card hover:bg-carbon-hover text-typography-muted hover:text-typography-primary transition-colors"
        >
          <X className="w-4 h-4" />
        </button>
      </div>

      {/* Tab Switcher */}
      <div className="flex items-center px-4 hairline-b bg-carbon-card/30 text-xs font-mono">
        {asset?.response_headers && (
          <button
            onClick={() => setActiveInspectorTab('headers')}
            className={`py-2 px-3 border-b-2 transition-colors flex items-center gap-1.5 ${
              activeInspectorTab === 'headers'
                ? 'border-signal-accent text-signal-accent font-semibold'
                : 'border-transparent text-typography-muted hover:text-typography-primary'
            }`}
          >
            <Server className="w-3.5 h-3.5" />
            <span>HTTP Headers</span>
          </button>
        )}

        {finding && (
          <button
            onClick={() => setActiveInspectorTab('evidence')}
            className={`py-2 px-3 border-b-2 transition-colors flex items-center gap-1.5 ${
              activeInspectorTab === 'evidence'
                ? 'border-signal-accent text-signal-accent font-semibold'
                : 'border-transparent text-typography-muted hover:text-typography-primary'
            }`}
          >
            <Code className="w-3.5 h-3.5" />
            <span>Raw Evidence & Matcher</span>
          </button>
        )}

        {finding?.remediation && (
          <button
            onClick={() => setActiveInspectorTab('remediation')}
            className={`py-2 px-3 border-b-2 transition-colors flex items-center gap-1.5 ${
              activeInspectorTab === 'remediation'
                ? 'border-signal-accent text-signal-accent font-semibold'
                : 'border-transparent text-typography-muted hover:text-typography-primary'
            }`}
          >
            <FileText className="w-3.5 h-3.5" />
            <span>Remediation Advice</span>
          </button>
        )}

        {asset && asset.open_ports?.length > 0 && (
          <button
            onClick={() => setActiveInspectorTab('ports')}
            className={`py-2 px-3 border-b-2 transition-colors flex items-center gap-1.5 ${
              activeInspectorTab === 'ports'
                ? 'border-signal-accent text-signal-accent font-semibold'
                : 'border-transparent text-typography-muted hover:text-typography-primary'
            }`}
          >
            <Server className="w-3.5 h-3.5" />
            <span>Service Ports ({asset.open_ports.length})</span>
          </button>
        )}
      </div>

      {/* Drawer Content */}
      <div className="flex-1 overflow-y-auto p-4 font-mono text-xs space-y-4">
        {/* HEADERS TAB */}
        {activeInspectorTab === 'headers' && asset?.response_headers && (
          <div className="space-y-2">
            <div className="flex items-center justify-between">
              <span className="text-[11px] text-typography-muted uppercase tracking-wider">
                Raw HTTP Response Headers
              </span>
              <button
                onClick={() => copyToClipboard(JSON.stringify(asset.response_headers, null, 2))}
                className="text-[10px] text-typography-muted hover:text-typography-primary flex items-center gap-1"
              >
                {copied ? <Check className="w-3 h-3 text-signal-nominal" /> : <Copy className="w-3 h-3" />}
                <span>{copied ? 'Copied' : 'Copy Headers'}</span>
              </button>
            </div>

            <div className="bg-carbon-canvas border border-carbon-border rounded-micro p-3 space-y-1 overflow-x-auto text-[11px] leading-relaxed">
              {Object.entries(asset.response_headers).map(([k, v]) => (
                <div key={k} className="flex">
                  <span className="text-signal-accent font-medium select-none w-44 shrink-0">{k}:</span>
                  <span className="text-typography-primary break-all">{v}</span>
                </div>
              ))}
            </div>
          </div>
        )}

        {/* EVIDENCE TAB */}
        {activeInspectorTab === 'evidence' && finding && (
          <div className="space-y-4">
            {/* Matched URL & Curl command */}
            <div className="space-y-1.5">
              <span className="text-[11px] text-typography-muted uppercase tracking-wider">Matched Target</span>
              <div className="bg-carbon-canvas border border-carbon-border rounded-micro p-2 text-typography-primary break-all">
                {finding.raw_evidence.matched_at}
              </div>
            </div>

            {/* Request Snippet */}
            {finding.raw_evidence.request && (
              <div className="space-y-1.5">
                <div className="flex items-center justify-between">
                  <span className="text-[11px] text-typography-muted uppercase tracking-wider">Audit Request</span>
                  <button
                    onClick={() => copyToClipboard(finding.raw_evidence.request || '')}
                    className="text-[10px] text-typography-muted hover:text-typography-primary flex items-center gap-1"
                  >
                    <Copy className="w-3 h-3" />
                    <span>Copy</span>
                  </button>
                </div>
                <pre className="bg-carbon-canvas border border-carbon-border rounded-micro p-3 text-[11px] text-typography-secondary overflow-x-auto whitespace-pre">
                  {finding.raw_evidence.request}
                </pre>
              </div>
            )}

            {/* Response Snippet */}
            {finding.raw_evidence.response && (
              <div className="space-y-1.5">
                <div className="flex items-center justify-between">
                  <span className="text-[11px] text-typography-muted uppercase tracking-wider">Match Response Evidence</span>
                  <button
                    onClick={() => copyToClipboard(finding.raw_evidence.response || '')}
                    className="text-[10px] text-typography-muted hover:text-typography-primary flex items-center gap-1"
                  >
                    <Copy className="w-3 h-3" />
                    <span>Copy</span>
                  </button>
                </div>
                <pre className="bg-carbon-canvas border border-carbon-border rounded-micro p-3 text-[11px] text-signal-nominal overflow-x-auto whitespace-pre">
                  {finding.raw_evidence.response}
                </pre>
              </div>
            )}
          </div>
        )}

        {/* REMEDIATION TAB */}
        {activeInspectorTab === 'remediation' && finding && (
          <div className="space-y-3">
            <span className="text-[11px] text-typography-muted uppercase tracking-wider">
              Recommended Remediation Strategy
            </span>
            <div className="bg-carbon-canvas border border-carbon-border rounded-micro p-4 text-typography-primary leading-relaxed text-xs">
              {finding.remediation}
            </div>
          </div>
        )}

        {/* PORTS TAB */}
        {activeInspectorTab === 'ports' && asset && (
          <div className="space-y-2">
            <span className="text-[11px] text-typography-muted uppercase tracking-wider">
              Active Listening TCP Services
            </span>
            <div className="grid grid-cols-2 gap-2">
              {asset.open_ports.map((port) => (
                <div
                  key={port}
                  className="bg-carbon-canvas border border-carbon-border rounded-micro p-2.5 flex items-center justify-between"
                >
                  <span className="font-bold text-typography-primary">{port}/tcp</span>
                  <span className="text-[10px] text-signal-accent uppercase font-mono">
                    {port === 443 ? 'HTTPS' : port === 80 ? 'HTTP' : port === 6379 ? 'REDIS' : port === 22 ? 'SSH' : 'SERVICE'}
                  </span>
                </div>
              ))}
            </div>
          </div>
        )}
      </div>
    </div>
  );
};

